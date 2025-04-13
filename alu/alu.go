package alu

import (
	utils "github.com/mikzorz/goboy-emu/helpers"
)

type ALU struct {
	ALUBusy bool
	adj     byte // sign of e8
}

func (alu *ALU) ALUInc(val byte) (result, hc byte) {
	result, hc, _ = alu.ALUAdd(val, 1, 0)
	return
}

func (alu *ALU) ALUDec(val byte) (result, hc byte) {
	result, hc, _ = alu.ALUSub(val, 1, 0)
	return
}

func (alu *ALU) ALUAdd(a, b, carry byte) (result, hc, c byte) {
	lo, hc := alu.addNibbles(lsn(a), lsn(b), carry)
	hi, c := alu.addNibbles(msn(a), msn(b), hc)
	result = joinNibbles(hi, lo)

	return
}

func (alu *ALU) ALUSub(a, b, carry byte) (result, hc, c byte) {
	lo, hc := alu.subNibbles(lsn(a), lsn(b), carry)
	hi, c := alu.subNibbles(msn(a), msn(b), hc)
	result = joinNibbles(hi, lo)

	return
}

func (alu *ALU) ALUAnd(a, b byte) (result byte) {
	return a & b
}

func (alu *ALU) AddSignedToUnsigned(a, e byte) (result, hc, c byte) {
	alu.adj = e >> 7
	result, hc, c = alu.ALUAdd(a, e, 0)

	return
}

// Adjust for signed integer addition
func (alu *ALU) Adjust(a, c byte) (result byte) {
	return a + c - alu.adj
}

// Decimal Adjust
// a = byte to adjust
// f = flags
func (alu *ALU) DecAdj(a, f byte) (result, carry byte) {
	neg := utils.IsBitSet(6, f)
	hc := utils.IsBitSet(5, f)
	c := utils.IsBitSet(4, f)
	result = a
	offset := byte(0)

	if c || (result > 0x99 && !neg) {
		offset += 0x60
		carry = 1
	}
	if hc || (lsn(result) > 0x9 && !neg) {
		offset += 0x6
	}

	if neg {
		result -= offset
	} else {
		result += offset
	}

	return
}

func (alu *ALU) ALUSwap(a byte) byte {
	return joinNibbles(lsn(a), msn(a))
}

func (alu *ALU) addNibbles(a, b, c byte) (result, carry byte) {
	result = a + b + c
	carry = (result & 0x10) >> 4
	result &= 0xF

	return
}

func (alu *ALU) subNibbles(a, b, c byte) (result, carry byte) {
	result = a - b - c
	carry = (result & 0x10) >> 4
	result &= 0xF

	return
}

// Copied from helpers.go. Can I put helpers in a shared package?
func joinNibbles(msn, lsn byte) byte {
	return (msn << 4) | lsn
}

// least significant nibble
func lsn(n byte) byte {
	return n & 0xF
}

// most significant nibble
func msn(n byte) byte {
	return n >> 4
}
