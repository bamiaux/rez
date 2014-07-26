// Copyright 2014 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	. "github.com/bamiaux/rez/asm"
)

const (
	xshift  = 4
	xwidth  = 1 << xshift // 128-bits per simd register
	xoffset = 2           // 16-bits per coeff
)

type horizontal struct {
	xtaps int
	// global data
	zero  Operand
	hbits Operand
	u8max Operand
	// arguments
	dst    []Operand
	src    []Operand
	cof    []Operand
	off    []Operand
	taps   Operand
	width  Operand
	height Operand
	dp     Operand
	sp     Operand
	// stack
	simdroll Operand
	asmroll  Operand
	srcref   Operand
	dstoff   Operand
	sum      Operand
	dstref   Operand
	count    Operand
	inner    Operand
}

func hgen(a *Asm) {
	h := horizontal{}
	h.zero = a.Data("zero", bytes.Repeat([]byte{0x00}, 16))
	h.hbits = a.Data("hbits", bytes.Repeat([]byte{0x00, 0x00, 0x20, 0x00}, 4))
	h.u8max = a.Data("u8max", bytes.Repeat([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF}, 2))
	h.genscale(a, 2)
	h.genscale(a, 4)
	h.genscale(a, 8)
	h.genscale(a, 10)
	h.genscale(a, 12)
	h.genscale(a, 0)
}

func (h *horizontal) genscale(a *Asm, taps int) {
	h.xtaps = taps
	suffix := "N"
	if taps > 0 {
		suffix = fmt.Sprintf("%v", taps)
	}
	a.NewFunction("h8scale" + suffix + "Amd64")
	// arguments
	h.dst = a.SliceArgument("dst")
	h.src = a.SliceArgument("src")
	h.cof = a.SliceArgument("cof")
	h.off = a.SliceArgument("off")
	h.taps = a.Argument("taps")
	h.width = a.Argument("width")
	h.height = a.Argument("height")
	h.dp = a.Argument("dp")
	h.sp = a.Argument("sp")
	// stack
	h.simdroll = a.PushStack("simdroll")
	h.asmroll = a.PushStack("asmroll")
	h.srcref = a.PushStack("srcref")
	h.dstoff = a.PushStack("dstoff")
	h.sum = a.PushStack("sum")
	if h.xtaps == 0 {
		h.dstref = a.PushStack("dstref")
		h.count = a.PushStack("count")
		h.inner = a.PushStack("inner")
	}
	a.Start()
	h.frame(a)
	a.Ret()
}

func (h *horizontal) setup(a *Asm) {
	a.Movq(BX, h.dp)
	a.Movq(CX, h.width)
	a.Movq(DX, CX)
	a.Subq(BX, CX)
	a.Shrq(CX, Constant(xshift))
	a.Andq(DX, Constant(1<<xshift-1))
	a.Movq(h.dstoff, BX)
	a.Movq(h.simdroll, CX)
	a.Movq(h.asmroll, DX)
	if false {
		// disable simd loops
		a.Movq(AX, Constant(0))
		a.Movq(h.simdroll, AX)
		a.Movq(AX, h.width)
		a.Movq(h.asmroll, AX)
	}
	a.Movq(AX, h.src[0])
	a.Movq(h.srcref, AX)
	a.Movq(DX, h.taps)
	a.Subq(DX, Constant(2))
	if h.xtaps == 0 {
		a.Movq(h.inner, DX)
	}
	a.Pxor(X15, X15)
	a.Movo(X14, h.hbits)
}

func (h *horizontal) frame(a *Asm) {
	h.setup(a)
	a.Movq(SI, h.src[0])
	a.Movq(DI, h.dst[0])
	yloop := a.NewLabel("yloop")
	a.Label(yloop)
	a.Movq(BX, h.off[0])
	a.Movq(BP, h.cof[0])
	h.line(a)
	h.nextline(a)
	a.Subq(h.height, Constant(1))
	a.Jne(yloop)
}

