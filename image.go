// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package rez provides image resizing for go.

Featuring:
 - YCbCr resizes
 - YCbCr Chroma subsample ratio conversions
 - RGB resizes
 - Optional interlaced-aware resizes
 - Parallel resizes
 - SSE2 optimisations on AMD64

The easiest way to use it is:

    err := Convert(output, input, NewBicubicFilter())

However, if you plan to convert video, where resize parameters are the same for
multiple images, the best way is:

    cfg, err := PrepareConversion(output, input)
    converter, err := NewConverter(cfg, NewBicubicFilter())
    for i := 0; i < N; i++ {
        err := converter.Convert(output[i], input[i])
    }

Note that by default, images are resized in parallel with GOMAXPROCS slices.
Best performance is obtained when GOMAXPROCS is at least equal to your CPU
count.
*/
package rez

import (
	"fmt"
	"image"
	"runtime"
	"sync"
)

// Converter is an interface that implements conversion between images
// It is currently able to convert only between ycbcr images
type Converter interface {
	// Converts one image into another, applying any necessary colorspace
	// conversion and/or resizing
	// dst = destination image
	// src = source image
	// Result is undefined if src points to the same data as dst
	// Returns an error if the conversion fails
	Convert(dst, src image.Image) error
}

// ChromaRatio is a chroma subsampling ratio
type ChromaRatio int

const (
	// Ratio411 is 4:1:1
	Ratio411 ChromaRatio = iota
	// Ratio420 is 4:2:0
	Ratio420
	// Ratio422 is 4:2:2
	Ratio422
	// Ratio440 is 4:4:0
	Ratio440
	// Ratio444 is 4:4:4
	Ratio444
)

// Descriptor describes an image properties
type Descriptor struct {
	Width      int         // width in pixels
	Height     int         // height in pixels
	Ratio      ChromaRatio // chroma ratio
	Pack       int         // pixels per pack
	Interlaced bool        // progressive or interlaced
	Planes     int         // number of planes
}

// Check returns whether the descriptor is valid
func (d *Descriptor) Check() error {
	w := 1
	h := 1
	switch d.Ratio {
	case Ratio411:
		w = 4
	case Ratio420:
		w = 2
		h = 2
	case Ratio422:
		w = 2
	case Ratio440:
		h = 2
	case Ratio444:
	default:
		return fmt.Errorf("invalid ratio %v", d.Ratio)
	}
	if d.Interlaced {
		h *= 2
	}
	if d.Pack < 1 || d.Pack > 4 {
		return fmt.Errorf("invalid pack value %v", d.Pack)
	}
	if d.Width%w != 0 {
		return fmt.Errorf("width must be mod %v", w)
	}
	if d.Height%h != 0 {
		return fmt.Errorf("height must be mod %v", h)
	}
	return nil
}

// GetWidth returns the width in pixels for the input plane
func (d *Descriptor) GetWidth(plane int) int {
	if plane < 0 || plane+1 > maxPlanes {
		panic(fmt.Errorf("invalid plane %v", plane))
	}
	if plane == 0 {
		return d.Width
	}
	switch d.Ratio {
	case Ratio411:
		return d.Width >> 2
	case Ratio420, Ratio422:
		return d.Width >> 1
	case Ratio440, Ratio444:
		return d.Width
	}
	panic(fmt.Errorf("invalid ratio %v", d.Ratio))
}

// GetHeight returns the height in pixels for the input plane
func (d *Descriptor) GetHeight(plane int) int {
	if plane < 0 || plane+1 > maxPlanes {
		panic(fmt.Errorf("invalid plane %v", plane))
	}
	if plane == 0 {
		return d.Height
	}
	switch d.Ratio {
	case Ratio411, Ratio422, Ratio444:
		return d.Height
	case Ratio420, Ratio440:
		return d.Height >> 1
	}
	panic(fmt.Errorf("invalid ratio %v", d.Ratio))
}

// ConverterConfig is a configuration used with NewConverter
type ConverterConfig struct {
	Input   Descriptor // input description
	Output  Descriptor // output description
	Threads int        // number of allowed "threads"
}

const (
	maxPlanes = 3
)

// Plane describes a single image plane
type Plane struct {
	Data   []byte // plane buffer
	Width  int    // width in pixels
	Height int    // height in pixels
	Pitch  int    // pitch in bytes
	Pack   int    // pixels per pack
}

type converterContext struct {
	ConverterConfig
	wrez   [maxPlanes]Resizer
	hrez   [maxPlanes]Resizer
	buffer [maxPlanes]*Plane
}

func toInterlacedString(interlaced bool) string {
	if interlaced {
		return "interlaced"
	}
	return "progressive"
}

func toPackedString(pack int) string {
	return fmt.Sprintf("%v-packed", pack)
}

func align(value, align int) int {
	return (value + align - 1) & -align
}

