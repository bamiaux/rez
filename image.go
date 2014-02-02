// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

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
	Width      int
	Height     int
	Ratio      ChromaRatio
	Interlaced bool
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
	if d.Width%w != 0 {
		return fmt.Errorf("width must be mod %v", w)
	}
	if d.Height%h != 0 {
		return fmt.Errorf("height must be mod %v", h)
	}
	return nil
}

// GetWidth returns the width in pixels for the input plane
func (d *Descriptor) GetWidth(plane uint) int {
	if plane == 0 {
		return d.Width
	}
	if plane > 2 {
		panic(fmt.Errorf("invalid plane %v", plane))
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
func (d *Descriptor) GetHeight(plane uint) int {
	if plane == 0 {
		return d.Height
	}
	if plane > 2 {
		panic(fmt.Errorf("invalid plane %v", plane))
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

func align(value, align int) int {
	return (value + align - 1) & -align
}

// NewConverter returns a Converter interface
// cfg = converter configuration
// filter = filter used for resizing
// Returns an error if the conversion is invalid or not implemented
func NewConverter(cfg *ConverterConfig, filter Filter) (Converter, error) {
	if err := cfg.Input.Check(); err != nil {
		return nil, err
	}
	if err := cfg.Output.Check(); err != nil {
		return nil, err
	}
	if cfg.Input.Interlaced != cfg.Output.Interlaced {
		return nil, fmt.Errorf("unable to convert %v input to %v output",
			toInterlacedString(cfg.Input.Interlaced),
			toInterlacedString(cfg.Output.Interlaced))
	}
	if cfg.Threads == 0 {
		cfg.Threads = runtime.GOMAXPROCS(0)
	}
	ctx := &converterContext{
		ConverterConfig: *cfg,
	}
	size := 0
	for i := uint(0); i < maxPlanes; i++ {
		win := cfg.Input.GetWidth(i)
		hin := cfg.Input.GetHeight(i)
		wout := cfg.Output.GetWidth(i)
		hout := cfg.Output.GetHeight(i)
		if win != wout {
			ctx.wrez[i] = NewResize(&Config{
				Depth:      8,
				Input:      win,
				Output:     wout,
				Vertical:   false,
				Interlaced: false,
				Threads:    cfg.Threads,
			}, filter)
		}
		if hin != hout {
			ctx.hrez[i] = NewResize(&Config{
				Depth:      8,
				Input:      hin,
				Output:     hout,
				Vertical:   true,
				Interlaced: cfg.Input.Interlaced,
				Threads:    cfg.Threads,
			}, filter)
		}
		if win != wout && hin != hout {
			p := &Plane{
				Width:  win,
				Height: hout,
				Pitch:  align(win, 16),
			}
			size += p.Pitch * p.Height
			ctx.buffer[i] = p
		}
	}
	if size != 0 {
		buffer := make([]byte, size)
		idx := 0
		for i := uint(0); i < maxPlanes; i++ {
			if p := ctx.buffer[i]; p != nil {
				size := p.Pitch*(p.Height-1) + p.Width
				p.Data = buffer[idx : idx+size]
				idx += p.Pitch * p.Height
			}
		}
	}
	return ctx, nil
}

// GetRatio returns a ChromaRation from an image.YCbCrSubsampleRatio
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

func parseYuv(data image.Image, interlaced bool) (*image.YCbCr, *Descriptor, error) {
	yuv, ok := data.(*image.YCbCr)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported image format")
	}
	return yuv, &Descriptor{
		Width:      yuv.Rect.Dx(),
		Height:     yuv.Rect.Dy(),
		Ratio:      GetRatio(yuv.SubsampleRatio),
		Interlaced: interlaced,
	}, nil
}

func parse(data image.Image, plane uint, interlaced bool) (*Plane, error) {
	yuv, d, err := parseYuv(data, interlaced)
	if err != nil {
		return nil, err
	}
	p := &Plane{
		Width:  d.GetWidth(plane),
		Height: d.GetHeight(plane),
	}
	switch plane {
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
	return p, nil
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
		copyPlane(dst.Data, src.Data, src.Width, src.Height, dst.Pitch, src.Pitch)
	}
}

func (ctx *converterContext) Convert(output, input image.Image) error {
	srcs := [maxPlanes]*Plane{}
	dsts := [maxPlanes]*Plane{}
	for i := uint(0); i < maxPlanes; i++ {
		src, err := parse(input, i, ctx.Input.Interlaced)
		if err != nil {
			return err
		}
		dst, err := parse(output, i, ctx.Output.Interlaced)
		if err != nil {
			return err
		}
		srcs[i] = src
		dsts[i] = dst
	}
	group := sync.WaitGroup{}
	for i := uint(0); i < maxPlanes; i++ {
		group.Add(1)
		go resizePlane(&group, dsts[i], srcs[i], ctx.buffer[i], ctx.hrez[i], ctx.wrez[i])
	}
	group.Wait()
	return nil
}

// PrepareConversion returns a ConverterConfig properly set for a conversion
// from input images to output images
// Returns an error if the conversion is not possible
func PrepareConversion(output, input image.Image) (*ConverterConfig, error) {
	_, src, err := parseYuv(input, false)
	if err != nil {
		return nil, err
	}
	_, dst, err := parseYuv(output, false)
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
	for i := uint(0); i < maxPlanes; i++ {
		aplane, err := parse(a, i, false)
		if err != nil {
			return nil, err
		}
		bplane, err := parse(b, i, false)
		if err != nil {
			return nil, err
		}
		if aplane.Width != bplane.Width || aplane.Height != bplane.Height {
			return nil, fmt.Errorf("invalid resolutions %vx%v != %vx%v\n",
				aplane.Width, aplane.Height, bplane.Width, bplane.Height)
		}
		psnrs = append(psnrs, psnrPlane(aplane.Data, bplane.Data, aplane.Width, aplane.Height, aplane.Pitch, bplane.Pitch))
	}
	return psnrs, nil
}
