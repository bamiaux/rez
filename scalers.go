// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

const (
	Bits = 14
)

func u8(x int) byte {
	if x < 0 {
		return 0
	}
	if x > 0xFF {
		return 0xFF
	}
	return byte(x)
}

func copyPlane(dst, src []byte, width, height, dp, sp int) {
	dst_idx := 0
	src_idx := 0
	for y := 0; y < height; y++ {
		copy(dst[dst_idx:dst_idx+width], src[src_idx:src_idx+width])
		dst_idx += dp
		src_idx += sp
	}
}

func h8scaleN(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	src_idx := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[dst_idx : dst_idx+width] {
			xoff := src_idx + off[x]
			pix := 0
			for i, d := range src[xoff : xoff+taps] {
				pix += int(d) * int(c[i])
			}
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[taps:]
		}
		dst_idx += dp
		src_idx += sp
	}
}

func v8scaleN(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[dst_idx : dst_idx+width] {
			pix := 0
			for i, c := range cof[:taps] {
				pix += int(src[sp*i+x]) * int(c)
			}
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[taps:]
		dst_idx += dp
	}
}
