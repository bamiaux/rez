// Copyright 2014 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package asm

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Asm struct {
	w      *bufio.Writer
	data   int
	label  int
	errors []string
	// per function
	name  string
	args  int
	stack int
}

func NewAsm(w io.Writer) *Asm {
	return &Asm{
		w: bufio.NewWriter(w),
	}
}

func (a *Asm) NewFunction(name string) {
	a.name = name
	a.args = 0
	a.stack = 0
}

func (a *Asm) Data(name string, data []byte) Operand {
	if len(data)&7 != 0 {
		return nil
	}
	name = fmt.Sprintf("%v_%v<>", name, a.data)
	a.data++
	for i := 0; i < len(data); i += 8 {
		a.write(fmt.Sprintf("DATA\t%v+0x%02X(SB)/8, $0x%016X", name, i, data[i:i+8]))
	}
	// RODATA should be included with textflag.h
	// it only works with go > 1.4 though
	const RODATA = 8
	a.write(fmt.Sprintf("GLOBL\t%v(SB), %v, $%v", name, RODATA, len(data)))
	return literal(fmt.Sprintf("%v(SB)", name))
}

type Argument struct {
	name   string
	offset int
}

func (s *Argument) String() string {
	return fmt.Sprintf("%v+%v(FP)", s.name, s.offset)
}

func (a *Asm) Argument(name string) Operand {
	a.args += 8
	return &Argument{
		name:   name,
		offset: a.args - 8,
	}
}

func (a *Asm) SliceArgument(name string) []Operand {
	rpy := []Operand{}
	for i := 0; i < 3; i++ {
		rpy = append(rpy, a.Argument(name))
	}
	return rpy
}

type StackOperand struct {
	name   string
	offset int
}

func (s *StackOperand) String() string {
	return fmt.Sprintf("%v+-%v(SP)", s.name, s.offset)
}

func (a *Asm) PushStack(name string) Operand {
	a.stack += 8
	return &StackOperand{
		name:   name,
		offset: a.stack,
	}
}

func (a *Asm) Start() {
	a.write(fmt.Sprintf("\nTEXT ·%v(SB),4,$%v-%v", a.name, a.stack, a.args))
}

func (a *Asm) Flush() error {
	err := a.w.Flush()
	a.save(err)
	return a.getErrors()
}

func (a *Asm) save(err error) {
	if err == nil {
		return
	}
	a.errors = append(a.errors, err.Error())
}

func (a *Asm) getErrors() error {
	if len(a.errors) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(a.errors, "\n"))
}

func (a *Asm) write(msg string) {
	_, err := a.w.WriteString(msg + "\n")
	a.save(err)
}

type Operand interface {
	String() string
}

type literal string

func (lit literal) String() string {
	return string(lit)
}

func Constant(value interface{}) Operand {
	return literal(fmt.Sprintf("$%v", value))
}

type Register struct{ literal }

type Scale uint

const (
	SX0 Scale = 0
	SX1 Scale = 1 << (iota - 1)
	SX2
	SX4
	SX8
)

func address(base Register) Operand {
	return literal(fmt.Sprintf("(%v)", base.String()))
}

func displaceaddress(base Register, index int) Operand {
	if index == 0 {
		return address(base)
	}
	return literal(fmt.Sprintf("%v(%v)", index, base.String()))
}

func scaledindex(index Register, scale Scale) string {
	if scale == SX0 {
		return ""
	}
	return fmt.Sprintf("(%v*%v)", index.String(), scale)
}

func indexaddress(base Register, index Register, scale Scale) Operand {
	return literal(fmt.Sprintf("(%v)%v", base.String(), scaledindex(index, scale)))
}

func fulladdress(base Register, index Register, scale Scale, displacement int) Operand {
	d := ""
	if displacement != 0 {
		d = fmt.Sprintf("%v", displacement)
	}
	return literal(fmt.Sprintf("%v(%v)%v", d, base.String(), scaledindex(index, scale)))
}

