// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

import (
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

func doubleWidth(t *testing.T, input, output string, filter Filter) {
	src := readImage(t, input)
	r := src.Bounds()
	dst := image.NewYCbCr(image.Rect(0, 0, r.Dx()*2, r.Dy()),
		image.YCbCrSubsampleRatio420)
	_ = dst
	for i := 0; i < 3; i++ {
		w := r.Dx()
		h := r.Dy()
		if i > 0 {
			w >>= 1
			h >>= 1
		}
		cfg := Config{
			depth:    8,
			input:    w,
			output:   w * 2,
			vertical: false,
		}
		rez := NewResize(&cfg, filter)
		sptr := src.Y
		dptr := dst.Y
		offset := src.YOffset
		spitch := src.YStride
		dpitch := dst.YStride
		if i > 0 {
			sptr = src.Cb
			dptr = dst.Cb
			offset = src.COffset
			spitch = src.CStride
			dpitch = dst.CStride
		}
		if i > 1 {
			sptr = src.Cr
			dptr = dst.Cr
		}
		sptr = sptr[offset(0, 0):]
		dptr = dptr[offset(0, 0):]
		rez.Resize(dptr, sptr, dpitch, spitch, w, h)
	}
	writeImage(t, output, dst)
}

func TestResizeWidth(t *testing.T) {
	doubleWidth(t, "testdata/lenna.jpg", "testdata/lenna-bilinear.png", NewBilinearFilter())
	doubleWidth(t, "testdata/lenna.jpg", "testdata/lenna-bicubic.png", NewBicubicFilter())
	doubleWidth(t, "testdata/lenna.jpg", "testdata/lenna-lanczos.png", NewLanczosFilter(3))
}
