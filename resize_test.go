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

func resize(t *testing.T, w, h int, input, output string, filter Filter) {
	src := readImage(t, input)
	dst := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio420)
	cfg := AdapterConfig{
		Input: Descriptor{
			Width:  src.Rect.Dx(),
			Height: src.Rect.Dy(),
			Ratio:  GetRatio(src.SubsampleRatio),
		},
		Output: Descriptor{
			Width:  w,
			Height: h,
			Ratio:  GetRatio(dst.SubsampleRatio),
		},
	}
	adapter, err := NewAdapter(&cfg, filter)
	expect(t, err, nil)
	err = adapter.Resize(dst, src)
	expect(t, err, nil)
	writeImage(t, output, dst)
}

func TestResize(t *testing.T) {
	filters := []Filter{
		NewBilinearFilter(),
		NewBicubicFilter(),
		NewLanczosFilter(3),
	}
	sizes := []struct{ w, h int }{
		{w: 128, h: 128},
		{w: 256, h: 256},
		{w: 720, h: 576},
		{w: 1920, h: 1080},
	}
	for _, f := range filters {
		for _, s := range sizes {
			dst := fmt.Sprintf("testdata/output-%vx%v-%v.png", s.w, s.h, f.Name())
			resize(t, s.w, s.h, "testdata/lenna.jpg", dst, f)
		}
	}
}