func checkConversion(dst, src *Descriptor) error {
	if err := src.Check(); err != nil {
		return fmt.Errorf("invalid input format: %v", err)
	}
	if err := dst.Check(); err != nil {
		return fmt.Errorf("invalid output format: %v", err)
	}
	if src.Interlaced != dst.Interlaced {
		return fmt.Errorf("unable to convert %v input to %v output",
			toInterlacedString(src.Interlaced),
			toInterlacedString(dst.Interlaced))
	}
	if src.Pack != dst.Pack {
		return fmt.Errorf("unable to convert %v input to %v output",
			toPackedString(src.Pack),
			toPackedString(dst.Pack))
	}
	if src.Planes != dst.Planes {
		return fmt.Errorf("unable to convert %v planes to %v planes",
			src.Planes, dst.Planes)
	}
	return nil
}

// NewConverter returns a Converter interface
// cfg = converter configuration
// filter = filter used for resizing
// Returns an error if the conversion is invalid or not implemented
func NewConverter(cfg *ConverterConfig, filter Filter) (Converter, error) {
	err := checkConversion(&cfg.Output, &cfg.Input)
	if err != nil {
		return nil, err
	}
	if cfg.Threads == 0 {
		cfg.Threads = runtime.GOMAXPROCS(0)
	}
	ctx := &converterContext{
		ConverterConfig: *cfg,
	}
	size := 0
	group := sync.WaitGroup{}
	for i := 0; i < cfg.Output.Planes; i++ {
		win := cfg.Input.GetWidth(i)
		hin := cfg.Input.GetHeight(i)
		wout := cfg.Output.GetWidth(i)
		hout := cfg.Output.GetHeight(i)
		if win != wout {
			group.Add(1)
			go func(i int) {
				defer group.Done()
				ctx.wrez[i] = NewResize(&ResizerConfig{
					Depth:      8,
					Input:      win,
					Output:     wout,
					Vertical:   false,
					Interlaced: false,
					Pack:       cfg.Input.Pack,
					Threads:    cfg.Threads,
				}, filter)
			}(i)
		}
		if hin != hout {
			group.Add(1)
			go func(i int) {
				defer group.Done()
				ctx.hrez[i] = NewResize(&ResizerConfig{
					Depth:      8,
					Input:      hin,
					Output:     hout,
					Vertical:   true,
					Interlaced: cfg.Output.Interlaced,
					Pack:       cfg.Output.Pack,
					Threads:    cfg.Threads,
				}, filter)
			}(i)
		}
		if win != wout && hin != hout {
			p := &Plane{
				Width:  win,
				Height: hout,
				Pitch:  align(win*cfg.Input.Pack, 16),
				Pack:   cfg.Input.Pack,
			}
			size += p.Pitch * p.Height
			ctx.buffer[i] = p
		}
	}
	if size != 0 {
		buffer := make([]byte, size)
		idx := 0
		for i := 0; i < cfg.Output.Planes; i++ {
			if p := ctx.buffer[i]; p != nil {
				size := p.Pitch*(p.Height-1) + p.Width*p.Pack
				p.Data = buffer[idx : idx+size]
				idx += p.Pitch * p.Height
			}
		}
	}
	group.Wait()
	return ctx, nil
}

// GetRatio returns a ChromaRatio from an image.YCbCrSubsampleRatio
func GetRatio(value image.YCbCrSubsampleRatio) ChromaRatio {
	switch value {
	case image.YCbCrSubsampleRatio420:
		return Ratio420
	case image.YCbCrSubsampleRatio422:
		return Ratio422
	case image.YCbCrSubsampleRatio440:
		return Ratio440
	case image.YCbCrSubsampleRatio444:
		return Ratio444
	}
	return Ratio444
}

func inspect(data image.Image, interlaced bool) (*Descriptor, []Plane, error) {
	switch t := data.(type) {
	case *image.YCbCr:
		d, p := inspectYuv(t, interlaced)
		return d, p, nil
	case *image.RGBA:
		d, p := inspectRgb(t, interlaced)
		return d, p, nil
	}
	return nil, nil, fmt.Errorf("unknown image format")
}

func getYuvDescriptor(yuv *image.YCbCr, interlaced bool) Descriptor {
	return Descriptor{
		Width:      yuv.Rect.Dx(),
		Height:     yuv.Rect.Dy(),
		Ratio:      GetRatio(yuv.SubsampleRatio),
		Interlaced: interlaced,
		Pack:       1,
		Planes:     3,
	}
}

func getRgbDescriptor(rgb *image.RGBA, interlaced bool) Descriptor {
	return Descriptor{
		Width:      rgb.Rect.Dx(),
		Height:     rgb.Rect.Dy(),
		Ratio:      Ratio444,
		Interlaced: interlaced,
		Pack:       4,
		Planes:     1,
	}
}

