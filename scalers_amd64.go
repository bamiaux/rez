// Copyright 2014 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

func hasAsm() bool { return true }

func h8scale2(dst, src []byte, cof []int16, off []int, taps, width, height, dp, sp int)
func h8scale4(dst, src []byte, cof []int16, off []int, taps, width, height, dp, sp int)
func h8scale8(dst, src []byte, cof []int16, off []int, taps, width, height, dp, sp int)
func h8scale10(dst, src []byte, cof []int16, off []int, taps, width, height, dp, sp int)
func h8scale12(dst, src []byte, cof []int16, off []int, taps, width, height, dp, sp int)
func h8scaleN(dst, src []byte, cof []int16, off []int, taps, width, height, dp, sp int)
func v8scale2(dst, src []byte, cof []int16, off []int, taps, width, height, dp, sp int)

var (
	h8scale6  = h8scaleN
	v8scale4  = v8scale4Go
	v8scale6  = v8scale6Go
	v8scale8  = v8scale8Go
	v8scale10 = v8scale10Go
	v8scale12 = v8scale12Go
	v8scaleN  = v8scaleNGo
)
