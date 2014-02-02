// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rez

// This file is auto-generated - do not modify

func h8scale2(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	si := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[di : di+width] {
			xoff := si + off[x]
			pix := int(src[xoff+0])*int(c[0]) +
				int(src[xoff+1])*int(c[1])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[2:]
		}
		di += dp
		si += sp
	}
}

func v8scale2(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[di : di+width] {
			pix := int(src[sp*0+x])*int(cof[0]) +
				int(src[sp*1+x])*int(cof[1])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[2:]
		di += dp
	}
}

func h8scale4(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	si := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[di : di+width] {
			xoff := si + off[x]
			pix := int(src[xoff+0])*int(c[0]) +
				int(src[xoff+1])*int(c[1]) +
				int(src[xoff+2])*int(c[2]) +
				int(src[xoff+3])*int(c[3])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[4:]
		}
		di += dp
		si += sp
	}
}

func v8scale4(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[di : di+width] {
			pix := int(src[sp*0+x])*int(cof[0]) +
				int(src[sp*1+x])*int(cof[1]) +
				int(src[sp*2+x])*int(cof[2]) +
				int(src[sp*3+x])*int(cof[3])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[4:]
		di += dp
	}
}

func h8scale6(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	si := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[di : di+width] {
			xoff := si + off[x]
			pix := int(src[xoff+0])*int(c[0]) +
				int(src[xoff+1])*int(c[1]) +
				int(src[xoff+2])*int(c[2]) +
				int(src[xoff+3])*int(c[3]) +
				int(src[xoff+4])*int(c[4]) +
				int(src[xoff+5])*int(c[5])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[6:]
		}
		di += dp
		si += sp
	}
}

func v8scale6(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[di : di+width] {
			pix := int(src[sp*0+x])*int(cof[0]) +
				int(src[sp*1+x])*int(cof[1]) +
				int(src[sp*2+x])*int(cof[2]) +
				int(src[sp*3+x])*int(cof[3]) +
				int(src[sp*4+x])*int(cof[4]) +
				int(src[sp*5+x])*int(cof[5])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[6:]
		di += dp
	}
}

func h8scale8(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	si := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[di : di+width] {
			xoff := si + off[x]
			pix := int(src[xoff+0])*int(c[0]) +
				int(src[xoff+1])*int(c[1]) +
				int(src[xoff+2])*int(c[2]) +
				int(src[xoff+3])*int(c[3]) +
				int(src[xoff+4])*int(c[4]) +
				int(src[xoff+5])*int(c[5]) +
				int(src[xoff+6])*int(c[6]) +
				int(src[xoff+7])*int(c[7])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[8:]
		}
		di += dp
		si += sp
	}
}

func v8scale8(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[di : di+width] {
			pix := int(src[sp*0+x])*int(cof[0]) +
				int(src[sp*1+x])*int(cof[1]) +
				int(src[sp*2+x])*int(cof[2]) +
				int(src[sp*3+x])*int(cof[3]) +
				int(src[sp*4+x])*int(cof[4]) +
				int(src[sp*5+x])*int(cof[5]) +
				int(src[sp*6+x])*int(cof[6]) +
				int(src[sp*7+x])*int(cof[7])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[8:]
		di += dp
	}
}

func h8scale10(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	si := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[di : di+width] {
			xoff := si + off[x]
			pix := int(src[xoff+0])*int(c[0]) +
				int(src[xoff+1])*int(c[1]) +
				int(src[xoff+2])*int(c[2]) +
				int(src[xoff+3])*int(c[3]) +
				int(src[xoff+4])*int(c[4]) +
				int(src[xoff+5])*int(c[5]) +
				int(src[xoff+6])*int(c[6]) +
				int(src[xoff+7])*int(c[7]) +
				int(src[xoff+8])*int(c[8]) +
				int(src[xoff+9])*int(c[9])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[10:]
		}
		di += dp
		si += sp
	}
}

func v8scale10(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[di : di+width] {
			pix := int(src[sp*0+x])*int(cof[0]) +
				int(src[sp*1+x])*int(cof[1]) +
				int(src[sp*2+x])*int(cof[2]) +
				int(src[sp*3+x])*int(cof[3]) +
				int(src[sp*4+x])*int(cof[4]) +
				int(src[sp*5+x])*int(cof[5]) +
				int(src[sp*6+x])*int(cof[6]) +
				int(src[sp*7+x])*int(cof[7]) +
				int(src[sp*8+x])*int(cof[8]) +
				int(src[sp*9+x])*int(cof[9])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[10:]
		di += dp
	}
}

func h8scale12(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	si := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[di : di+width] {
			xoff := si + off[x]
			pix := int(src[xoff+0])*int(c[0]) +
				int(src[xoff+1])*int(c[1]) +
				int(src[xoff+2])*int(c[2]) +
				int(src[xoff+3])*int(c[3]) +
				int(src[xoff+4])*int(c[4]) +
				int(src[xoff+5])*int(c[5]) +
				int(src[xoff+6])*int(c[6]) +
				int(src[xoff+7])*int(c[7]) +
				int(src[xoff+8])*int(c[8]) +
				int(src[xoff+9])*int(c[9]) +
				int(src[xoff+10])*int(c[10]) +
				int(src[xoff+11])*int(c[11])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[12:]
		}
		di += dp
		si += sp
	}
}

func v8scale12(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	di := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[di : di+width] {
			pix := int(src[sp*0+x])*int(cof[0]) +
				int(src[sp*1+x])*int(cof[1]) +
				int(src[sp*2+x])*int(cof[2]) +
				int(src[sp*3+x])*int(cof[3]) +
				int(src[sp*4+x])*int(cof[4]) +
				int(src[sp*5+x])*int(cof[5]) +
				int(src[sp*6+x])*int(cof[6]) +
				int(src[sp*7+x])*int(cof[7]) +
				int(src[sp*8+x])*int(cof[8]) +
				int(src[sp*9+x])*int(cof[9]) +
				int(src[sp*10+x])*int(cof[10]) +
				int(src[sp*11+x])*int(cof[11])
			dst[di+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[12:]
		di += dp
	}
}
