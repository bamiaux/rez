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

func expect(t *testing.T, a, b interface{}) {
	if reflect.DeepEqual(a, b) {
		return
	}
	typea := reflect.TypeOf(a)
	typeb := reflect.TypeOf(b)
	_, file, line, _ := runtime.Caller(1)
	t.Fatalf("%v:%v got %v(%v), want %v(%v)\n", file, line,
		typea, a, typeb, b)
}

func readImage(t *testing.T, name string) *image.YCbCr {
	file, err := os.Open(name)
	expect(t, err, nil)
	defer file.Close()
	raw, _, err := image.Decode(file)
	expect(t, err, nil)
	yuv, ok := raw.(*image.YCbCr)
	expect(t, ok, true)
	return yuv
}

func writeImage(t *testing.T, name string, img image.Image) {
	file, err := os.Create(name)
	expect(t, err, nil)
	defer file.Close()
	err = png.Encode(file, img)
	expect(t, err, nil)
}

func resize(t *testing.T, dst, src *image.YCbCr, filter Filter) {
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
	err = adapter.Resize(dst, src)
	expect(t, err, nil)
}

func resizeFiles(t *testing.T, w, h int, input, output string, filter Filter) {
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
