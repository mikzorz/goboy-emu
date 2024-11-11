package main

import (
// "fmt"
// "log"
)

type ALU struct {
}

type IDU struct {
}

type RegisterFile struct {
	IR, IE, A, F       uint8
	BC, DE, HL, PC, SP uint16 // may need to set sp to start at top of stack range
}

type ControlUnit struct {
}

type CPU struct {
	ALU
	IDU
	RegisterFile
	ControlUnit
	speed int // 4194304 Hz
}

func NewCPU() CPU {
	return CPU{
		ALU{},
		IDU{},
		// RegisterFile{PC: 0xFF},
		RegisterFile{PC: 0x150}, // After header
		ControlUnit{},
		4194304,
	}
}

// read bank from romData, write to ram[destRegion * 0x4000]
// func readBank(bank, destRegion int) {
// 	if destRegion != 0 || destRegion != 1 {
// 		fmt.Errorf("destRegion must be either 0 or 1, given %d", destRegion)
// 	}
// 	bankStart := bank * 0x4000
// 	destStart := destRegion * 0x4000
// 	for i := 0; i < 0x4000; i++ {
// 		ram[destStart+i] = romData[bankStart+i]
// 	}
// }

// func (c CPU) ReadOpCode() {
// 	cpu.IR = readByte()
// 	fmt.Printf("Ram[%04X] Op: %02X\n", cpu.PC-1, cpu.IR)
//
// 	switch cpu.IR {
// 	case 0x00:
// 		// NOP
// 		// todo?
// 	case 0x01:
// 		// Load imm 16bit into BC
// 		cpu.BC = imm16()
// 	case 0x02:
// 		// LD [BC], A
// 		ram[cpu.BC] = cpu.A
// 	case 0x03:
// 		// Inc BC
// 		cpu.BC++
// 	case 0x05:
// 		// DEC B
// 		B := msb(cpu.BC)
// 		cpu.BC = joinBytes(B-1, lsb(cpu.BC))
// 		setZFlag(B - 1)
// 		setNegFlag(1)
// 		setHalfCarrySub(B, 1)
// 	case 0x06:
// 		// LD B, n8
// 		cpu.BC = joinBytes(readByte(), lsb(cpu.BC))
// 	case 0x08:
// 		// Load SP into [a16]
// 		lsb := lsb(cpu.SP)
// 		msb := msb(cpu.SP)
// 		nn := imm16()
// 		ram[nn] = lsb
// 		ram[nn+1] = msb
// 	case 0x0A:
// 		// LD A, [BC]
// 		cpu.A = ram[cpu.BC]
// 	case 0x0B:
// 		// DEC BC
// 		cpu.BC--
// 	case 0x0C:
// 		// INC C
// 		C := lsb(cpu.BC)
// 		cpu.BC = joinBytes(msb(cpu.BC), C+1)
// 		setZFlag(C + 1)
// 		setNegFlag(0)
// 		setHalfCarryAdd(C, 1)
// 	case 0x0E:
// 		// LD C, n8
// 		cpu.BC = joinBytes(msb(cpu.BC), readByte())
// 	case 0x10:
// 		// STOP n8
// 		// todo
// 		ram[0xFF04] = 0
// 	case 0x11:
// 		// Load nn into DE
// 		cpu.DE = imm16()
// 	case 0x12:
// 		// Load [DE], A
// 		ram[cpu.DE] = cpu.A
// 	case 0x13:
// 		// INC DE
// 		cpu.DE++
// 	case 0x18:
// 		// Jump forward by imm8
// 		e := readByte()
// 		cpu.PC = addInt8ToUint16(cpu.PC, e) + 1
// 	case 0x1A:
// 		// LD A, [DE]
// 		cpu.A = ram[cpu.DE]
// 		// case 0x1F:
// 		//   // RRA, rotate A to right (todo, add carry to msb?)
// 		//   setCarry(cpu.A & 0x1)
// 		//   cpu.A >>= 1
// 	case 0x20:
// 		// Jump by e8 if z == 0
// 		e := readByte()
// 		if !bit(7, cpu.F) {
// 			// log.Println("cpu.PC = ", cpu.PC)
// 			// log.Println("e = ", e)
// 			cpu.PC = addInt8ToUint16(cpu.PC, e) + 1
// 			// log.Println("cpu.PC + e = ", cpu.PC)
// 		}
// 	case 0x21:
// 		// Load n16 into HL
// 		lsb := readByte()
// 		msb := readByte()
// 		nn := joinBytes(msb, lsb)
// 		cpu.HL = nn
// 	case 0x22:
// 		// Load A into [HL], inc HL
// 		ram[cpu.HL] = cpu.A
// 		cpu.HL++
// 	case 0x28:
// 		// Jump by imm8 if Z == 1
// 		e := readByte()
// 		if bit(7, cpu.F) {
// 			cpu.PC = addInt8ToUint16(cpu.PC, e) + 1
// 		}
// 		// case 0x29:
// 	// Add HL to HL
// 	// todo
// 	case 0x2A:
// 		// LD A, [HL+]
// 		cpu.A = ram[cpu.HL]
// 		cpu.HL++
// 	case 0x2B:
// 		// Dec HL
// 		cpu.HL--
// 	case 0x31:
// 		// Load imm 16bit data into SP
// 		cpu.SP = imm16()
// 	case 0x39:
// 		// ADD HL, SP
// 		var z byte = 0
// 		if bit(7, cpu.F) {
// 			z = 1
// 		}
// 		cpu.HL = addAndSetFlags16(cpu.HL, cpu.SP)
// 		setZFlag(z)
// 	case 0x3E:
// 		// Load imm8 into A
// 		cpu.A = readByte()
// 	case 0x44:
// 		// LD B, H
// 		cpu.BC = joinBytes(msb(cpu.HL), lsb(cpu.BC))
// 	// case 0x47:
// 	// Load A into B
// 	// todo
// 	// case 0x48:
// 	// Load B into C
// 	// todo
// 	case 0x49:
// 	// Load C into C
// 	// todo
// 	case 0x53:
// 		// LD D, E
// 		cpu.DE = joinBytes(lsb(cpu.DE), lsb(cpu.DE))
// 	case 0x57:
// 		// LD D, A
// 		cpu.DE = joinBytes(cpu.A, lsb(cpu.DE))
// 	case 0x58:
// 		// Load B into E
// 		cpu.DE = joinBytes(msb(cpu.DE), msb(cpu.BC))
// 	case 0x60:
// 		// LD H, B
// 		cpu.HL = joinBytes(msb(cpu.BC), lsb(cpu.HL))
// 	// case 0x68:
// 	//   // Load B into L
// 	//   B := (cpu.BC & 0xF0) >> 4
// 	//   cpu.HL &= 0xF0
// 	//   cpu.HL |= B
// 	case 0x6B:
// 		// Load E into L
// 		cpu.HL = joinBytes(msb(cpu.HL), lsb(cpu.DE))
// 	case 0x6D:
// 	// Load L into L
// 	// todo, maybe something to do with interrupts?
// 	case 0x78:
// 		// Load B into A
// 		cpu.A = msb(cpu.BC)
// 	case 0x7A:
// 		// LD A, D
// 		cpu.A = msb(cpu.DE)
// 	case 0x7C:
// 		// LD A, H
// 		cpu.A = msb(cpu.HL)
// 	case 0x7D:
// 		// LD A, L
// 		cpu.A = lsb(cpu.HL)
// 	case 0x85:
// 		// Add L to A, store result in A
// 		cpu.A = addAndSetFlags(cpu.A, lsb(cpu.HL))
// 	// case 0x8E:
// 	// Add ram[HL] with carry to A
// 	// todo
// 	case 0x90:
// 		// Sub B from A
// 		cpu.A = subAndSetFlags(cpu.A, msb(cpu.BC))
// 	// case 0x9D:
// 	// Substract L and carry flag from A
// 	// todo
// 	case 0xAF:
// 		// XOR reg A with itself
// 		cpu.A ^= cpu.A
// 		clearFlags()
// 		setZFlag(1)
// 	case 0xB1:
// 		// OR A C
// 		C := lsb(cpu.BC)
// 		result := cpu.A | C
// 		clearFlags()
// 		setZFlag(result)
// 		cpu.A = result
// 	case 0xC3:
// 		// JP a16
// 		cpu.PC = imm16()
// 	case 0xC8:
// 		// RET Z
// 		if bit(7, cpu.F) {
// 			cpu.PC = popFromStack16()
// 		}
// 	case 0xC9:
// 		// RET
// 		cpu.PC = popFromStack16()
// 	case 0xCB:
// 		// PREFIX
// 		// BIG TODO
// 		cpu.IR = readByte()
// 		fmt.Printf("Ram[%04X] Op: %02X\n", cpu.PC-1, cpu.IR)
// 		switch cpu.IR {
// 		case 0x37:
// 			// SWAP A
// 			// todo
// 		case 0x87:
// 			// Reset bit 0 in A
// 			cpu.A &= 0xFE
// 		default:
// 			log.Fatalf("unimplemented prefixed opcode %02X", cpu.IR)
// 		}
// 	case 0xCD:
// 		// Call function at address imm16
// 		lsb := readByte()
// 		msb := readByte()
// 		pushToStack16(cpu.PC)
// 		nn := joinBytes(msb, lsb)
// 		cpu.PC = nn
// 	case 0xE0:
// 		// Load A into ram[0xFF00 + n]
// 		ram[0xFF00|uint16(readByte())] = cpu.A
// 	case 0xE2:
// 		// LD [C], A
// 		ram[lsb(cpu.BC)] = cpu.A
// 	case 0xE6:
// 		// AND A n8
// 		cpu.A &= readByte()
// 		clearFlags()
// 		setZFlag(cpu.A)
// 		setHalfCarry(1)
// 	case 0xE7:
// 		// Call 0x20
// 		pushToStack16(cpu.PC)
// 		cpu.PC = 0x0020
// 	case 0xEA:
// 		// Load A into ram[nn]
// 		lsb := readByte()
// 		msb := readByte()
// 		nn := joinBytes(msb, lsb)
// 		ram[nn] = cpu.A
// 	case 0xF0:
// 		// Load [a8] into A
// 		cpu.A = ram[0xFF00|uint16(readByte())]
// 	case 0xFE:
// 		// Compare A with n. Sub n from A and update flags. Do not update A.
// 		n := readByte()
// 		subAndSetFlags(cpu.A, n)
// 	case 0xFF:
// 		// Call 0x38
// 		pushToStack16(cpu.PC)
// 		cpu.PC = 0x0038
// 	default:
// 		log.Fatalf("unimplemented opcode %02X", cpu.IR)
// 	}
//
// }

