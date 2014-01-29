// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

import (
	"fmt"
	"image"
)

type Adapter interface {
	Resize(dst, src image.Image) error
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

type AdapterConfig struct {
	Input  Descriptor
	Output Descriptor
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

type AdapterContext struct {
	AdapterConfig
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

func NewAdapter(cfg *AdapterConfig, filter Filter) (Adapter, error) {
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
	ctx := &AdapterContext{
		AdapterConfig: *cfg,
	}
	count := 0
	size := 0
	for i := uint(0); i < maxPlanes; i++ {
		win := cfg.Input.GetWidth(i)
		hin := cfg.Input.GetHeight(i)
		wout := cfg.Output.GetWidth(i)
		hout := cfg.Output.GetHeight(i)
		if win != wout {
			ctx.wrez[i] = NewResize(&Config{
				depth:      8,
				input:      win,
				output:     wout,
				vertical:   false,
				interlaced: false,
			}, filter)
			count++
		}
		if hin != hout {
			ctx.hrez[i] = NewResize(&Config{
				depth:      8,
				input:      hin,
				output:     hout,
				vertical:   true,
				interlaced: cfg.Input.Interlaced,
			}, filter)
			count++
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
		last := buffer
		for i := uint(0); i < maxPlanes; i++ {
			if p := ctx.buffer[i]; p != nil {
				p.Data = last
				last = buffer[p.Pitch*p.Height:]
			}
		}
	}
	if count == 0 {
		return nil, fmt.Errorf("nothing to do")
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

func parse(data image.Image, plane uint, interlaced bool) (*Plane, error) {
	yuv, ok := data.(*image.YCbCr)
	if !ok {
		return nil, fmt.Errorf("unsupported image format")
	}
	d := Descriptor{
		Width:      yuv.Rect.Dx(),
		Height:     yuv.Rect.Dy(),
		Ratio:      GetRatio(yuv.SubsampleRatio),
		Interlaced: interlaced,
	}
	p := &Plane{
		Width:  d.GetWidth(plane),
		Height: d.GetHeight(plane),
	}
	switch plane {
	case 0:
		p.Data = yuv.Y[yuv.YOffset(0, 0):]
		p.Pitch = yuv.YStride
	case 1:
		p.Data = yuv.Cb[yuv.COffset(0, 0):]
		p.Pitch = yuv.CStride
	case 2:
		p.Data = yuv.Cr[yuv.COffset(0, 0):]
		p.Pitch = yuv.CStride
	}
	return p, nil
}

func (ctx *AdapterContext) Resize(output, input image.Image) error {
	for i := uint(0); i < maxPlanes; i++ {
		src, err := parse(input, i, ctx.Input.Interlaced)
		if err != nil {
			return err
		}
		dst, err := parse(output, i, ctx.Output.Interlaced)
		if err != nil {
			return err
		}
		hrez := ctx.hrez[i]
		wrez := ctx.wrez[i]
		hdst := dst
		wsrc := src
		if hrez != nil && wrez != nil {
			hdst = ctx.buffer[i]
			wsrc = ctx.buffer[i]
		}
		if hrez != nil {
			hrez.Resize(hdst.Data, src.Data, hdst.Pitch, src.Pitch, src.Width, src.Height)
		}
		if wrez != nil {
			wrez.Resize(dst.Data, wsrc.Data, dst.Pitch, wsrc.Pitch, wsrc.Width, wsrc.Height)
		}
	}
	return nil
}