func Address(base Register, offsets ...interface{}) Operand {
	// happily panics if not given expected input
	switch len(offsets) {
	case 0:
		return address(base)
	case 1:
		switch t := offsets[0].(type) {
		case int:
			return displaceaddress(base, t)
		case uint:
			return displaceaddress(base, int(t))
		case Register:
			return indexaddress(base, t, SX1)
		case Scale:
			return literal(scaledindex(base, t))
		}
	case 2:
		index, ok := offsets[0].(Register)
		if !ok {
			break
		}
		switch t := offsets[1].(type) {
		case int:
			return fulladdress(base, index, SX1, t)
		case uint:
			return fulladdress(base, index, SX1, int(t))
		case Scale:
			return indexaddress(base, index, t)
		}
	case 3:
		index, ok := offsets[0].(Register)
		if !ok {
			break
		}
		scale, ok := offsets[1].(Scale)
		if !ok {
			break
		}
		switch t := offsets[2].(type) {
		case int:
			return fulladdress(base, index, scale, t)
		case uint:
			return fulladdress(base, index, scale, int(t))
		}
	}
	panic("unexpected input")
}

type SimdRegister struct{ literal }

var (
	SP  = Register{literal: "SP"}
	AX  = Register{literal: "AX"}
	AH  = Register{literal: "AH"}
	AL  = Register{literal: "AL"}
	BX  = Register{literal: "BX"}
	BH  = Register{literal: "BH"}
	BL  = Register{literal: "BL"}
	CX  = Register{literal: "CX"}
	CH  = Register{literal: "CH"}
	CL  = Register{literal: "CL"}
	DX  = Register{literal: "DX"}
	DH  = Register{literal: "DH"}
	DL  = Register{literal: "DL"}
	BP  = Register{literal: "BP"}
	DI  = Register{literal: "DI"}
	SI  = Register{literal: "SI"}
	R8  = Register{literal: "R8"}
	R9  = Register{literal: "R9"}
	R10 = Register{literal: "R10"}
	R11 = Register{literal: "R11"}
	R12 = Register{literal: "R12"}
	R13 = Register{literal: "R13"}
	R14 = Register{literal: "R14"}
	R15 = Register{literal: "R15"}
	X0  = SimdRegister{literal: "X0"}
	X1  = SimdRegister{literal: "X1"}
	X2  = SimdRegister{literal: "X2"}
	X3  = SimdRegister{literal: "X3"}
	X4  = SimdRegister{literal: "X4"}
	X5  = SimdRegister{literal: "X5"}
	X6  = SimdRegister{literal: "X6"}
	X7  = SimdRegister{literal: "X7"}
	X8  = SimdRegister{literal: "X8"}
	X9  = SimdRegister{literal: "X9"}
	X10 = SimdRegister{literal: "X10"}
	X11 = SimdRegister{literal: "X11"}
	X12 = SimdRegister{literal: "X12"}
	X13 = SimdRegister{literal: "X13"}
	X14 = SimdRegister{literal: "X14"}
	X15 = SimdRegister{literal: "X15"}
)

type label string

func (a *Asm) NewLabel(name string) label {
	idx := a.label
	a.label++
	return label(fmt.Sprintf("%v_%v", name, idx))
}

func (l label) String() string {
	return string(l)
}

func (a *Asm) op0(instruction string) {
	a.write("\t\t" + instruction)
}

func (a *Asm) op1(instruction string, opa Operand) {
	a.write("\t\t" + instruction + "\t" + opa.String())
}

func (a *Asm) op2(instruction string, opa, opb Operand) {
	a.write("\t\t" + instruction + "\t" + opb.String() + ", " + opa.String())
}

func (a *Asm) op3(instruction string, opa, opb, opc Operand) {
	a.write(fmt.Sprintf("\t\t%v\t%v, %v, %v", instruction, opc.String(), opb.String(), opa.String()))
}

func (a *Asm) Label(name label) {
	a.write(name.String() + ":")
}

