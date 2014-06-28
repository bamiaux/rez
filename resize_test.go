// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
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

func readImage(t Tester, name string) image.Image {
	file, err := os.Open(name)
	expect(t, err, nil)
	defer file.Close()
	raw, _, err := image.Decode(file)
	expect(t, err, nil)
	return raw
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

func convert(t Tester, dst, src image.Image, interlaced bool, filter Filter) {
	converter := prepare(t, dst, src, interlaced, filter)
	err := converter.Convert(dst, src)
	expect(t, err, nil)
}

func convertFiles(t Tester, w, h int, input string, filter Filter, rgb bool) (image.Image, image.Image) {
	src := readImage(t, input)
	raw := image.NewYCbCr(image.Rect(0, 0, w*2, h*2), image.YCbCrSubsampleRatio420)
	dst := raw.SubImage(image.Rect(7, 7, 7+w, 7+h))
	if rgb {
		src = toRgb(src)
		dst = toRgb(dst)
	}
	err := Convert(dst, src, filter)
	expect(t, err, nil)
	return src, dst
}

var (
	filters = []Filter{
		NewBilinearFilter(),
		NewBicubicFilter(),
		NewLanczosFilter(3),
	}
)

func TestU8(t *testing.T) {
	expect(t, u8(-1), byte(0))
	expect(t, u8(0), byte(0))
	expect(t, u8(255), byte(255))
	expect(t, u8(256), byte(255))
}

func toRgb(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, b, src, image.ZP, draw.Src)
	return dst
}

func testConvertWith(t *testing.T, rgb bool) {
	t.Skip("skipping slow test")
	sizes := []struct{ w, h int }{
		{128, 128},
		{256, 256},
		{720, 576},
		{1920, 1080},
	}
	suffix := "yuv"
	if rgb {
		suffix = "rgb"
	}
	for _, f := range filters {
		for _, s := range sizes {
			_, out := convertFiles(t, s.w, s.h, "testdata/lenna.jpg", f, rgb)
			dst := fmt.Sprintf("testdata/output-%vx%v-%v-%v.png", s.w, s.h, f.Name(), suffix)
			writeImage(t, dst, out)
		}
	}
}

func TestConvertYuv(t *testing.T) { testConvertWith(t, false) }
func TestConvertRgb(t *testing.T) { testConvertWith(t, true) }

func testBoundariesWith(t *testing.T, interlaced, rgb bool) {
	// test we don't go overread/overwrite even with exotic resolutions
	src := readImage(t, "testdata/lenna.jpg")
	min := 0
	if interlaced {
		min = 1
	}
	for _, f := range filters {
		tmp := image.Image(image.NewYCbCr(image.Rect(0, 0, 256, 256), image.YCbCrSubsampleRatio444))
		convert(t, tmp, src, interlaced, f)
		last := tmp.Bounds().Dx()
		if rgb {
			tmp = toRgb(tmp)
		}
		for i := 32; i > min; i >>= 1 {
			last += i
			dst := image.Image(image.NewYCbCr(image.Rect(0, 0, last, last), image.YCbCrSubsampleRatio444))
			if rgb {
				dst = toRgb(dst)
			}
			convert(t, dst, tmp, interlaced, f)
			convert(t, tmp, dst, interlaced, f)
		}
	}
}

func TestProgressiveYuvBoundaries(t *testing.T) { testBoundariesWith(t, false, false) }
func TestInterlacedYuvBoundaries(t *testing.T)  { testBoundariesWith(t, true, false) }
func TestProgressiveRgbBoundaries(t *testing.T) { testBoundariesWith(t, false, true) }
func TestInterlacedRgbBoundaries(t *testing.T)  { testBoundariesWith(t, true, true) }

func TestCopy(t *testing.T) {
	a, b := convertFiles(t, 512, 512, "testdata/lenna.jpg", NewBilinearFilter(), false)
	if false {
		writeImage(t, "testdata/copy-yuv.png", b)
	}
	psnrs, err := Psnr(a, b)
	expect(t, err, nil)
	expect(t, psnrs, []float64{math.Inf(1), math.Inf(1), math.Inf(1)})
	a, b = convertFiles(t, 512, 512, "testdata/lenna.jpg", NewBilinearFilter(), true)
	if false {
		writeImage(t, "testdata/copy-rgb.png", b)
	}
	psnrs, err = Psnr(a, b)
	expect(t, err, nil)
	expect(t, psnrs, []float64{math.Inf(1)})
}

func testInterlacedFailWith(t *testing.T, rgb bool) {
	src := readImage(t, "testdata/lenna.jpg")
	dst := image.Image(image.NewYCbCr(image.Rect(0, 0, 640, 480), image.YCbCrSubsampleRatio420))
	if rgb {
		src = toRgb(src)
		dst = toRgb(dst)
	}
	convert(t, dst, src, true, NewBicubicFilter())
}

func TestInterlacedFail(t *testing.T) {
	testInterlacedFailWith(t, false)
	testInterlacedFailWith(t, true)
}

func testDegradation(t *testing.T, w, h int, interlaced, rgb bool, filter Filter) {
	src := readImage(t, "testdata/lenna.jpg")
	ydst := image.NewYCbCr(image.Rect(0, 0, w*2, h*2), image.YCbCrSubsampleRatio444)
	dst := ydst.SubImage(image.Rect(7, 7, 7+w, 7+h))
	if rgb {
		src = toRgb(src)
		dst = toRgb(dst)
	}
	fwd := prepare(t, dst, src, interlaced, filter)
	bwd := prepare(t, src, dst, interlaced, filter)
	for i := 0; i < 32; i++ {
		err := fwd.Convert(dst, src)
		expect(t, err, nil)
		err = bwd.Convert(src, dst)
		expect(t, err, nil)
	}
	ref := readImage(t, "testdata/lenna.jpg")
	suffix := "yuv"
	if rgb {
		ref = toRgb(ref)
		suffix = "rgb"
	}
	psnrs, err := Psnr(ref, src)
	expect(t, err, nil)
	if false {
		name := fmt.Sprintf("testdata/degraded-%vx%v-%v-%v-%v.png", w, h, toInterlacedString(interlaced), filter.Name(), suffix)
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
		testDegradation(t, 256+1, 256+1, false, false, f)
		testDegradation(t, 256+2, 256+2, true, false, f)
		if false { //too slow for now
			testDegradation(t, 256+1, 256+1, false, true, f)
			testDegradation(t, 256+2, 256+2, true, true, f)
		}
	}
}

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
