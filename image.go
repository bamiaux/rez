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

type Converter interface {
	Convert(dst, src image.Image) error
}

type ChromaRatio int

const (
	Ratio411 ChromaRatio = iota
	Ratio420
	Ratio422
	Ratio440
	Ratio444
)

type Descriptor struct {
	Width      int
	Height     int
	Ratio      ChromaRatio
	Interlaced bool
}

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

type ConverterConfig struct {
	Input   Descriptor
	Output  Descriptor
	Threads int
}

const (
	maxPlanes = 3
)

type Plane struct {
	Data   []byte
	Width  int
	Height int
	Pitch  int
}

type ConverterContext struct {
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
	ctx := &ConverterContext{
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

func (ctx *ConverterContext) Convert(output, input image.Image) error {
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

func Convert(output, input image.Image, filter Filter) error {
	_, src, err := parseYuv(input, false)
	if err != nil {
		return err
	}
	_, dst, err := parseYuv(output, false)
	if err != nil {
		return err
	}
	cfg := ConverterConfig{
		Input:  *src,
		Output: *dst,
	}
	converter, err := NewConverter(&cfg, filter)
	if err != nil {
		return err
	}
	return converter.Convert(output, input)
}
