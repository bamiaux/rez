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
	xtaps  int
	xshift uint
	xwidth int
	mmx    bool
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
}

func vgen(a *Asm) {
	v := vertical{}
	v.xshift = 4
	v.xwidth = 1 << v.xshift
	v.zero = a.Data("zero", bytes.Repeat([]byte{0x00}, 16))
	v.hbits = a.Data("hbits", bytes.Repeat([]byte{0x00, 0x00, 0x20, 0x00}, 4))
	v.genscale(a, 2)
}

func (v *vertical) genscale(a *Asm, taps int) {
	v.xtaps = taps
	suffix := "N"
	if taps > 0 {
		suffix = fmt.Sprintf("%v", taps)
	}
	a.NewFunction("v8scale" + suffix)
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
	v.srcref = a.PushStack("srcref")
	v.offref = a.PushStack("offref")
	v.dstoff = a.PushStack("dstoff")
	v.maxroll = a.PushStack("maxroll")
	v.backroll = a.PushStack("backroll")
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
	a.Movq(AX, Address(DX))
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
	a.Andq(DX, Constant(v.xwidth*2-1))
	a.Shrq(CX, Constant(v.xshift+1))
	a.Movq(v.dstoff, BX)
	a.Movq(v.maxroll, CX)
	a.Movq(AX, DX)
	a.Andq(AX, Constant(15))
	norollback := a.NewLabel("norollback")
	a.Je(norollback)
	a.Subq(AX, Constant(v.xwidth*2))
	a.Neg(AX)
	a.Label(norollback)
	a.Movq(v.backroll, AX)
	a.Movq(CX, v.off[0])
	a.Movq(v.offref, CX)
	a.Movo(X14, v.zero)
	a.Movo(X13, v.hbits)
}

func (v *vertical) nextline(a *Asm) {
	a.Addq(DI, v.dstoff)
	a.Addq(BP, Constant(v.xwidth*v.xtaps))
	a.Addq(v.offref, Constant(8))
}

func (v *vertical) line(a *Asm) {
	a.Movq(CX, v.maxroll)
	a.Orq(CX, CX)
	nomaxloop := a.NewLabel("nomaxloop")
	a.Je(nomaxloop)
	maxloop := a.NewLabel("maxloop")
	a.Label(maxloop)
	v.taps2(a)
	a.Subq(CX, Constant(1))
	a.Jne(maxloop)
	a.Label(nomaxloop)
	a.Movq(CX, v.backroll)
	a.Subq(SI, v.backroll)
	a.Subq(DI, v.backroll)
	a.Orq(CX, CX)
	nobackroll := a.NewLabel("nobackroll")
	a.Je(nobackroll)
	v.taps2(a)
	a.Subq(CX, Constant(1))
	a.Label(nobackroll)
}

func (v *vertical) taps2(a *Asm) {
	a.Movou(X12, Address(BP))
	a.Movou(X0, Address(SI, BX, SX0, v.xwidth*0))
	a.Movou(X4, Address(SI, BX, SX0, v.xwidth*1))
	a.Movou(X3, Address(SI, BX, SX1, v.xwidth*0))
	a.Movou(X7, Address(SI, BX, SX1, v.xwidth*1))
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
	a.Punpcklbw(X0, X14)
	a.Punpckhbw(X1, X14)
	a.Punpcklbw(X4, X14)
	a.Punpckhbw(X5, X14)
	a.Punpcklbw(X2, X14)
	a.Punpckhbw(X3, X14)
	a.Punpcklbw(X6, X14)
	a.Punpckhbw(X7, X14)
	a.Pmaddwd(X0, X12)
	a.Pmaddwd(X1, X12)
	a.Pmaddwd(X4, X12)
	a.Pmaddwd(X5, X12)
	a.Pmaddwd(X2, X12)
	a.Pmaddwd(X3, X12)
	a.Pmaddwd(X6, X12)
	a.Pmaddwd(X7, X12)
	a.Paddd(X0, X13)
	a.Paddd(X1, X13)
	a.Paddd(X4, X13)
	a.Paddd(X5, X13)
	a.Paddd(X2, X13)
	a.Paddd(X3, X13)
	a.Paddd(X6, X13)
	a.Paddd(X7, X13)
	a.Psrad(X0, Constant(14))
	a.Psrad(X1, Constant(14))
	a.Psrad(X4, Constant(14))
	a.Psrad(X5, Constant(14))
	a.Psrad(X2, Constant(14))
	a.Psrad(X3, Constant(14))
	a.Psrad(X6, Constant(14))
	a.Psrad(X7, Constant(14))
	a.Packssdw(X0, X1)
	a.Packssdw(X4, X5)
	a.Packssdw(X2, X3)
	a.Packssdw(X6, X7)
	a.Packuswb(X0, X2)
	a.Packuswb(X4, X6)
	a.Movou(Address(DI, v.xwidth*0), X0)
	a.Movou(Address(DI, v.xwidth*1), X4)
	a.Addq(SI, Constant(v.xwidth*2))
	a.Addq(DI, Constant(v.xwidth*2))
}