// func readByte() byte {
// 	b := ram[cpu.PC]
// 	cpu.PC++
// 	return b
// }
//
// func pushToStack16(data uint16) {
// 	cpu.SP--
// 	ram[cpu.SP] = msb(data)
// 	cpu.SP--
// 	ram[cpu.SP] = lsb(data)
// }

// func popFromStack16() uint16 {
// 	lsb := ram[cpu.SP]
// 	cpu.SP++
// 	msb := ram[cpu.SP]
// 	cpu.SP++
// 	return joinBytes(msb, lsb)
// }

func lsb(nn uint16) byte {
	return uint8(nn & 0xFF)
}

func msb(nn uint16) byte {
	return uint8(nn >> 8)
}

// func imm16() uint16 {
// 	lsb := readByte()
// 	msb := readByte()
// 	return joinBytes(msb, lsb)
// }

// For relative jumps, need to add signed int e to PC
func addInt8ToUint16(a uint16, e uint8) uint16 {
	sign := bit(7, e)
	result := a + uint16(e&0x7F)
	if sign {
		result -= 128
	}
	return result
}

func bit(col int, b byte) bool {
	return b>>col == 1
}

func clearFlags() {
	cpu.F &= 0x0F
}

func addAndSetFlags(a, b byte) byte {
	result := a + b
	// set Z flag
	setZFlag(result)

	// set N to 0
	setNegFlag(0)

	// if bit[3] carried, H = 1, else 0
	setHalfCarryAdd(a, b)

	// if bit[7] carried, C = 1, else 0
	if result < a {
		cpu.F |= 0x10
	} else {
		cpu.F &= 0xEF
	}

	return result
}