func (h *horizontal) nextline(a *Asm) {
	a.Movq(SI, h.srcref)
	a.Addq(DI, h.dstoff)
	a.Addq(SI, h.sp)
	a.Movq(h.srcref, SI)
}

func (h *horizontal) line(a *Asm) {
	simdloop := a.NewLabel("simdloop")
	asmloop := a.NewLabel("asmloop")
	nosimdloop := a.NewLabel("nosimdloop")
	end := a.NewLabel("end")

	// check if we have simd loops
	a.Movq(CX, h.simdroll)
	a.Orq(CX, CX)
	a.Je(nosimdloop)

	// apply simd loops
	a.Label(simdloop)
	switch h.xtaps {
	case 2:
		h.taps2(a)
	case 4:
		h.taps4(a)
	case 8:
		h.taps8(a)
	case 10, 12, 0:
		h.tapsn(a)
	}
	a.Subq(CX, Constant(1))
	a.Jne(simdloop)

	// check if we have asm loops
	a.Label(nosimdloop)
	a.Movq(CX, h.asmroll)
	a.Orq(CX, CX)
	a.Je(end)

	// apply asm loops
	a.Label(asmloop)
	h.asmtaps(a)
	a.Subq(CX, Constant(1))
	a.Jne(asmloop)

	a.Label(end)
}

func (h *horizontal) taps1(a *Asm, idx int) {
	a.Movwqsx(DX, Address(BX))
	a.Movbqzx(AX, Address(SI, DX, idx))
	a.Movwqsx(DX, Address(BP, idx*2))
	a.Imulq(DX)
}

func (h *horizontal) asmtaps(a *Asm) {
	h.taps1(a, 0)
	a.Movq(h.sum, AX)
	if h.xtaps > 0 {
		i := 1
		for ; i <= h.xtaps-2; i++ {
			h.taps1(a, i)
			a.Addq(h.sum, AX)
		}
		h.taps1(a, i)
		a.Addq(BP, Constant(h.xtaps*2))
	} else {
		a.Movq(AX, h.inner)
		a.Movq(h.count, AX)
		loop := a.NewLabel("loop")
		a.Label(loop)
		h.taps1(a, 1)
		a.Addq(SI, Constant(1))
		a.Addq(BP, Constant(2))
		a.Addq(h.sum, AX)
		a.Subq(h.count, Constant(1))
		a.Jne(loop)
		h.taps1(a, 1)
		a.Addq(BP, Constant(2*2))
		a.Subq(SI, h.inner)
	}
	a.Addq(AX, h.sum)
	a.Addq(AX, Constant(1<<(14-1)))
	a.Cmovql(AX, h.zero)
	a.Shrq(AX, Constant(14))
	a.Cmpq(AX, h.u8max)
	a.Cmovql(AX, h.u8max)
	a.Addq(BX, Constant(xoffset))
	a.Movb(Address(DI), AL)
	a.Addq(DI, Constant(1))
}

func (h *horizontal) load2(a *Asm, op Operand, idx uint) {
	a.Movwqsx(R8, Address(BX, (idx*4+0)*xoffset))
	a.Movwqsx(R9, Address(BX, (idx*4+1)*xoffset))
	a.Movwqsx(R10, Address(BX, (idx*4+2)*xoffset))
	a.Movwqsx(R11, Address(BX, (idx*4+3)*xoffset))
	a.Pinsrw(op, Address(SI, R8), Constant(0))
	a.Pinsrw(op, Address(SI, R9), Constant(1))
	a.Pinsrw(op, Address(SI, R10), Constant(2))
	a.Pinsrw(op, Address(SI, R11), Constant(3))
}

