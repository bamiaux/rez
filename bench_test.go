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

func benchSpeed(b *testing.B, bt BenchType, asm bool) {
	raw := readImage(b, "testdata/lenna.jpg")
	src := image.NewYCbCr(image.Rect(0, 0, bt.win, bt.hin), image.YCbCrSubsampleRatio420)
	convert(b, src, raw, asm, bt.interlaced, bt.filter)
	dst := image.NewYCbCr(image.Rect(0, 0, bt.wout, bt.hout), image.YCbCrSubsampleRatio420)
	converter := prepare(b, dst, src, asm, bt.interlaced, bt.filter, 0)
	b.SetBytes(int64(bt.wout*bt.hout*3) >> 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.Convert(dst, src)
	}
}

func BenchmarkImageBilinearUpGo(b *testing.B)    { benchSpeed(b, benchs[0], false) }
func BenchmarkImageBilinearUpAsm(b *testing.B)   { benchSpeed(b, benchs[0], true) }
func BenchmarkImageBicubicUpGo(b *testing.B)     { benchSpeed(b, benchs[1], false) }
func BenchmarkImageBicubicUpAsm(b *testing.B)    { benchSpeed(b, benchs[1], true) }
func BenchmarkImageLanczosUpGo(b *testing.B)     { benchSpeed(b, benchs[2], false) }
func BenchmarkImageLanczosUpAsm(b *testing.B)    { benchSpeed(b, benchs[2], true) }
func BenchmarkImageBilinearDownGo(b *testing.B)  { benchSpeed(b, benchs[3], false) }
func BenchmarkImageBilinearDownAsm(b *testing.B) { benchSpeed(b, benchs[3], true) }
func BenchmarkImageBicubicDownGo(b *testing.B)   { benchSpeed(b, benchs[4], false) }
func BenchmarkImageBicubicDownAsm(b *testing.B)  { benchSpeed(b, benchs[4], true) }
func BenchmarkImageLanczosDownGo(b *testing.B)   { benchSpeed(b, benchs[5], false) }
func BenchmarkImageLanczosDownAsm(b *testing.B)  { benchSpeed(b, benchs[5], true) }
func BenchmarkImageBicubicIUpGo(b *testing.B)    { benchSpeed(b, benchs[6], false) }
func BenchmarkImageBicubicIUpAsm(b *testing.B)   { benchSpeed(b, benchs[6], true) }
func BenchmarkImageBicubicIDownGo(b *testing.B)  { benchSpeed(b, benchs[7], false) }
func BenchmarkImageBicubicIDownAsm(b *testing.B) { benchSpeed(b, benchs[7], true) }
func BenchmarkImageBicubicRgbGo(b *testing.B)    { benchSpeed(b, benchs[9], false) }
func BenchmarkImageBicubicRgbAsm(b *testing.B)   { benchSpeed(b, benchs[9], true) }
func BenchmarkCopy(b *testing.B)                 { benchSpeed(b, benchs[8], false) }

func benchScaler(b *testing.B, asm, vertical bool, taps int) {
	n := 96
	src := make([]byte, n*n)
	dst := make([]byte, n*n*2)
	cfg := ResizerConfig{
		Input:      n,
		Output:     n * 2,
		Vertical:   vertical,
		Interlaced: false,
		Threads:    1,
		DisableAsm: !asm,
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
func BenchmarkVerticalScaler2Go(b *testing.B)     { benchScaler(b, false, true, 2) }
func BenchmarkVerticalScaler2Asm(b *testing.B)    { benchScaler(b, true, true, 2) }
func BenchmarkVerticalScaler4Go(b *testing.B)     { benchScaler(b, false, true, 4) }
func BenchmarkVerticalScaler4Asm(b *testing.B)    { benchScaler(b, true, true, 4) }
func BenchmarkVerticalScaler6Go(b *testing.B)     { benchScaler(b, false, true, 6) }
func BenchmarkVerticalScaler6Asm(b *testing.B)    { benchScaler(b, true, true, 6) }
func BenchmarkVerticalScaler8Go(b *testing.B)     { benchScaler(b, false, true, 8) }
func BenchmarkVerticalScaler8Asm(b *testing.B)    { benchScaler(b, true, true, 8) }
func BenchmarkVerticalScaler10Go(b *testing.B)    { benchScaler(b, false, true, 10) }
func BenchmarkVerticalScaler10Asm(b *testing.B)   { benchScaler(b, true, true, 10) }
func BenchmarkVerticalScaler12Go(b *testing.B)    { benchScaler(b, false, true, 12) }
func BenchmarkVerticalScaler12Asm(b *testing.B)   { benchScaler(b, true, true, 12) }
func BenchmarkVerticalScalerNGo(b *testing.B)     { benchScaler(b, false, true, 14) }
func BenchmarkVerticalScalerNAsm(b *testing.B)    { benchScaler(b, true, true, 14) }
func BenchmarkHorizontalScaler2Go(b *testing.B)   { benchScaler(b, false, false, 2) }
func BenchmarkHorizontalScaler2Asm(b *testing.B)  { benchScaler(b, true, false, 2) }
func BenchmarkHorizontalScaler4Go(b *testing.B)   { benchScaler(b, false, false, 4) }
func BenchmarkHorizontalScaler4Asm(b *testing.B)  { benchScaler(b, true, false, 4) }
func BenchmarkHorizontalScaler6Go(b *testing.B)   { benchScaler(b, false, false, 6) }
func BenchmarkHorizontalScaler6Asm(b *testing.B)  { benchScaler(b, true, false, 6) }
func BenchmarkHorizontalScaler8Go(b *testing.B)   { benchScaler(b, false, false, 8) }
func BenchmarkHorizontalScaler8Asm(b *testing.B)  { benchScaler(b, true, false, 8) }
func BenchmarkHorizontalScaler10Go(b *testing.B)  { benchScaler(b, false, false, 10) }
func BenchmarkHorizontalScaler10Asm(b *testing.B) { benchScaler(b, true, false, 10) }
func BenchmarkHorizontalScaler12Go(b *testing.B)  { benchScaler(b, false, false, 12) }
func BenchmarkHorizontalScaler12Asm(b *testing.B) { benchScaler(b, true, false, 12) }
func BenchmarkHorizontalScalerNGo(b *testing.B)   { benchScaler(b, false, false, 14) }
func BenchmarkHorizontalScalerNAsm(b *testing.B)  { benchScaler(b, true, false, 14) }