func addAndSetFlags16(a, b uint16) uint16 {
	result := a + b
	if result == 0 {
		cpu.F |= 0x80
	} else {
		cpu.F &= 0x7F
	}
	setNegFlag(1)

	hc := byte(((a ^ b ^ result) & 0x1000) >> 11)
	setHalfCarry(hc)

	if result < a {
		setCarry(1)
	} else {
		setCarry(0)
	}

	return result
}

func subAndSetFlags(a, b byte) byte {
	result := a - b
	setZFlag(result)
	setNegFlag(1)

	// if bit[3] carried, H = 1, else 0
	setHalfCarrySub(a, b)

	// if bit[7] carried, C = 1, else 0
	if result > a {
		cpu.F |= 0x10
	} else {
		cpu.F &= 0xEF
	}

	return result
}

func setZFlag(result byte) {
	if result == 0 {
		cpu.F |= 0x80
	} else {
		cpu.F &= 0x7F
	}
}

func setNegFlag(to byte) {
	if to == 0 {
		cpu.F &= 0xBF
	} else {
		cpu.F |= 0x40
	}
}

func setHalfCarryAdd(a, b byte) {
	result := a + b
	halfCarry := ((a ^ b ^ result) & 0x10) >> 4
	setHalfCarry(halfCarry)
}

func setHalfCarrySub(a, b byte) {
	result := a - b
	halfCarry := ((a ^ -b ^ result) & 0x10) >> 4
	setHalfCarry(halfCarry)
}

func setHalfCarry(to byte) {
	if to == 1 {
		cpu.F |= 0x20
	} else {
		cpu.F &= 0xDF
	}
}

func setCarry(setTo byte) {
	if setTo != 0 {
		cpu.F |= 0x20
	} else {
		cpu.F &= 0xDF
	}
}