func (h *horizontal) madd(a *Asm, xa, xb, xc, xd SimdRegister, idx uint) {
	a.Punpcklbw(xa, X15)
	a.Pmaddwd(xa, Address(BP, (idx*4+0)*xwidth))
	a.Punpcklbw(xb, X15)
	a.Pmaddwd(xb, Address(BP, (idx*4+1)*xwidth))
	a.Punpcklbw(xc, X15)
	a.Pmaddwd(xc, Address(BP, (idx*4+2)*xwidth))
	a.Punpcklbw(xd, X15)
	a.Pmaddwd(xd, Address(BP, (idx*4+3)*xwidth))
}

func (h *horizontal) taps2(a *Asm) {
	h.load2(a, X0, 0)
	h.load2(a, X1, 1)
	h.load2(a, X2, 2)
	h.load2(a, X3, 3)
	a.Addq(BX, Constant(xwidth*xoffset))
	h.madd(a, X0, X1, X2, X3, 0)
	h.flush(a, X0, X1, X2, X3, BP, 4)
}

func (h *horizontal) flush(a *Asm, xa, xb, xc, xd SimdRegister, op Register, count uint) {
	a.Addq(op, Constant(xwidth*count))
	a.Paddd(xa, X14)
	a.Paddd(xb, X14)
	a.Paddd(xc, X14)
	a.Paddd(xd, X14)
	a.Psrad(xa, Constant(14))
	a.Psrad(xb, Constant(14))
	a.Psrad(xc, Constant(14))
	a.Psrad(xd, Constant(14))
	a.Packssdw(xa, xb)
	a.Packssdw(xc, xd)
	a.Packuswb(xa, xc)
	a.Movou(Address(DI), xa)
	a.Addq(DI, Constant(xwidth))
}

func (h *horizontal) load4(a *Asm, xa, xb SimdRegister, idx uint, tmpa, tmpb SimdRegister) {
	a.Movwqsx(AX, Address(BX, (idx*4+0)*xoffset))
	a.Movwqsx(DX, Address(BX, (idx*4+1)*xoffset))
	a.Movd(xa, Address(SI, AX))
	a.Movd(tmpa, Address(SI, DX))
	a.Movwqsx(AX, Address(BX, (idx*4+2)*xoffset))
	a.Movwqsx(DX, Address(BX, (idx*4+3)*xoffset))
	a.Movd(xb, Address(SI, AX))
	a.Movd(tmpb, Address(SI, DX))
	a.Punpckldq(xa, tmpa)
	a.Punpckldq(xb, tmpb)
}

func (h *horizontal) madd4(a *Asm, xa, xb, xc, xd SimdRegister, idx uint, tmpa, tmpb SimdRegister) {
	h.madd(a, xa, xb, xc, xd, idx)
	a.Movo(tmpa, xa)
	a.Movo(tmpb, xc)
	a.Shufps(tmpa, xb, Constant(0xDD))
	a.Shufps(tmpb, xd, Constant(0xDD))
	a.Shufps(xa, xb, Constant(0x88))
	a.Shufps(xc, xd, Constant(0x88))
	a.Paddd(xa, tmpa)
	a.Paddd(xc, tmpb)
}

func (h *horizontal) taps4(a *Asm) {
	h.load4(a, X0, X1, 0, X8, X9)
	h.load4(a, X2, X3, 1, X10, X11)
	h.load4(a, X4, X5, 2, X12, X13)
	h.load4(a, X6, X7, 3, X8, X9)
	a.Addq(BX, Constant(xwidth*xoffset))
	h.madd4(a, X0, X1, X2, X3, 0, X10, X11)
	h.madd4(a, X4, X5, X6, X7, 1, X12, X13)
	h.flush(a, X0, X2, X4, X6, BP, 8)
}

