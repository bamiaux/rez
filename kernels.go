// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

import (
	"math"
	"sort"
)

type Kernel struct {
	coeffs  []int16
	offsets []int
	size    int
}

func bin(v bool) uint {
	if v {
		return 1
	}
	return 0
}

func clip(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func makeDoubleKernel(cfg *Config, filter Filter, field, idx uint) ([]int16, []float64, []float64, int, int) {
	scale := float64(cfg.Output) / float64(cfg.Input)
	step := math.Min(1, scale)
	support := float64(filter.Taps()) / step
	taps := int(math.Ceil(support)) * 2
	offsets := make([]int16, cfg.Output)
	sums := make([]float64, cfg.Output)
	weights := make([]float64, cfg.Output*taps)
	xmid := float64(cfg.Input-cfg.Output) / float64(cfg.Output*2)
	xstep := 1 / scale
	// interlaced resize see only one field but still use full res pixel positions
	ftaps := taps << field
	size := cfg.Output >> field
	step /= float64(1 + field)
	xmid += xstep * float64(field*idx)
	for i := 0; i < size; i++ {
		left := int(math.Ceil(xmid)) - ftaps>>1
		x := clip(left, 0, max(0, cfg.Input-ftaps))
		offsets[i] = int16(x)
		for j := 0; j < ftaps; j++ {
			src := left + j
			if field != 0 && idx^uint(src&1) != 0 {
				continue
			}
			weight := filter.Get(math.Abs(xmid-float64(src)) * step)
			src = clip(src, x, cfg.Input-1) - x
			src >>= field
			weights[i*taps+src] += weight
			sums[i] += weight
		}
		xmid += xstep * float64(1+field)
	}
	return offsets, sums, weights, taps, size
}

type Weight struct {
	weight float64
	offset int
}

type Weights []Weight

func (w Weights) Len() int {
	return len(w)
}

func (w Weights) Less(i, j int) bool {
	return math.Abs(w[j].weight) < math.Abs(w[i].weight)
}

func (w Weights) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

func makeIntegerKernel(taps, size int, weights, sums []float64, pos []int16, field, idx uint) ([]int16, []int) {
	coeffs := make([]int16, taps*size)
	offsets := make([]int, size)
	fweights := make(Weights, taps)
	for i, sum := range sums[:size] {
		for j, w := range weights[:taps] {
			fweights[j].weight = w
			fweights[j].offset = j
		}
		sort.Sort(fweights)
		diff := float64(0)
		scale := 1 << Bits / sum
		for _, it := range fweights {
			w := it.weight*scale + diff
			iw := math.Floor(w + 0.5)
			coeffs[i*taps+it.offset] = int16(iw)
			diff = w - iw
		}
		weights = weights[taps:]
		off := int(pos[i]) + int(field) - int(idx)
		offsets[i] = off >> field
	}
	return coeffs, offsets
}

func makeKernel(cfg *Config, filter Filter, idx uint) Kernel {
	field := bin(cfg.Interlaced)
	pos, sums, weights, taps, size := makeDoubleKernel(cfg, filter, field, idx)
	coeffs, offsets := makeIntegerKernel(taps, size, weights, sums, pos, field, idx)
	//coeffs, offsets = reduceKernel(coeffs, offsets, taps, size)
	if cfg.Vertical {
		for i := size - 1; i > 0; i-- {
			offsets[i] = offsets[i] - offsets[i-1]
		}
	}
	return Kernel{coeffs, offsets, taps}
}
