// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

import (
	"math"
)

type Filter interface {
	Taps() int
	Get(dx float64) float64
}

type bilinear struct{}

func (bilinear) Taps() int { return 1 }

func (bilinear) Get(x float64) float64 {
	if x < 1 {
		return 1 - x
	}
	return 0
}

func NewBilinearFilter() Filter {
	return bilinear{}
}

type bicubic struct {
	a, b, c, d, e, f, g float64
}

func (bicubic) Taps() int {
	return 2
}

func (f *bicubic) Get(x float64) float64 {
	if x < 1 {
		return f.a + x*x*(f.b+x*f.c)
	} else if x < 2 {
		return f.d + x*(f.e+x*(f.f+x*f.g))
	}
	return 0
}

func NewCustomBicubicFilter(b, c float64) Filter {
	f := &bicubic{}
	f.a = 1 - b/3
	f.b = -3 + 2*b + c
	f.c = 2 - 3*b/2 - c
	f.d = 4*b/3 + 4*c
	f.e = -2*b - 8*c
	f.f = b + 5*c
	f.g = -b/6 - c
	return f
}

func NewBicubicFilter() Filter {
	return NewCustomBicubicFilter(0, 0.5)
}

type lanczos struct {
	taps float64
}

func (f lanczos) Taps() int {
	return int(f.taps)
}

func (f lanczos) Get(x float64) float64 {
	if x > f.taps {
		return 0
	} else if x == 0 {
		return 1
	}
	b := x * math.Pi
	c := b / f.taps
	return math.Sin(b) * math.Sin(c) / (b * c)
}

func NewLanczosFilter(taps int) Filter {
	return lanczos{taps: float64(taps)}
}
