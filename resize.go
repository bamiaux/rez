// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

import (
	"sync"
)

// ResizerConfig is a configuration used with NewResizer
type ResizerConfig struct {
	Depth      int  // bits per pixel
	Input      int  // input size in pixels
	Output     int  // output size in pixels
	Vertical   bool // true for vertical resizes
	Interlaced bool // true if input/output is interlaced
	Pack       int  // pixels per pack [default=1]
	Threads    int  // number of threads, [default=0]
}

// Resizer is a interface that implements resizes
type Resizer interface {
	// Resize one plane into another
	// dst, src = destination and source buffer
	// width, height = plane dimensions in pixels
	// dstPitch, srcPitch = destination and source pitchs/strides in bytes
	Resize(dst, src []byte, width, height, dstPitch, srcPitch int)
}

type scaler func(dst, src []byte, cof, off []int16,
	taps, width, height, dstPitch, srcPitch int)

type context struct {
	cfg     ResizerConfig
	kernels []kernel
	scaler  scaler
}

func getHorizontalScaler(taps int) scaler {
	switch taps {
	case 2:
		return h8scale2
	case 4:
		return h8scale4
	case 6:
		return h8scale6
	case 8:
		return h8scale8
	case 10:
		return h8scale10
	case 12:
		return h8scale12
	}
	return h8scaleN
}

func getVerticalScaler(taps int) scaler {
	switch taps {
	case 2:
		return v8scale2
	case 4:
		return v8scale4
	case 6:
		return v8scale6
	case 8:
		return v8scale8
	case 10:
		return v8scale10
	case 12:
		return v8scale12
	}
	return v8scaleN
}

// NewResize returns a new resizer
// cfg = resize configuration
// filter = filter used for computing weights
func NewResize(cfg *ResizerConfig, filter Filter) Resizer {
	ctx := context{
		cfg: *cfg,
	}
	ctx.cfg.Depth = 8 // only 8-bit for now
	if ctx.cfg.Pack < 1 {
		ctx.cfg.Pack = 1
	}
	ctx.kernels = []kernel{makeKernel(&ctx.cfg, filter, 0)}
	ctx.scaler = getHorizontalScaler(ctx.kernels[0].size)
	if cfg.Vertical {
		ctx.scaler = getVerticalScaler(ctx.kernels[0].size)
		if cfg.Interlaced {
			ctx.kernels = append(ctx.kernels, makeKernel(&ctx.cfg, filter, 1))
		}
	}
	return &ctx
}

func scaleSlice(group *sync.WaitGroup, scaler scaler,
	dst, src []byte, cof, off []int16, taps, width, height, dp, sp int) {
	defer group.Done()
	scaler(dst, src, cof, off, taps, width, height, dp, sp)
}

func scaleSlices(group *sync.WaitGroup, scaler scaler,
	vertical bool, threads, taps, width, height, dp, sp int,
	dst, src []byte, cof []int16, cofscale int, off []int16) {
	defer group.Done()
	nh := height / threads
	if nh < 1 {
		nh = 1
	}
	di := 0
	si := 0
	oi := 0
	ci := 0
	for i := 0; i < threads; i++ {
		last := i+1 == threads
		ih := nh
		if last {
			ih = height - nh*(threads-1)
		}
		if ih == 0 {
			continue
		}
		next := width
		if vertical {
			next = ih
		}
		group.Add(1)
		go scaleSlice(group, scaler,
			dst[di:di+dp*(ih-1)+width],
			src[si:],
			cof[ci:ci+next*taps*cofscale],
			off[oi:oi+next],
			taps, width, ih, dp, sp)
		if last {
			break
		}
		di += ih * dp
		if vertical {
			ci += ih * taps * cofscale
			for j := 0; j < ih; j++ {
				si += sp * int(off[oi+j])
			}
			oi += ih
		} else {
			si += sp * ih
		}
	}
}

func (c *context) Resize(dst, src []byte, width, height, dp, sp int) {
	field := bin(c.cfg.Vertical && c.cfg.Interlaced)
	dwidth := c.cfg.Output
	dheight := height
	if c.cfg.Vertical {
		dwidth = width
		dheight = c.cfg.Output >> field
	}
	pk := c.cfg.Pack
	group := sync.WaitGroup{}
	for i, k := range c.kernels[:1+field] {
		group.Add(1)
		go scaleSlices(&group, c.scaler, c.cfg.Vertical, c.cfg.Threads,
			k.size, dwidth*pk, dheight, dp<<field, sp<<field,
			dst[dp*i:], src[sp*i:], k.coeffs, k.cofscale, k.offsets)
	}
	group.Wait()
}