func (h *horizontal) load8(a *Asm, xa, xb SimdRegister, idx uint, xc, xd SimdRegister) {
	a.Movwqsx(AX, Address(BX, (idx*4+0)*xoffset))
	a.Movq(xa, Address(SI, AX))
	a.Movwqsx(DX, Address(BX, (idx*4+1)*xoffset))
	a.Movq(xb, Address(SI, DX))
	a.Movwqsx(AX, Address(BX, (idx*4+2)*xoffset))
	a.Movq(xc, Address(SI, AX))
	a.Movwqsx(DX, Address(BX, (idx*4+3)*xoffset))
	a.Movq(xd, Address(SI, DX))
}

func (h *horizontal) padd8(a *Asm, xa, xb, xc, xd, tmpa, tmpb SimdRegister) {
	a.Movo(tmpa, xa)
	a.Movo(tmpb, xc)
	a.Punpcklqdq(xa, xb)
	a.Punpckhqdq(tmpa, xb)
	a.Paddd(xa, tmpa)
	a.Punpcklqdq(xc, xd)
	a.Punpckhqdq(tmpb, xd)
	a.Paddd(xc, tmpb)
	a.Movo(tmpa, xa)
	a.Shufps(xa, xc, Constant(0x88))
	a.Shufps(tmpa, xc, Constant(0xDD))
	a.Paddd(xa, tmpa)
}

func (h *horizontal) madd8(a *Asm, xa, xb, xc, xd SimdRegister, idx uint, tmpa, tmpb SimdRegister) {
	h.madd(a, xa, xb, xc, xd, idx)
	h.padd8(a, xa, xb, xc, xd, tmpa, tmpb)
}

func (h *horizontal) taps8(a *Asm) {
	h.load8(a, X0, X1, 0, X2, X3)
	h.load8(a, X4, X5, 1, X6, X7)
	h.load8(a, X8, X9, 2, X10, X11)
	h.madd8(a, X0, X1, X2, X3, 0, X12, X13)
	h.madd8(a, X4, X5, X6, X7, 1, X1, X2)
	h.load8(a, X1, X2, 3, X3, X5)
	a.Addq(BX, Constant(xwidth*xoffset))
	h.madd8(a, X8, X9, X10, X11, 2, X12, X13)
	h.madd8(a, X1, X2, X3, X5, 3, X10, X11)
	h.flush(a, X0, X4, X8, X1, BP, 16)
}

func (h *horizontal) loadn(a *Asm, xa, xb, xc, xd SimdRegister) {
	h.load2(a, xa, 0)
	h.load2(a, xb, 1)
	h.load2(a, xc, 2)
	h.load2(a, xd, 3)
	a.Addq(SI, Constant(2))
}

func (h *horizontal) maddn(a *Asm, xa, xb, xc, xd SimdRegister) {
	h.madd(a, xa, xb, xc, xd, 0)
	a.Addq(BP, Constant(xwidth*4))
}

func (h *horizontal) tapsn(a *Asm) {
	h.loadn(a, X0, X1, X2, X3)
	h.maddn(a, X0, X1, X2, X3)
	// unloop when we know how many taps
	for i := 1; i*2 < h.xtaps; i++ {
		h.loadn(a, X4, X5, X6, X7)
		h.maddn(a, X4, X5, X6, X7)
		a.Paddd(X0, X4)
		a.Paddd(X1, X5)
		a.Paddd(X2, X6)
		a.Paddd(X3, X7)
	}
	if h.xtaps == 0 {
		a.Movq(h.dstref, DI)
		a.Movq(DI, h.inner)
		loop := a.NewLabel("loop")
		a.Label(loop)
		h.loadn(a, X4, X5, X6, X7)
		h.maddn(a, X4, X5, X6, X7)
		a.Paddd(X0, X4)
		a.Paddd(X1, X5)
		a.Paddd(X2, X6)
		a.Paddd(X3, X7)
		a.Subq(DI, Constant(2))
		a.Jne(loop)
		a.Movq(DI, h.dstref)
	}
	a.Movq(AX, h.taps)
	a.Subq(SI, AX)
	h.flush(a, X0, X1, X2, X3, BX, xoffset)
}
