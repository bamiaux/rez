// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

type Config struct {
	Depth      int
	Input      int
	Output     int
	Vertical   bool
	Interlaced bool
}

type Resizer interface {
	Resize(dst, src []byte, width, height, dstPitch, srcPitch int)
}

type Scaler func(dst, src []byte, cof []int16, off []int,
	taps, width, height, dstPitch, srcPitch int)

type Context struct {
	cfg     Config
	kernels []Kernel
	scaler  Scaler
}

func NewResize(cfg *Config, filter Filter) Resizer {
	ctx := Context{
		cfg:    *cfg,
		scaler: h8scaleN,
	}
	ctx.cfg.Depth = 8 // only 8-bit for now
	ctx.kernels = []Kernel{makeKernel(&ctx.cfg, filter, 0)}
	if cfg.Vertical {
		ctx.scaler = v8scaleN
		if cfg.Interlaced {
			ctx.kernels = append(ctx.kernels, makeKernel(&ctx.cfg, filter, 1))
		}
	}
	return &ctx
}

func (c *Context) Resize(dst, src []byte, dpitch, spitch int, width, height int) {
	field := bin(c.cfg.Vertical && c.cfg.Interlaced)
	dwidth := c.cfg.Output
	dheight := height
	if c.cfg.Vertical {
		dwidth = width
		dheight = c.cfg.Output >> field
	}
	for i, k := range c.kernels[:1+field] {
		c.scaler(dst[dpitch*i:], src[spitch*i:], k.coeffs, k.offsets,
			k.size, dwidth, dheight, dpitch<<field, spitch<<field)
	}
}