func (a *Asm) Ret() { a.op0("RET") }

func (a *Asm) Imulq(op Operand) { a.op1("IMULQ", op) }
func (a *Asm) Incq(op Operand)  { a.op1("INCQ", op) }
func (a *Asm) Je(name label)    { a.op1("JE", name) }
func (a *Asm) Jmp(name label)   { a.op1("JMP", name) }
func (a *Asm) Jne(name label)   { a.op1("JNE", name) }
func (a *Asm) Mulq(op Operand)  { a.op1("MULQ", op) }
func (a *Asm) Neg(op Operand)   { a.op1("NEGQ", op) }

func (a *Asm) Addq(opa, opb Operand)       { a.op2("ADDQ", opa, opb) }
func (a *Asm) Andq(opa, opb Operand)       { a.op2("ANDQ", opa, opb) }
func (a *Asm) Cmovql(opa, opb Operand)     { a.op2("CMOVQLT", opa, opb) }
func (a *Asm) Cmpq(opa, opb Operand)       { a.op2("CMPQ", opa, opb) }
func (a *Asm) Leaq(opa, opb Operand)       { a.op2("LEAQ", opa, opb) }
func (a *Asm) Movb(opa, opb Operand)       { a.op2("MOVB", opa, opb) }
func (a *Asm) Movbqzx(opa, opb Operand)    { a.op2("MOVBQZX", opa, opb) }
func (a *Asm) Movd(opa, opb Operand)       { a.op2("MOVL", opa, opb) }
func (a *Asm) Movo(opa, opb Operand)       { a.op2("MOVO", opa, opb) }
func (a *Asm) Movou(opa, opb Operand)      { a.op2("MOVOU", opa, opb) }
func (a *Asm) Movq(opa, opb Operand)       { a.op2("MOVQ", opa, opb) }
func (a *Asm) Movwqsx(opa, opb Operand)    { a.op2("MOVWQSX", opa, opb) }
func (a *Asm) Orq(opa, opb Operand)        { a.op2("ORQ", opa, opb) }
func (a *Asm) Packssdw(opa, opb Operand)   { a.op2("PACKSSLW", opa, opb) }
func (a *Asm) Packuswb(opa, opb Operand)   { a.op2("PACKUSWB", opa, opb) }
func (a *Asm) Paddd(opa, opb Operand)      { a.op2("PADDL", opa, opb) }
func (a *Asm) Pmaddwd(opa, opb Operand)    { a.op2("PMADDWL", opa, opb) }
func (a *Asm) Psrad(opa, opb Operand)      { a.op2("PSRAL", opa, opb) }
func (a *Asm) Punpckhbw(opa, opb Operand)  { a.op2("PUNPCKHBW", opa, opb) }
func (a *Asm) Punpckhqdq(opa, opb Operand) { a.op2("PUNPCKHQDQ", opa, opb) }
func (a *Asm) Punpcklbw(opa, opb Operand)  { a.op2("PUNPCKLBW", opa, opb) }
func (a *Asm) Punpckldq(opa, opb Operand)  { a.op2("PUNPCKLLQ", opa, opb) }
func (a *Asm) Punpcklqdq(opa, opb Operand) { a.op2("PUNPCKLQDQ", opa, opb) }
func (a *Asm) Pxor(opa, opb Operand)       { a.op2("PXOR", opa, opb) }
func (a *Asm) Shlq(opa, opb Operand)       { a.op2("SHLQ", opa, opb) }
func (a *Asm) Shrq(opa, opb Operand)       { a.op2("SHRQ", opa, opb) }
func (a *Asm) Subq(opa, opb Operand)       { a.op2("SUBQ", opa, opb) }

func (a *Asm) Pinsrw(opa, opb, opc Operand) { a.op3("PINSRW", opa, opb, opc) }
func (a *Asm) Shufps(opa, opb, opc Operand) { a.op3("SHUFPS", opa, opb, opc) }
