// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"
	"reflect"
	"runtime"
	"testing"
)

type Tester interface {
	Fatalf(format string, args ...interface{})
}

func expect(t Tester, a, b interface{}) {
	if reflect.DeepEqual(a, b) {
		return
	}
	typea := reflect.TypeOf(a)
	typeb := reflect.TypeOf(b)
	_, file, line, _ := runtime.Caller(1)
	t.Fatalf("%v:%v got %v(%v), want %v(%v)\n", file, line,
		typea, a, typeb, b)
}

func readImage(t Tester, name string) *image.YCbCr {
	file, err := os.Open(name)
	expect(t, err, nil)
	defer file.Close()
	raw, _, err := image.Decode(file)
	expect(t, err, nil)
	yuv, ok := raw.(*image.YCbCr)
	expect(t, ok, true)
	return yuv
}

func writeImage(t Tester, name string, img image.Image) {
	file, err := os.Create(name)
	expect(t, err, nil)
	defer file.Close()
	err = png.Encode(file, img)
	expect(t, err, nil)
}

func prepare(t Tester, dst, src image.Image, interlaced bool, filter Filter) Converter {
	cfg, err := PrepareConversion(dst, src)
	cfg.Input.Interlaced = interlaced
	cfg.Output.Interlaced = interlaced
	converter, err := NewConverter(cfg, filter)
	expect(t, err, nil)
	return converter
}

func convert(t Tester, dst, src *image.YCbCr, interlaced bool, filter Filter) {
	converter := prepare(t, dst, src, interlaced, filter)
	err := converter.Convert(dst, src)
	expect(t, err, nil)
}

func convertFiles(t Tester, w, h int, input, output string, filter Filter) {
	src := readImage(t, input)
	dst := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio420)
	err := Convert(dst, src, filter)
	expect(t, err, nil)
	writeImage(t, output, dst)
}

var (
	filters = []Filter{
		NewBilinearFilter(),
		NewBicubicFilter(),
		NewLanczosFilter(3),
	}
)

func TestConvert(t *testing.T) {
	t.Skip("skipping slow test")
	sizes := []struct{ w, h int }{
		{128, 128},
		{256, 256},
		{720, 576},
		{1920, 1080},
	}
	for _, f := range filters {
		for _, s := range sizes {
			dst := fmt.Sprintf("testdata/output-%vx%v-%v.png", s.w, s.h, f.Name())
			convertFiles(t, s.w, s.h, "testdata/lenna.jpg", dst, f)
		}
	}
}

func testBoundariesWith(t *testing.T, interlaced bool) {
	// test we don't go overread/overwrite even with exotic resolutions
	src := readImage(t, "testdata/lenna.jpg")
	min := 0
	if interlaced {
		min = 1
	}
	for _, f := range filters {
		tmp := image.NewYCbCr(image.Rect(0, 0, 256, 256), image.YCbCrSubsampleRatio444)
		convert(t, tmp, src, interlaced, f)
		last := tmp.Rect.Dx()
		for i := 32; i > min; i >>= 1 {
			last += i
			dst := image.NewYCbCr(image.Rect(0, 0, last, last), image.YCbCrSubsampleRatio444)
			convert(t, dst, tmp, interlaced, f)
			convert(t, tmp, dst, interlaced, f)
		}
	}
}

func TestProgressiveBoundaries(t *testing.T) { testBoundariesWith(t, false) }
func TestInterlacedBoundaries(t *testing.T)  { testBoundariesWith(t, true) }

func TestCopy(t *testing.T) {
	convertFiles(t, 512, 512, "testdata/lenna.jpg", "testdata/copy.png", NewBilinearFilter())
}

func TestInterlacedFail(t *testing.T) {
	raw := readImage(t, "testdata/lenna.jpg")
	src := image.NewYCbCr(image.Rect(0, 0, 640, 480), image.YCbCrSubsampleRatio420)
	convert(t, src, raw, true, NewBicubicFilter())
}

func testDegradation(t *testing.T, w, h int, interlaced bool, filter Filter) {
	src := readImage(t, "testdata/lenna.jpg")
	dst := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio444)
	fwd := prepare(t, dst, src, interlaced, filter)
	bwd := prepare(t, src, dst, interlaced, filter)
	for i := 0; i < 32; i++ {
		err := fwd.Convert(dst, src)
		expect(t, err, nil)
		err = bwd.Convert(src, dst)
		expect(t, err, nil)
	}
	ref := readImage(t, "testdata/lenna.jpg")
	psnrs, err := Psnr(ref, src)
	expect(t, err, nil)
	if false {
		name := fmt.Sprintf("testdata/degraded-%vx%v-%v-%v.png", w, h, toInterlacedString(interlaced), filter.Name())
		writeImage(t, name, src)
	}
	for i, v := range psnrs {
		min := float64(22)
		if i > 0 {
			min = 30
		}
		expect(t, v > min, true)
	}
}

func TestDegradations(t *testing.T) {
	for _, f := range filters {
		testDegradation(t, 256+1, 256+1, false, f)
		testDegradation(t, 256+2, 256+2, true, f)
	}
}

type BenchType struct {
	win, hin   int
	wout, hout int
	interlaced bool
	filter     Filter
}

var (
	benchs = []BenchType{
		{640, 480, 1920, 1080, false, NewBilinearFilter()},
		{640, 480, 1920, 1080, false, NewBicubicFilter()},
		{640, 480, 1920, 1080, false, NewLanczosFilter(3)},
		{1920, 1080, 640, 480, false, NewBilinearFilter()},
		{1920, 1080, 640, 480, false, NewBicubicFilter()},
		{1920, 1080, 640, 480, false, NewLanczosFilter(3)},
		{640, 480, 1920, 1080, true, NewBicubicFilter()},
		{640, 480, 1920, 1080, true, NewBicubicFilter()},
		{512, 512, 512, 512, true, NewBilinearFilter()},
		{512, 512, 512, 512, true, NewLanczosFilter(3)},
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

func BenchmarkBilinearUp(b *testing.B)   { benchSpeed(b, benchs[0]) }
func BenchmarkBicubicUp(b *testing.B)    { benchSpeed(b, benchs[1]) }
func BenchmarkLanczosUp(b *testing.B)    { benchSpeed(b, benchs[2]) }
func BenchmarkBilinearDown(b *testing.B) { benchSpeed(b, benchs[3]) }
func BenchmarkBicubicDown(b *testing.B)  { benchSpeed(b, benchs[4]) }
func BenchmarkLanczosDown(b *testing.B)  { benchSpeed(b, benchs[5]) }
func BenchmarkBicubicIUp(b *testing.B)   { benchSpeed(b, benchs[6]) }
func BenchmarkBicubicIDown(b *testing.B) { benchSpeed(b, benchs[7]) }

// filter should not matter
func BenchmarkBicubicCopy(b *testing.B) { benchSpeed(b, benchs[8]) }
func BenchmarkLanczosCopy(b *testing.B) { benchSpeed(b, benchs[9]) }
