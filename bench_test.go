// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

import (
	"image"
	_ "image/jpeg"
	"testing"
)

type BenchType struct {
	win, hin   int
	wout, hout int
	interlaced bool
	rgb        bool
	filter     Filter
}

var (
	benchs = []BenchType{
		{640, 480, 1920, 1080, false, false, NewBilinearFilter()},
		{640, 480, 1920, 1080, false, false, NewBicubicFilter()},
		{640, 480, 1920, 1080, false, false, NewLanczosFilter(3)},
		{1920, 1080, 640, 480, false, false, NewBilinearFilter()},
		{1920, 1080, 640, 480, false, false, NewBicubicFilter()},
		{1920, 1080, 640, 480, false, false, NewLanczosFilter(3)},
		{640, 480, 1920, 1080, true, false, NewBicubicFilter()},
		{1920, 1080, 640, 480, true, false, NewBicubicFilter()},
		{512, 512, 512, 512, true, false, NewBilinearFilter()},
		{720, 576, 640, 480, false, true, NewBicubicFilter()},
	}
)

func benchSpeed(b *testing.B, bt BenchType) {
	raw := readImage(b, "testdata/lenna.jpg")
	src := image.NewYCbCr(image.Rect(0, 0, bt.win, bt.hin), image.YCbCrSubsampleRatio420)
	convert(b, src, raw, bt.interlaced, bt.filter)
	dst := image.NewYCbCr(image.Rect(0, 0, bt.wout, bt.hout), image.YCbCrSubsampleRatio420)
	converter := prepare(b, dst, src, bt.interlaced, bt.filter)
	b.SetBytes(int64(bt.wout*bt.hout*3) >> 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.Convert(dst, src)
	}
}

func BenchmarkImageBilinearUp(b *testing.B)   { benchSpeed(b, benchs[0]) }
func BenchmarkImageBicubicUp(b *testing.B)    { benchSpeed(b, benchs[1]) }
func BenchmarkImageLanczosUp(b *testing.B)    { benchSpeed(b, benchs[2]) }
func BenchmarkImageBilinearDown(b *testing.B) { benchSpeed(b, benchs[3]) }
func BenchmarkImageBicubicDown(b *testing.B)  { benchSpeed(b, benchs[4]) }
func BenchmarkImageLanczosDown(b *testing.B)  { benchSpeed(b, benchs[5]) }
func BenchmarkImageBicubicIUp(b *testing.B)   { benchSpeed(b, benchs[6]) }
func BenchmarkImageBicubicIDown(b *testing.B) { benchSpeed(b, benchs[7]) }
func BenchmarkCopy(b *testing.B)              { benchSpeed(b, benchs[8]) }
func BenchmarkImageBicubicRgb(b *testing.B)   { benchSpeed(b, benchs[9]) }

func benchScaler(b *testing.B, vertical bool, taps int) {
	n := 96
	src := make([]byte, n*n)
	dst := make([]byte, n*n*2)
	cfg := ResizerConfig{
		Input:      n,
		Output:     n * 2,
		Vertical:   vertical,
		Interlaced: false,
		Threads:    1,
	}
	dp := n
	if !vertical {
		dp *= 2
	}
	resizer := NewResize(&cfg, NewLanczosFilter(taps>>1))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resizer.Resize(dst, src, n, n, dp, n)
	}

}

// synthetic benchmarks
func BenchmarkVerticalScaler2(b *testing.B)    { benchScaler(b, true, 2) }
func BenchmarkVerticalScaler4(b *testing.B)    { benchScaler(b, true, 4) }
func BenchmarkVerticalScaler6(b *testing.B)    { benchScaler(b, true, 6) }
func BenchmarkVerticalScaler8(b *testing.B)    { benchScaler(b, true, 8) }
func BenchmarkVerticalScaler10(b *testing.B)   { benchScaler(b, true, 10) }
func BenchmarkVerticalScaler12(b *testing.B)   { benchScaler(b, true, 12) }
func BenchmarkVerticalScalerN(b *testing.B)    { benchScaler(b, true, 14) }
func BenchmarkHorizontalScaler2(b *testing.B)  { benchScaler(b, false, 2) }
func BenchmarkHorizontalScaler4(b *testing.B)  { benchScaler(b, false, 4) }
func BenchmarkHorizontalScaler6(b *testing.B)  { benchScaler(b, false, 6) }
func BenchmarkHorizontalScaler8(b *testing.B)  { benchScaler(b, false, 8) }
func BenchmarkHorizontalScaler10(b *testing.B) { benchScaler(b, false, 10) }
func BenchmarkHorizontalScaler12(b *testing.B) { benchScaler(b, false, 12) }
func BenchmarkHorizontalScalerN(b *testing.B)  { benchScaler(b, false, 14) }