func getYuvPlanes(yuv *image.YCbCr, d *Descriptor) []Plane {
	planes := []Plane{}
	for i := 0; i < maxPlanes; i++ {
		p := Plane{
			Width:  d.GetWidth(i),
			Height: d.GetHeight(i),
			Pack:   d.Pack,
		}
		switch i {
		case 0:
			p.Pitch = yuv.YStride
			p.Data = yuv.Y[yuv.YOffset(0, 0) : p.Pitch*(p.Height-1)+p.Width]
		case 1:
			p.Pitch = yuv.CStride
			p.Data = yuv.Cb[yuv.COffset(0, 0) : p.Pitch*(p.Height-1)+p.Width]
		case 2:
			p.Pitch = yuv.CStride
			p.Data = yuv.Cr[yuv.COffset(0, 0) : p.Pitch*(p.Height-1)+p.Width]
		}
		planes = append(planes, p)
	}
	return planes
}

func getRgbPlane(rgb *image.RGBA, d *Descriptor) []Plane {
	p := Plane{
		Width:  d.GetWidth(0),
		Height: d.GetHeight(0),
		Pack:   d.Pack,
		Pitch:  rgb.Stride,
	}
	p.Data = rgb.Pix[rgb.PixOffset(0, 0) : p.Pitch*(p.Height-1)+p.Width*p.Pack]
	return []Plane{p}
}

func inspectYuv(yuv *image.YCbCr, interlaced bool) (*Descriptor, []Plane) {
	d := getYuvDescriptor(yuv, interlaced)
	return &d, getYuvPlanes(yuv, &d)
}

func inspectRgb(rgb *image.RGBA, interlaced bool) (*Descriptor, []Plane) {
	d := getRgbDescriptor(rgb, interlaced)
	return &d, getRgbPlane(rgb, &d)
}

func resizePlane(group *sync.WaitGroup, dst, src, buf *Plane, hrez, wrez Resizer) {
	defer group.Done()
	hdst := dst
	wsrc := src
	if hrez != nil && wrez != nil {
		hdst = buf
		wsrc = buf
	}
	if hrez != nil {
		hrez.Resize(hdst.Data, src.Data, src.Width, src.Height, hdst.Pitch, src.Pitch)
	}
	if wrez != nil {
		wrez.Resize(dst.Data, wsrc.Data, wsrc.Width, wsrc.Height, dst.Pitch, wsrc.Pitch)
	}
	if hrez == nil && wrez == nil {
		copyPlane(dst.Data, src.Data, src.Width*src.Pack, src.Height, dst.Pitch, src.Pitch)
	}
}

func (ctx *converterContext) Convert(output, input image.Image) error {
	id, src, err := inspect(input, ctx.Input.Interlaced)
	if err != nil {
		return err
	}
	od, dst, err := inspect(output, ctx.Output.Interlaced)
	if err != nil {
		return err
	}
	err = checkConversion(od, id)
	if err != nil {
		return err
	}
	group := sync.WaitGroup{}
	for i := 0; i < ctx.Input.Planes; i++ {
		group.Add(1)
		go resizePlane(&group, &dst[i], &src[i], ctx.buffer[i], ctx.hrez[i], ctx.wrez[i])
	}
	group.Wait()
	return nil
}

// PrepareConversion returns a ConverterConfig properly set for a conversion
// from input images to output images
// Returns an error if the conversion is not possible
func PrepareConversion(output, input image.Image) (*ConverterConfig, error) {
	src, _, err := inspect(input, false)
	if err != nil {
		return nil, err
	}
	dst, _, err := inspect(output, false)
	if err != nil {
		return nil, err
	}
	err = checkConversion(dst, src)
	if err != nil {
		return nil, err
	}
	return &ConverterConfig{
		Input:  *src,
		Output: *dst,
	}, nil
}

// Convert converts an input image into output, applying any color conversion
// and/or resizing, using the input filter for interpolation.
// Note that if you plan to do the same conversion over and over, it is faster
// to use a Converter interface
func Convert(output, input image.Image, filter Filter) error {
	cfg, err := PrepareConversion(output, input)
	if err != nil {
		return err
	}
	converter, err := NewConverter(cfg, filter)
	if err != nil {
		return err
	}
	return converter.Convert(output, input)
}

// Psnr computes the PSNR between two input images
// Only ycbcr is currently supported
func Psnr(a, b image.Image) ([]float64, error) {
	psnrs := []float64{}
	id, src, err := inspect(a, false)
	if err != nil {
		return nil, err
	}
	od, dst, err := inspect(b, false)
	if err != nil {
		return nil, err
	}
	if *id != *od {
		return nil, fmt.Errorf("unable to psnr different formats")
	}
	for i := 0; i < len(dst); i++ {
		psnrs = append(psnrs, psnrPlane(src[i].Data, dst[i].Data, src[i].Width*src[i].Pack, src[i].Height, src[i].Pitch, dst[i].Pitch))
	}
	return psnrs, nil
}
