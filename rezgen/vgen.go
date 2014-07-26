// Copyright 2014 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	. "github.com/bamiaux/rez/asm"
)

type vertical struct {
	xtaps int
	// global data
	zero  Operand
	hbits Operand
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
	srcref   Operand
	offref   Operand
	dstoff   Operand
	maxroll  Operand
	backroll Operand
	inner    Operand
}

func vgen(a *Asm) {
	v := vertical{}
	v.zero = a.Data("zero", bytes.Repeat([]byte{0x00}, 16))
	v.hbits = a.Data("hbits", bytes.Repeat([]byte{0x00, 0x00, 0x20, 0x00}, 4))
	v.genscale(a, 2)
	v.genscale(a, 4)
	v.genscale(a, 6)
	v.genscale(a, 8)
	v.genscale(a, 10)
	v.genscale(a, 12)
	v.genscale(a, 0)
}

func (v *vertical) genscale(a *Asm, taps int) {
	v.xtaps = taps
	suffix := "N"
	if taps > 0 {
		suffix = fmt.Sprintf("%v", taps)
	}
	a.NewFunction("v8scale" + suffix + "Amd64")
	// arguments
	v.dst = a.SliceArgument("dst")
	v.src = a.SliceArgument("src")
	v.cof = a.SliceArgument("cof")
	v.off = a.SliceArgument("off")
	v.taps = a.Argument("taps")
	v.width = a.Argument("width")
	v.height = a.Argument("height")
	v.dp = a.Argument("dp")
	v.sp = a.Argument("sp")
	// stack
	v.srcref = R9
	v.offref = R10
	v.dstoff = R11
	v.maxroll = R12
	v.backroll = R13
	if v.xtaps == 0 {
		v.inner = R14
	}
	a.Start()
	v.frame(a)
	a.Ret()
}

func (v *vertical) frame(a *Asm) {
	v.setup(a)
	a.Movq(SI, v.src[0])
	a.Movq(v.srcref, SI)
	a.Movq(DI, v.dst[0])
	a.Movq(BP, v.cof[0])
	a.Movq(BX, v.sp)
	yloop := a.NewLabel("yloop")
	a.Label(yloop)
	a.Movq(SI, v.srcref)
	a.Movq(DX, v.offref)
	a.Movwqsx(AX, Address(DX))
	a.Mulq(BX)
	a.Addq(SI, AX)
	a.Movq(v.srcref, SI)
	v.line(a)
	v.nextline(a)
	a.Subq(v.height, Constant(1))
	a.Jne(yloop)
}

func (v *vertical) setup(a *Asm) {
	a.Movq(BX, v.dp)
	a.Movq(CX, v.width)
	a.Movq(DX, CX)
	a.Subq(BX, CX)
	a.Andq(DX, Constant(xwidth-1))
	a.Shrq(CX, Constant(xshift))
	a.Movq(v.dstoff, BX)
	a.Movq(v.maxroll, CX)
	norollback := a.NewLabel("norollback")
	a.Movq(AX, DX)
	a.Orq(AX, AX)
	a.Je(norollback)
	a.Subq(DX, Constant(xwidth))
	a.Neg(DX)
	a.Label(norollback)
	a.Movq(v.backroll, DX)
	a.Movq(CX, v.off[0])
	a.Movq(v.offref, CX)
	a.Movo(X14, v.zero)
	a.Movo(X13, v.hbits)
	if v.xtaps == 0 {
		a.Movq(DX, v.taps)
		a.Subq(DX, Constant(4))
		a.Shrq(DX, Constant(1))
		a.Movq(v.inner, DX)
	}
}

func (v *vertical) nextline(a *Asm) {
	a.Addq(DI, v.dstoff)
	if v.xtaps == 0 {
		a.Movq(DX, v.taps)
		a.Shlq(DX, Constant(xshift))
		a.Addq(BP, DX)
	} else {
		a.Addq(BP, Constant(xwidth*v.xtaps))
	}
	a.Addq(v.offref, Constant(xoffset))
}

func (v *vertical) line(a *Asm) {
	taps := v.tapsn
	if v.xtaps == 2 {
		taps = v.taps2
	}
	a.Movq(CX, v.maxroll)
	a.Orq(CX, CX)
	nomaxloop := a.NewLabel("nomaxloop")
	a.Je(nomaxloop)
	maxloop := a.NewLabel("maxloop")
	a.Label(maxloop)
	taps(a)
	a.Subq(CX, Constant(1))
	a.Jne(maxloop)
	a.Label(nomaxloop)
	a.Movq(CX, v.backroll)
	a.Subq(SI, v.backroll)
	a.Subq(DI, v.backroll)
	a.Orq(CX, CX)
	nobackroll := a.NewLabel("nobackroll")
	a.Je(nobackroll)
	taps(a)
	a.Label(nobackroll)
}

