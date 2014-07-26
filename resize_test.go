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

func prepare(t Tester, dst, src image.Image, asm, interlaced bool, filter Filter, threads int) Converter {
	cfg, err := PrepareConversion(dst, src)
	expect(t, err, nil)
	cfg.Input.Interlaced = interlaced
	cfg.Output.Interlaced = interlaced
	cfg.Threads = threads
	cfg.DisableAsm = !asm
	converter, err := NewConverter(cfg, filter)
	expect(t, err, nil)
	return converter
}

func convert(t Tester, dst, src image.Image, asm, interlaced bool, filter Filter) {
	converter := prepare(t, dst, src, asm, interlaced, filter, 0)
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

func toRgb(src image.Image) *image.RGBA {
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

type TestCase struct {
	file       string
	src        image.Rectangle
	dst        image.Rectangle
	rgb        bool
	interlaced bool
	filter     Filter
	threads    int
	psnrs      []float64
	psnrRect   image.Rectangle
	dump       string
}

func NewTestCase(w, h int, interlaced bool) *TestCase {
	return &TestCase{
		file:       "lenna.jpg",
		filter:     NewBicubicFilter(),
		interlaced: interlaced,
		dst:        image.Rect(0, 0, w, h),
	}
}

func checkPsnrs(t *testing.T, ref, img image.Image, sub image.Rectangle, min []float64) {
	var a, b image.Image
	a, b = ref, img
	if !sub.Empty() {
		switch a.(type) {
		case *image.RGBA:
			a = a.(*image.RGBA).SubImage(sub)
			b = b.(*image.RGBA).SubImage(sub)
		case *image.NRGBA:
			a = a.(*image.NRGBA).SubImage(sub)
			b = b.(*image.NRGBA).SubImage(sub)
		case *image.YCbCr:
			a = a.(*image.YCbCr).SubImage(sub)
			b = b.(*image.YCbCr).SubImage(sub)
		case *image.Gray:
			a = a.(*image.Gray).SubImage(sub)
			b = b.(*image.Gray).SubImage(sub)
		}
	}
	psnrs, err := Psnr(a, b)
	expect(t, err, nil)
	for i, v := range psnrs {
		if v < min[i] {
			t.Fatalf("invalid psnr %v < %v\n", v, min[i])
		}
	}
}

func runTestCaseWith(t *testing.T, tc *TestCase, asm bool, cycles int) image.Image {
	srcRaw := readImage(t, "testdata/"+tc.file).(*image.YCbCr)
	dstRaw := image.NewYCbCr(image.Rect(0, 0, tc.dst.Max.X*2, tc.dst.Max.Y*2), srcRaw.SubsampleRatio)
	var src, dst, ref image.Image
	if tc.src.Empty() {
		tc.src = srcRaw.Bounds()
	}
	suffix := "yuv"
	if tc.rgb {
		suffix = "rgb"
		src = toRgb(srcRaw).SubImage(tc.src)
		ref = toRgb(srcRaw).SubImage(tc.src)
		dst = toRgb(dstRaw).SubImage(tc.dst)
	} else {
		src = srcRaw.SubImage(tc.src)
		refRaw := image.NewYCbCr(src.Bounds(), srcRaw.SubsampleRatio)
		err := Convert(refRaw, src, nil)
		expect(t, err, nil)
		ref = refRaw.SubImage(tc.src)
		dst = dstRaw.SubImage(tc.dst)
	}
	fwd := prepare(t, dst, src, asm, tc.interlaced, tc.filter, tc.threads)
	bwd := prepare(t, src, dst, asm, tc.interlaced, tc.filter, tc.threads)
	for i := 0; i < cycles; i++ {
		err := fwd.Convert(dst, src)
		expect(t, err, nil)
		err = bwd.Convert(src, dst)
		expect(t, err, nil)
	}
	if len(tc.psnrs) > 0 {
		checkPsnrs(t, ref, src, tc.psnrRect, tc.psnrs)
	}
	if len(tc.dump) > 0 {
		sb := src.Bounds()
		db := dst.Bounds()
		asmSuffix := "go"
		if asm {
			asmSuffix = "asm"
		}
		name := fmt.Sprintf("testdata/%v-%vx%v-%vx%v-%v-%v-%v-%v.png",
			tc.dump, sb.Dx(), sb.Dy(), db.Dx(), db.Dy(), suffix,
			toInterlacedString(tc.interlaced), asmSuffix, tc.filter.Name())
		writeImage(t, name, src)
	}
	return src
}

func runTestCase(t *testing.T, tc *TestCase, cycles int) {
	asm := runTestCaseWith(t, tc, true, cycles)
	if !hasAsm() || testing.Short() {
		return
	}
	noasm := runTestCaseWith(t, tc, false, cycles)
	if true {
		checkPsnrs(t, noasm, asm, image.Rectangle{}, []float64{math.Inf(1), math.Inf(1), math.Inf(1)})
	}
}

func TestCopy(t *testing.T) {
	tc := NewTestCase(512, 512, false)
	tc.psnrs = []float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	runTestCase(t, tc, 1)
	tc = NewTestCase(512, 512, false)
	tc.rgb = true
	tc.psnrs = []float64{math.Inf(1)}
	runTestCase(t, tc, 1)
}

func testInterlacedFailWith(t *testing.T, rgb bool) {
	src := readImage(t, "testdata/lenna.jpg")
	dst := image.Image(image.NewYCbCr(image.Rect(0, 0, 640, 480), image.YCbCrSubsampleRatio420))
	if rgb {
		src = toRgb(src)
		dst = toRgb(dst)
	}
	convert(t, dst, src, false, true, NewBicubicFilter())
	convert(t, dst, src, true, true, NewBicubicFilter())
}

func TestInterlacedFail(t *testing.T) {
	testInterlacedFailWith(t, false)
	testInterlacedFailWith(t, true)
}

func TestDegradations(t *testing.T) {
	interlaced := []bool{false, true}
	rgb := []bool{false, true}
	for _, f := range filters {
		for _, ii := range interlaced {
			offset := 1
			if ii {
				offset = 2
			}
			for _, rgb := range rgb {
				tc := NewTestCase(256+offset, 256+offset, ii)
				tc.filter = f
				tc.rgb = rgb
				tc.psnrs = []float64{22, 35, 35}
				runTestCase(t, tc, 32)
			}
		}
	}
}

func TestTooManyThreads(t *testing.T) {
	sizes := []struct{ w, h int }{{128, 16}, {16, 128}, {16, 16}}
	interlaced := []bool{false, true}
	for _, s := range sizes {
		for _, ii := range interlaced {
			tc := NewTestCase(s.w, s.h, ii)
			tc.threads = 32
			runTestCase(t, tc, 1)
		}
	}
}

func TestSmallSizes(t *testing.T) {
	if testing.Short() {
		return
	}
	interlaced := []bool{false, true}
	// we need at least 2 taps per pixel, so 4 pixels in 4:2:0
	for w := 4; w < 24; w++ {
		for _, ii := range interlaced {
			// we need at least 2 taps per field/pixel
			// so 4/8 pixels with 4:2:0 progressive/interlaced
			for h := 4 + int(bin(ii))*4; h < 24; h++ {
				tc := NewTestCase(w, h, ii)
				tc.src = image.Rect(0, 0, 32, 32)
				runTestCase(t, tc, 1)
			}
		}
	}
}

func TestSaturatedRightBorder(t *testing.T) {
	tc := NewTestCase(171, 500, false)
	tc.file = "bug3img.jpg"
	tc.rgb = true
	tc.psnrs = []float64{16}
	tc.psnrRect = image.Rect(280, 0, 286, 500)
	runTestCase(t, tc, 1)
}

func TestGrayPlanes(t *testing.T) {
	w, h := 256, 256
	src := readImage(t, "testdata/gray.png")
	ref := image.NewGray(src.Bounds())
	err := Convert(ref, src, nil)
	expect(t, err, nil)
	raw := image.NewGray(image.Rect(0, 0, w*2, h*2))
	dst := raw.SubImage(image.Rect(7, 7, 7+w, 7+h))
	err = Convert(dst, src, NewBicubicFilter())
	expect(t, err, nil)
	err = Convert(src, dst, NewBicubicFilter())
	expect(t, err, nil)
	checkPsnrs(t, ref, src, image.Rectangle{}, []float64{38})
}

func TestNrgbaPlanes(t *testing.T) {
	w, h := 256, 256
	src := readImage(t, "testdata/nrgba.png")
	ref := image.NewNRGBA(src.Bounds())
	err := Convert(ref, src, nil)
	expect(t, err, nil)
	raw := image.NewNRGBA(image.Rect(0, 0, w*2, h*2))
	dst := raw.SubImage(image.Rect(7, 7, 7+w, 7+h))
	err = Convert(dst, src, NewBicubicFilter())
	expect(t, err, nil)
	err = Convert(src, dst, NewBicubicFilter())
	expect(t, err, nil)
	checkPsnrs(t, ref, src, image.Rectangle{}, []float64{39})
}

func TestBigKernels(t *testing.T) {
	interlaced := []bool{false, true}
	for _, ii := range interlaced {
		tc := NewTestCase(256, 256, ii)
		tc.src = image.Rect(0, 0, 32, 32)
		tc.filter = NewLanczosFilter(128)
		runTestCase(t, tc, 1)
	}
}
