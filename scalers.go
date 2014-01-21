// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

const (
	Bits = 14
)

func u8(x int) uint8 {
	if x < 0 {
		return 0
	}
	if x > 0xFF {
		return 0xFF
	}
	return uint8(x)
}

func h8scaleN(taps, width, height int, coeffs []int16, offsets []int,
	dst, src []uint8, dpitch, spitch int) {
	for y := 0; y < height; y++ {
		c := coeffs
		for x := range dst[:width] {
			offset := offsets[x]
			pix := 0
			for i, d := range src[offset : offset+taps] {
				pix += int(d) * int(c[i])
			}
			dst[x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[taps:]
		}
		src = src[spitch:]
		dst = dst[dpitch:]
	}
}

func v8scaleN(taps, width, height int, coeffs []int16, offsets []int,
	dst, src []uint8, dpitch, spitch int) {
	for _, offset := range offsets {
		src = src[spitch*offset:]
		for x := range dst[:width] {
			pix := 0
			for i, c := range coeffs[:taps] {
				pix += int(src[spitch*i+x]) * int(c)
			}
			dst[x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		coeffs = coeffs[taps:]
		dst = dst[dpitch:]
	}
}
