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

func h8scale2(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	src_idx := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[dst_idx : dst_idx+width] {
			xoff := src_idx + off[x]
			pix := int(src[xoff+0])*int(c[0]) +
				int(src[xoff+1])*int(c[1])
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[2:]
		}
		dst_idx += dp
		src_idx += sp
	}
}

func v8scale2(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[dst_idx : dst_idx+width] {
			pix := int(src[sp*0+x])*int(cof[0]) +
				int(src[sp*1+x])*int(cof[1])
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[2:]
		dst_idx += dp
	}
}

func h8scale4(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	src_idx := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[dst_idx : dst_idx+width] {
			xoff := src_idx + off[x]
			pix := int(src[xoff+0])*int(c[0]) +
				int(src[xoff+1])*int(c[1]) +
				int(src[xoff+2])*int(c[2]) +
				int(src[xoff+3])*int(c[3])
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[4:]
		}
		dst_idx += dp
		src_idx += sp
	}
}

func v8scale4(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[dst_idx : dst_idx+width] {
			pix := int(src[sp*0+x])*int(cof[0]) +
				int(src[sp*1+x])*int(cof[1]) +
				int(src[sp*2+x])*int(cof[2]) +
				int(src[sp*3+x])*int(cof[3])
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[4:]
		dst_idx += dp
	}
}

func h8scale6(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	src_idx := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[dst_idx : dst_idx+width] {
			xoff := src_idx + off[x]
			pix := int(src[xoff+0])*int(c[0]) +
				int(src[xoff+1])*int(c[1]) +
				int(src[xoff+2])*int(c[2]) +
				int(src[xoff+3])*int(c[3]) +
				int(src[xoff+4])*int(c[4]) +
				int(src[xoff+5])*int(c[5])
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[6:]
		}
		dst_idx += dp
		src_idx += sp
	}
}

func v8scale6(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[dst_idx : dst_idx+width] {
			pix := int(src[sp*0+x])*int(cof[0]) +
				int(src[sp*1+x])*int(cof[1]) +
				int(src[sp*2+x])*int(cof[2]) +
				int(src[sp*3+x])*int(cof[3]) +
				int(src[sp*4+x])*int(cof[4]) +
				int(src[sp*5+x])*int(cof[5])
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[6:]
		dst_idx += dp
	}
}

func h8scale8(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	src_idx := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[dst_idx : dst_idx+width] {
			xoff := src_idx + off[x]
			pix := int(src[xoff+0])*int(c[0]) +
				int(src[xoff+1])*int(c[1]) +
				int(src[xoff+2])*int(c[2]) +
				int(src[xoff+3])*int(c[3]) +
				int(src[xoff+4])*int(c[4]) +
				int(src[xoff+5])*int(c[5]) +
				int(src[xoff+6])*int(c[6]) +
				int(src[xoff+7])*int(c[7])
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[8:]
		}
		dst_idx += dp
		src_idx += sp
	}
}

func v8scale8(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[dst_idx : dst_idx+width] {
			pix := int(src[sp*0+x])*int(cof[0]) +
				int(src[sp*1+x])*int(cof[1]) +
				int(src[sp*2+x])*int(cof[2]) +
				int(src[sp*3+x])*int(cof[3]) +
				int(src[sp*4+x])*int(cof[4]) +
				int(src[sp*5+x])*int(cof[5]) +
				int(src[sp*6+x])*int(cof[6]) +
				int(src[sp*7+x])*int(cof[7])
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[8:]
		dst_idx += dp
	}
}

func h8scale10(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	src_idx := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[dst_idx : dst_idx+width] {
			xoff := src_idx + off[x]
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
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[10:]
		}
		dst_idx += dp
		src_idx += sp
	}
}

func v8scale10(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[dst_idx : dst_idx+width] {
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
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[10:]
		dst_idx += dp
	}
}

func h8scale12(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	src_idx := 0
	for y := 0; y < height; y++ {
		c := cof
		for x := range dst[dst_idx : dst_idx+width] {
			xoff := src_idx + off[x]
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
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
			c = c[12:]
		}
		dst_idx += dp
		src_idx += sp
	}
}

func v8scale12(dst, src []byte, cof []int16, off []int,
	taps, width, height, dp, sp int) {
	dst_idx := 0
	for _, yoff := range off[:height] {
		src = src[sp*yoff:]
		for x := range dst[dst_idx : dst_idx+width] {
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
			dst[dst_idx+x] = u8((pix + 1<<(Bits-1)) >> Bits)
		}
		cof = cof[12:]
		dst_idx += dp
	}
}