func (v *vertical) taps2(a *Asm) {
	a.Movou(X12, Address(BP))
	a.Movou(X0, Address(SI, BX, SX0))
	a.Movou(X3, Address(SI, BX, SX1))
	a.Movo(X2, X0)
	a.Punpcklbw(X0, X3)
	a.Punpckhbw(X2, X3)
	a.Movo(X1, X0)
	a.Movo(X3, X2)
	a.Punpcklbw(X0, X14)
	a.Punpckhbw(X1, X14)
	a.Punpcklbw(X2, X14)
	a.Punpckhbw(X3, X14)
	a.Pmaddwd(X0, X12)
	a.Pmaddwd(X1, X12)
	a.Pmaddwd(X2, X12)
	a.Pmaddwd(X3, X12)
	a.Paddd(X0, X13)
	a.Paddd(X1, X13)
	a.Paddd(X2, X13)
	a.Paddd(X3, X13)
	a.Psrad(X0, Constant(14))
	a.Psrad(X1, Constant(14))
	a.Psrad(X2, Constant(14))
	a.Psrad(X3, Constant(14))
	a.Packssdw(X0, X1)
	a.Packssdw(X2, X3)
	a.Packuswb(X0, X2)
	a.Movou(Address(DI), X0)
	a.Addq(SI, Constant(xwidth))
	a.Addq(DI, Constant(xwidth))
}

func (v *vertical) tapsn(a *Asm) {
	if v.xtaps != 4 {
		a.Leaq(AX, Address(SI, BX, SX4))
	}
	v.tapsn4(a)
	if v.xtaps == 0 {
		v.leftntaps(a)
	} else if v.xtaps != 4 {
		v.left2taps(a)
	}
	a.Paddd(X0, X13)
	a.Paddd(X1, X13)
	a.Paddd(X2, X13)
	a.Paddd(X3, X13)
	a.Psrad(X0, Constant(14))
	a.Psrad(X1, Constant(14))
	a.Psrad(X2, Constant(14))
	a.Psrad(X3, Constant(14))
	a.Packssdw(X0, X1)
	a.Packssdw(X2, X3)
	a.Packuswb(X0, X2)
	a.Movou(Address(DI), X0)
	a.Addq(SI, Constant(xwidth))
	a.Addq(DI, Constant(xwidth))
}

func (v *vertical) tapsn4(a *Asm) {
	a.Movou(X0, Address(SI, BX, SX0))
	a.Movou(X3, Address(SI, BX, SX1))
	a.Movou(X4, Address(SI, BX, SX2))
	a.Movou(X10, Address(BP))
	a.Movou(X11, Address(BP, xwidth*2))
	a.Addq(SI, BX)
	a.Movou(X7, Address(SI, BX, SX2))
	a.Movo(X2, X0)
	a.Movo(X6, X4)
	a.Punpcklbw(X0, X3)
	a.Punpcklbw(X4, X7)
	a.Punpckhbw(X2, X3)
	a.Punpckhbw(X6, X7)
	a.Movo(X1, X0)
	a.Movo(X5, X4)
	a.Movo(X3, X2)
	a.Movo(X7, X6)
	a.Subq(SI, BX)
	a.Punpcklbw(X0, X14)
	a.Punpckhbw(X1, X14)
	a.Punpcklbw(X4, X14)
	a.Punpckhbw(X5, X14)
	a.Punpcklbw(X2, X14)
	a.Punpckhbw(X3, X14)
	a.Punpcklbw(X6, X14)
	a.Punpckhbw(X7, X14)
	a.Pmaddwd(X0, X10)
	a.Pmaddwd(X1, X10)
	a.Pmaddwd(X4, X11)
	a.Pmaddwd(X5, X11)
	a.Pmaddwd(X2, X10)
	a.Pmaddwd(X3, X10)
	a.Pmaddwd(X6, X11)
	a.Pmaddwd(X7, X11)
	a.Paddd(X0, X4)
	a.Paddd(X1, X5)
	a.Paddd(X2, X6)
	a.Paddd(X3, X7)
}

func (v *vertical) left2taps(a *Asm) {
	for i := 2; i*2 < v.xtaps; i++ {
		v.tapsn2(a, X4, X5, X6, X7, AX, Address(BP, i*xwidth*2))
		if i*2+1 < v.xtaps {
			a.Leaq(AX, Address(AX, BX, SX2))
		}
		a.Paddd(X0, X4)
		a.Paddd(X1, X5)
		a.Paddd(X2, X6)
		a.Paddd(X3, X7)
	}
}

func (v *vertical) leftntaps(a *Asm) {
	a.Movq(R15, v.inner)
	a.Movq(DX, BP)
	a.Addq(DX, Constant(xwidth*2))
	innerloop := a.NewLabel("innerloop")
	a.Label(innerloop)
	a.Addq(DX, Constant(xwidth*2))
	v.tapsn2(a, X4, X5, X6, X7, AX, Address(DX))
	a.Leaq(AX, Address(AX, BX, SX2))
	a.Paddd(X0, X4)
	a.Paddd(X1, X5)
	a.Paddd(X2, X6)
	a.Paddd(X3, X7)
	a.Subq(R15, Constant(1))
	a.Jne(innerloop)
}

func (v *vertical) tapsn2(a *Asm, xa, xb, xc, xd SimdRegister, src Register, cof Operand) {
	a.Movou(xa, Address(src, BX, SX0))
	a.Movou(xd, Address(src, BX, SX1))
	a.Movo(xc, xa)
	a.Punpcklbw(xa, xd)
	a.Punpckhbw(xc, xd)
	a.Movo(xb, xa)
	a.Movo(xd, xc)
	a.Punpcklbw(xa, X14)
	a.Punpckhbw(xb, X14)
	a.Punpcklbw(xc, X14)
	a.Punpckhbw(xd, X14)
	a.Pmaddwd(xa, cof)
	a.Pmaddwd(xb, cof)
	a.Pmaddwd(xc, cof)
	a.Pmaddwd(xd, cof)
}
