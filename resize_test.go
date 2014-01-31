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

func adapt(t Tester, dst, src *image.YCbCr, filter Filter) Adapter {
	cfg := AdapterConfig{
		Input: Descriptor{
			Width:  src.Rect.Dx(),
			Height: src.Rect.Dy(),
			Ratio:  GetRatio(src.SubsampleRatio),
		},
		Output: Descriptor{
			Width:  dst.Rect.Dx(),
			Height: dst.Rect.Dy(),
			Ratio:  GetRatio(dst.SubsampleRatio),
		},
	}
	adapter, err := NewAdapter(&cfg, filter)
	expect(t, err, nil)
	return adapter
}

func resize(t Tester, dst, src *image.YCbCr, filter Filter) {
	adapter := adapt(t, dst, src, filter)
	err := adapter.Resize(dst, src)
	expect(t, err, nil)
}

func resizeFiles(t Tester, w, h int, input, output string, filter Filter) {
	src := readImage(t, input)
	dst := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio420)
	resize(t, dst, src, filter)
	writeImage(t, output, dst)
}

var (
	filters = []Filter{
		NewBilinearFilter(),
		NewBicubicFilter(),
		NewLanczosFilter(3),
	}
)

func TestResize(t *testing.T) {
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
			resizeFiles(t, s.w, s.h, "testdata/lenna.jpg", dst, f)
		}
	}
}

func TestBoundaries(t *testing.T) {
	// test we don't go overread/overwrite even with exotic resolutions
	src := readImage(t, "testdata/lenna.jpg")
	for _, f := range filters {
		tmp := image.NewYCbCr(image.Rect(0, 0, 256, 256), image.YCbCrSubsampleRatio444)
		resize(t, tmp, src, f)
		last := tmp.Rect.Dx()
		for i := 32; i > 0; i >>= 1 {
			last += i
			dst := image.NewYCbCr(image.Rect(0, 0, last, last), image.YCbCrSubsampleRatio444)
			resize(t, dst, tmp, f)
			resize(t, tmp, dst, f)
		}
	}
}

func TestCopy(t *testing.T) {
	resizeFiles(t, 512, 512, "testdata/lenna.jpg", "testdata/copy.png", NewBilinearFilter())
}

func benchSpeed(b *testing.B, win, hin int, wout, hout int, filter Filter) {
	raw := readImage(b, "testdata/lenna.jpg")
	src := image.NewYCbCr(image.Rect(0, 0, win, hin), image.YCbCrSubsampleRatio420)
	resize(b, src, raw, filter)
	dst := image.NewYCbCr(image.Rect(0, 0, wout, hout), image.YCbCrSubsampleRatio420)
	adapter := adapt(b, dst, src, filter)
	b.SetBytes(int64(wout*hout*3) >> 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.Resize(dst, src)
	}
}

func BenchmarkBilinearUp(b *testing.B)   { benchSpeed(b, 640, 480, 1920, 1080, NewBilinearFilter()) }
func BenchmarkBicubicUp(b *testing.B)    { benchSpeed(b, 640, 480, 1920, 1080, NewBicubicFilter()) }
func BenchmarkLanczosUp(b *testing.B)    { benchSpeed(b, 640, 480, 1920, 1080, NewLanczosFilter(3)) }
func BenchmarkBilinearDown(b *testing.B) { benchSpeed(b, 1920, 1080, 640, 480, NewBilinearFilter()) }
func BenchmarkBicubicDown(b *testing.B)  { benchSpeed(b, 1920, 1080, 640, 480, NewBicubicFilter()) }
func BenchmarkLanczosDown(b *testing.B)  { benchSpeed(b, 1920, 1080, 640, 480, NewLanczosFilter(3)) }
