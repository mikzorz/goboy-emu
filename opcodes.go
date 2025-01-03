package main

import "log"
import "fmt"

type datatype string
type register string // register name strings

const (
	// DataTypes
	NODATA datatype = "NODATA" // For 1 byte opcode with no succeeding data
	N8     datatype = "N8"     // unsigned
	N16    datatype = "N16"
	A8     datatype = "A8"
	A16    datatype = "A16" // used as address
	E8     datatype = "E8"  // signed

	// Registers
	A    register = "A"
	B    register = "B"
	C    register = "C"
	mC   register = "[C]"
	BC   register = "BC"
	mBC  register = "[BC]"
	D    register = "D"
	E    register = "E"
	DE   register = "DE"
	mDE  register = "[DE]"
	H    register = "H"
	L    register = "L"
	HL   register = "HL"
	mHL  register = "[HL]"
	mHLp register = "[HL+]"
	mHLm register = "[HL-]"
	SP   register = "SP"
	SPe8 register = "SP + e8"
	m8   register = "[a8]"
	m16  register = "[a16]"
	AF   register = "AF"
	WZ   register = "WZ" // cpu 16bit buffer
	W    register = "W"  // WZ hi byte buffer
	Z    register = "Z"  // WZ lo byte buffer
	PC   register = "PC"

	// Flags
	NOFLAG = "NOFLAG"
	ZERO   = "ZERO"
	NZ     = "NZ"
	NC     = "NC"
	CARRY  = "CARRY"
)

type Instruction struct {
	Op       string
	DataType datatype
	To       register // Where result is stored
	From     register // 2nd value is taken from here, but not stored here. The name 'From' might be weird for SUB but whatever
	Flag     string
	// Len      int
	Bit int
	Abs byte // For RST, absolute address
}

// Could have probably used bit masking and whatnot. Oh well... maybe in version 2...
var mainLookup map[byte]Instruction = map[byte]Instruction{
	0x00: Instruction{Op: "NOP", DataType: NODATA},                        // No Operation
	0x01: Instruction{Op: "LD", DataType: N16, To: BC},                    // LD BC, n16
	0x02: Instruction{Op: "LD", DataType: NODATA, To: mBC, From: A},       // LD [BC], A
	0x03: Instruction{Op: "INC", DataType: NODATA, To: BC},                // INC BC
	0x04: Instruction{Op: "INC", DataType: NODATA, To: B},                 // INC B
	0x05: Instruction{Op: "DEC", DataType: NODATA, To: B},                 // DEC B
	0x06: Instruction{Op: "LD", DataType: N8, To: B},                      // LD B, n8
	0x07: Instruction{Op: "RLCA", DataType: NODATA},                       // RLCA (rotate left circular accumulator)
	0x08: Instruction{Op: "LD", DataType: A16, To: m16, From: SP},         // LD [a16], SP
	0x09: Instruction{Op: "ADD", DataType: NODATA, To: HL, From: BC},      // ADD HL, BC
	0x0A: Instruction{Op: "LD", DataType: NODATA, To: A, From: mBC},       // LD A, [BC]
	0x0B: Instruction{Op: "DEC", DataType: NODATA, To: BC},                // DEC BC
	0x0C: Instruction{Op: "INC", DataType: NODATA, To: C},                 // INC C
	0x0D: Instruction{Op: "DEC", DataType: NODATA, To: C},                 // DEC C
	0x0E: Instruction{Op: "LD", DataType: N8, To: C},                      // LD C, n8
	0x0F: Instruction{Op: "RRCA", DataType: NODATA},                       // RRCA (rotate right circular accumulator)
	0x10: Instruction{Op: "STOP", DataType: N8},                           // STOP n8
	0x11: Instruction{Op: "LD", DataType: N16, To: DE},                    // LD DE, n16
	0x12: Instruction{Op: "LD", DataType: NODATA, To: mDE, From: A},       // LD [DE], A
	0x13: Instruction{Op: "INC", DataType: NODATA, To: DE},                // INC DE
	0x14: Instruction{Op: "INC", DataType: NODATA, To: D},                 // INC D
	0x15: Instruction{Op: "DEC", DataType: NODATA, To: D},                 // DEC D
	0x16: Instruction{Op: "LD", DataType: N8, To: D},                      // LD D, n8
	0x17: Instruction{Op: "RLA", DataType: NODATA},                        // RLA (rotate left accumulator)
	0x18: Instruction{Op: "JR", DataType: E8, Flag: NOFLAG},               // Jump forward by e8
	0x19: Instruction{Op: "ADD", DataType: NODATA, To: HL, From: DE},      // ADD HL, DE
	0x1A: Instruction{Op: "LD", DataType: NODATA, To: A, From: mDE},       // LD A, [DE]
	0x1B: Instruction{Op: "DEC", DataType: NODATA, To: DE},                // DEC DE
	0x1C: Instruction{Op: "INC", DataType: NODATA, To: E},                 // INC E
	0x1D: Instruction{Op: "DEC", DataType: NODATA, To: E},                 // DEC E
	0x1E: Instruction{Op: "LD", DataType: N8, To: E},                      // LD E, n8
	0x1F: Instruction{Op: "RRA", DataType: NODATA, To: A, From: A},        // RRA, rotate A to right (todo, add carry to msb?)
	0x20: Instruction{Op: "JR", DataType: E8, Flag: NZ},                   // Jump by e8 if z == 0
	0x21: Instruction{Op: "LD", DataType: N16, To: HL},                    // LD HL, n16
	0x22: Instruction{Op: "LD", DataType: NODATA, To: mHLp, From: A},      // LD [HL+], A
	0x23: Instruction{Op: "INC", DataType: NODATA, To: HL},                // INC HL
	0x24: Instruction{Op: "INC", DataType: NODATA, To: H},                 // INC H
	0x25: Instruction{Op: "DEC", DataType: NODATA, To: H},                 // DEC H
	0x26: Instruction{Op: "LD", DataType: N8, To: H},                      // LD H, n8
	0x27: Instruction{Op: "DAA", DataType: NODATA},                        // DAA (Decimal Adjust Accumulator)
	0x28: Instruction{Op: "JR", DataType: E8, Flag: ZERO},                 // Jump by e8 if Z == 1
	0x29: Instruction{Op: "ADD", DataType: NODATA, To: HL, From: HL},      // ADD HL, HL
	0x2A: Instruction{Op: "LD", DataType: NODATA, To: A, From: mHLp},      // LD A, [HL+]
	0x2B: Instruction{Op: "DEC", DataType: NODATA, To: HL},                // DEC HL
	0x2C: Instruction{Op: "INC", DataType: NODATA, To: L},                 // INC L
	0x2D: Instruction{Op: "DEC", DataType: NODATA, To: L},                 // DEC L
	0x2E: Instruction{Op: "LD", DataType: N8, To: L},                      // LD L, n8
	0x2F: Instruction{Op: "CPL", DataType: NODATA},                        // CPL
	0x30: Instruction{Op: "JR", DataType: E8, Flag: NC},                   // Jump by e8 if c == 0
	0x31: Instruction{Op: "LD", DataType: N16, To: SP},                    // LD SP, n16
	0x32: Instruction{Op: "LD", DataType: NODATA, To: mHLm, From: A},      // LD [HL-], A
	0x33: Instruction{Op: "INC", DataType: NODATA, To: SP},                // INC SP
	0x34: Instruction{Op: "INC", DataType: NODATA, To: mHL, From: mHL},    // INC [HL]
	0x35: Instruction{Op: "DEC", DataType: NODATA, To: mHL, From: mHL},    // DEC [HL]
	0x36: Instruction{Op: "LD", DataType: N8, To: mHL},                    // LD [HL], n8
	0x37: Instruction{Op: "SCF", DataType: NODATA},                        // SCF
	0x38: Instruction{Op: "JR", DataType: E8, Flag: CARRY},                // Jump by e8 if c == 1
	0x39: Instruction{Op: "ADD", DataType: NODATA, To: HL, From: SP},      // ADD HL, SP
	0x3A: Instruction{Op: "LD", DataType: NODATA, To: A, From: mHLm},      // LD A, [HL-]
	0x3B: Instruction{Op: "DEC", DataType: NODATA, To: SP},                // DEC SP
	0x3C: Instruction{Op: "INC", DataType: NODATA, To: A},                 // INC A
	0x3D: Instruction{Op: "DEC", DataType: NODATA, To: A},                 // DEC A
	0x3E: Instruction{Op: "LD", DataType: N8, To: A},                      // LD A, n8
	0x3F: Instruction{Op: "CCF", DataType: NODATA},                        // CCF
	0x40: Instruction{Op: "LD", DataType: NODATA, To: B, From: B},         // LD B, B
	0x41: Instruction{Op: "LD", DataType: NODATA, To: B, From: C},         // LD B, C
	0x42: Instruction{Op: "LD", DataType: NODATA, To: B, From: D},         // LD B, D
	0x43: Instruction{Op: "LD", DataType: NODATA, To: B, From: E},         // LD B, E
	0x44: Instruction{Op: "LD", DataType: NODATA, To: B, From: H},         // LD B, H
	0x45: Instruction{Op: "LD", DataType: NODATA, To: B, From: L},         // LD B, L
	0x46: Instruction{Op: "LD", DataType: NODATA, To: B, From: mHL},       // LD B, [HL]
	0x47: Instruction{Op: "LD", DataType: NODATA, To: B, From: A},         // LD B, A
	0x48: Instruction{Op: "LD", DataType: NODATA, To: C, From: B},         // LD C, B
	0x49: Instruction{Op: "LD", DataType: NODATA, To: C, From: C},         // LD C, C
	0x4A: Instruction{Op: "LD", DataType: NODATA, To: C, From: D},         // LD C, D
	0x4B: Instruction{Op: "LD", DataType: NODATA, To: C, From: E},         // LD C, E
	0x4C: Instruction{Op: "LD", DataType: NODATA, To: C, From: H},         // LD C, H
	0x4D: Instruction{Op: "LD", DataType: NODATA, To: C, From: L},         // LD C, L
	0x4E: Instruction{Op: "LD", DataType: NODATA, To: C, From: mHL},       // LD C, [HL]
	0x4F: Instruction{Op: "LD", DataType: NODATA, To: C, From: A},         // LD C, A
	0x50: Instruction{Op: "LD", DataType: NODATA, To: D, From: B},         // LD D, B
	0x51: Instruction{Op: "LD", DataType: NODATA, To: D, From: C},         // LD D, C
	0x52: Instruction{Op: "LD", DataType: NODATA, To: D, From: D},         // LD D, D
	0x53: Instruction{Op: "LD", DataType: NODATA, To: D, From: E},         // LD D, E
	0x54: Instruction{Op: "LD", DataType: NODATA, To: D, From: H},         // LD D, H
	0x55: Instruction{Op: "LD", DataType: NODATA, To: D, From: L},         // LD D, L
	0x56: Instruction{Op: "LD", DataType: NODATA, To: D, From: mHL},       // LD D, [HL]
	0x57: Instruction{Op: "LD", DataType: NODATA, To: D, From: A},         // LD D, A
	0x58: Instruction{Op: "LD", DataType: NODATA, To: E, From: B},         // LD E, B
	0x59: Instruction{Op: "LD", DataType: NODATA, To: E, From: C},         // LD E, C
	0x5A: Instruction{Op: "LD", DataType: NODATA, To: E, From: D},         // LD E, D
	0x5B: Instruction{Op: "LD", DataType: NODATA, To: E, From: E},         // LD E, E
	0x5C: Instruction{Op: "LD", DataType: NODATA, To: E, From: H},         // LD E, H
	0x5D: Instruction{Op: "LD", DataType: NODATA, To: E, From: L},         // LD E, L
	0x5E: Instruction{Op: "LD", DataType: NODATA, To: E, From: mHL},       // LD E, [HL]
	0x5F: Instruction{Op: "LD", DataType: NODATA, To: E, From: A},         // LD E, A
	0x60: Instruction{Op: "LD", DataType: NODATA, To: H, From: B},         // LD H, B
	0x61: Instruction{Op: "LD", DataType: NODATA, To: H, From: C},         // LD H, C
	0x62: Instruction{Op: "LD", DataType: NODATA, To: H, From: D},         // LD H, D
	0x63: Instruction{Op: "LD", DataType: NODATA, To: H, From: E},         // LD H, E
	0x64: Instruction{Op: "LD", DataType: NODATA, To: H, From: H},         // LD H, H
	0x65: Instruction{Op: "LD", DataType: NODATA, To: H, From: L},         // LD H, L
	0x66: Instruction{Op: "LD", DataType: NODATA, To: H, From: mHL},       // LD H, [HL]
	0x67: Instruction{Op: "LD", DataType: NODATA, To: H, From: A},         // LD H, A
	0x68: Instruction{Op: "LD", DataType: NODATA, To: L, From: B},         // LD L, B
	0x69: Instruction{Op: "LD", DataType: NODATA, To: L, From: C},         // LD L, C
	0x6A: Instruction{Op: "LD", DataType: NODATA, To: L, From: D},         // LD L, D
	0x6B: Instruction{Op: "LD", DataType: NODATA, To: L, From: E},         // LD L, E
	0x6C: Instruction{Op: "LD", DataType: NODATA, To: L, From: H},         // LD L, H
	0x6D: Instruction{Op: "LD", DataType: NODATA, To: L, From: L},         // LD L, L
	0x6E: Instruction{Op: "LD", DataType: NODATA, To: L, From: mHL},       // LD L, [HL]
	0x6F: Instruction{Op: "LD", DataType: NODATA, To: L, From: A},         // LD L, A
	0x70: Instruction{Op: "LD", DataType: NODATA, To: mHL, From: B},       // LD [HL], B
	0x71: Instruction{Op: "LD", DataType: NODATA, To: mHL, From: C},       // LD [HL], C
	0x72: Instruction{Op: "LD", DataType: NODATA, To: mHL, From: D},       // LD [HL], D
	0x73: Instruction{Op: "LD", DataType: NODATA, To: mHL, From: E},       // LD [HL], E
	0x74: Instruction{Op: "LD", DataType: NODATA, To: mHL, From: H},       // LD [HL], H
	0x75: Instruction{Op: "LD", DataType: NODATA, To: mHL, From: L},       // LD [HL], L
	0x76: Instruction{Op: "HALT", DataType: NODATA},                       // HALT
	0x77: Instruction{Op: "LD", DataType: NODATA, To: mHL, From: A},       // LD [HL], A
	0x78: Instruction{Op: "LD", DataType: NODATA, To: A, From: B},         // LD A, B
	0x79: Instruction{Op: "LD", DataType: NODATA, To: A, From: C},         // LD A, C
	0x7A: Instruction{Op: "LD", DataType: NODATA, To: A, From: D},         // LD A, D
	0x7B: Instruction{Op: "LD", DataType: NODATA, To: A, From: E},         // LD A, E
	0x7C: Instruction{Op: "LD", DataType: NODATA, To: A, From: H},         // LD A, H
	0x7D: Instruction{Op: "LD", DataType: NODATA, To: A, From: L},         // LD A, L
	0x7E: Instruction{Op: "LD", DataType: NODATA, To: A, From: mHL},       // LD A, [HL]
	0x7F: Instruction{Op: "LD", DataType: NODATA, To: A, From: A},         // LD A, A
	0x80: Instruction{Op: "ADD", DataType: NODATA, To: A, From: B},        // ADD A, B
	0x81: Instruction{Op: "ADD", DataType: NODATA, To: A, From: C},        // ADD A, C
	0x82: Instruction{Op: "ADD", DataType: NODATA, To: A, From: D},        // ADD A, D
	0x83: Instruction{Op: "ADD", DataType: NODATA, To: A, From: E},        // ADD A, E
	0x84: Instruction{Op: "ADD", DataType: NODATA, To: A, From: H},        // ADD A, H
	0x85: Instruction{Op: "ADD", DataType: NODATA, To: A, From: L},        // ADD A, L
	0x86: Instruction{Op: "ADD", DataType: NODATA, To: A, From: mHL},      // ADD A, [HL]
	0x87: Instruction{Op: "ADD", DataType: NODATA, To: A, From: A},        // ADD A, A
	0x88: Instruction{Op: "ADC", DataType: NODATA, To: A, From: B},        // ADC A, B
	0x89: Instruction{Op: "ADC", DataType: NODATA, To: A, From: C},        // ADC A, C
	0x8A: Instruction{Op: "ADC", DataType: NODATA, To: A, From: D},        // ADC A, D
	0x8B: Instruction{Op: "ADC", DataType: NODATA, To: A, From: E},        // ADC A, E
	0x8C: Instruction{Op: "ADC", DataType: NODATA, To: A, From: H},        // ADC A, H
	0x8D: Instruction{Op: "ADC", DataType: NODATA, To: A, From: L},        // ADC A, L
	0x8E: Instruction{Op: "ADC", DataType: NODATA, To: A, From: mHL},      // ADC A, [HL]
	0x8F: Instruction{Op: "ADC", DataType: NODATA, To: A, From: A},        // ADC A, A
	0x90: Instruction{Op: "SUB", DataType: NODATA, To: A, From: B},        // SUB A, B
	0x91: Instruction{Op: "SUB", DataType: NODATA, To: A, From: C},        // SUB A, C
	0x92: Instruction{Op: "SUB", DataType: NODATA, To: A, From: D},        // SUB A, D
	0x93: Instruction{Op: "SUB", DataType: NODATA, To: A, From: E},        // SUB A, E
	0x94: Instruction{Op: "SUB", DataType: NODATA, To: A, From: H},        // SUB A, H
	0x95: Instruction{Op: "SUB", DataType: NODATA, To: A, From: L},        // SUB A, L
	0x96: Instruction{Op: "SUB", DataType: NODATA, To: A, From: mHL},      // SUB A, [HL]
	0x97: Instruction{Op: "SUB", DataType: NODATA, To: A, From: A},        // SUB A, A
	0x98: Instruction{Op: "SBC", DataType: NODATA, To: A, From: B},        // SBC A, B
	0x99: Instruction{Op: "SBC", DataType: NODATA, To: A, From: C},        // SBC A, C
	0x9A: Instruction{Op: "SBC", DataType: NODATA, To: A, From: D},        // SBC A, D
	0x9B: Instruction{Op: "SBC", DataType: NODATA, To: A, From: E},        // SBC A, E
	0x9C: Instruction{Op: "SBC", DataType: NODATA, To: A, From: H},        // SBC A, H
	0x9D: Instruction{Op: "SBC", DataType: NODATA, To: A, From: L},        // SBC A, L
	0x9E: Instruction{Op: "SBC", DataType: NODATA, To: A, From: mHL},      // SBC A, [HL]
	0x9F: Instruction{Op: "SBC", DataType: NODATA, To: A, From: A},        // SBC A, A
	0xA0: Instruction{Op: "AND", DataType: NODATA, To: A, From: B},        // AND A, B
	0xA1: Instruction{Op: "AND", DataType: NODATA, To: A, From: C},        // AND A, C
	0xA2: Instruction{Op: "AND", DataType: NODATA, To: A, From: D},        // AND A, D
	0xA3: Instruction{Op: "AND", DataType: NODATA, To: A, From: E},        // AND A, E
	0xA4: Instruction{Op: "AND", DataType: NODATA, To: A, From: H},        // AND A, H
	0xA5: Instruction{Op: "AND", DataType: NODATA, To: A, From: L},        // AND A, L
	0xA6: Instruction{Op: "AND", DataType: NODATA, To: A, From: mHL},      // AND A, [HL]
	0xA7: Instruction{Op: "AND", DataType: NODATA, To: A, From: A},        // AND A, A
	0xA8: Instruction{Op: "XOR", DataType: NODATA, To: A, From: B},        // XOR A, B
	0xA9: Instruction{Op: "XOR", DataType: NODATA, To: A, From: C},        // XOR A, C
	0xAA: Instruction{Op: "XOR", DataType: NODATA, To: A, From: D},        // XOR A, D
	0xAB: Instruction{Op: "XOR", DataType: NODATA, To: A, From: E},        // XOR A, E
	0xAC: Instruction{Op: "XOR", DataType: NODATA, To: A, From: H},        // XOR A, H
	0xAD: Instruction{Op: "XOR", DataType: NODATA, To: A, From: L},        // XOR A, L
	0xAE: Instruction{Op: "XOR", DataType: NODATA, To: A, From: mHL},      // XOR A, [HL]
	0xAF: Instruction{Op: "XOR", DataType: NODATA, To: A, From: A},        // XOR A, A
	0xB0: Instruction{Op: "OR", DataType: NODATA, To: A, From: B},         // OR A, B
	0xB1: Instruction{Op: "OR", DataType: NODATA, To: A, From: C},         // OR A, C
	0xB2: Instruction{Op: "OR", DataType: NODATA, To: A, From: D},         // OR A, D
	0xB3: Instruction{Op: "OR", DataType: NODATA, To: A, From: E},         // OR A, E
	0xB4: Instruction{Op: "OR", DataType: NODATA, To: A, From: H},         // OR A, H
	0xB5: Instruction{Op: "OR", DataType: NODATA, To: A, From: L},         // OR A, L
	0xB6: Instruction{Op: "OR", DataType: NODATA, To: A, From: mHL},       // OR A, [HL]
	0xB7: Instruction{Op: "OR", DataType: NODATA, To: A, From: A},         // OR A, A
	0xB8: Instruction{Op: "CP", DataType: NODATA, To: A, From: B},         // CP A, B
	0xB9: Instruction{Op: "CP", DataType: NODATA, To: A, From: C},         // CP A, C
	0xBA: Instruction{Op: "CP", DataType: NODATA, To: A, From: D},         // CP A, D
	0xBB: Instruction{Op: "CP", DataType: NODATA, To: A, From: E},         // CP A, E
	0xBC: Instruction{Op: "CP", DataType: NODATA, To: A, From: H},         // CP A, H
	0xBD: Instruction{Op: "CP", DataType: NODATA, To: A, From: L},         // CP A, L
	0xBE: Instruction{Op: "CP", DataType: NODATA, To: A, From: mHL},       // CP A, [HL]
	0xBF: Instruction{Op: "CP", DataType: NODATA, To: A, From: A},         // CP A, A
	0xC0: Instruction{Op: "RET", DataType: NODATA, Flag: NZ},              // RET NZ
	0xC1: Instruction{Op: "POP", DataType: NODATA, To: BC},                // POP BC
	0xC2: Instruction{Op: "JP", DataType: A16, From: m16, Flag: NZ},       // JP NZ, a16
	0xC3: Instruction{Op: "JP", DataType: A16, From: m16, Flag: NOFLAG},   // JP a16
	0xC4: Instruction{Op: "CALL", DataType: A16, Flag: NZ, From: m16},     // CALL NZ, a16
	0xC5: Instruction{Op: "PUSH", DataType: NODATA, From: BC},             // PUSH BC
	0xC6: Instruction{Op: "ADD", DataType: N8, To: A},                     // ADD A, n8
	0xC7: Instruction{Op: "RST", DataType: NODATA, Abs: 0x00},             // RST 0x00
	0xC8: Instruction{Op: "RET", DataType: NODATA, Flag: ZERO},            // RET Z
	0xC9: Instruction{Op: "RET", DataType: NODATA, Flag: NOFLAG},          // RET
	0xCA: Instruction{Op: "JP", DataType: A16, From: m16, Flag: ZERO},     // JP Z, a16
	0xCB: Instruction{Op: "PREFIX", DataType: NODATA},                     // PREFIX
	0xCC: Instruction{Op: "CALL", DataType: A16, Flag: ZERO, From: m16},   // CALL Z, a16
	0xCD: Instruction{Op: "CALL", DataType: A16, Flag: NOFLAG, From: m16}, // CALL a16
	0xCE: Instruction{Op: "ADC", DataType: N8, To: A},                     // ADC A, n8
	0xCF: Instruction{Op: "RST", DataType: NODATA, Abs: 0x08},             // RST 0x08
	0xD0: Instruction{Op: "RET", DataType: NODATA, Flag: NC},              // RET NC
	0xD1: Instruction{Op: "POP", DataType: NODATA, To: DE},                // POP DE
	0xD2: Instruction{Op: "JP", DataType: A16, From: m16, Flag: NC},       // JP NC, a16
	0xD3: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xD4: Instruction{Op: "CALL", DataType: A16, Flag: NC, From: m16},     // CALL NC, a16
	0xD5: Instruction{Op: "PUSH", DataType: NODATA, From: DE},             // PUSH DE
	0xD6: Instruction{Op: "SUB", DataType: N8, To: A},                     // SUB A, n8
	0xD7: Instruction{Op: "RST", DataType: NODATA, Abs: 0x10},             // RST 0x10
	0xD8: Instruction{Op: "RET", DataType: NODATA, Flag: CARRY},           // RET C
	0xD9: Instruction{Op: "RETI", DataType: NODATA},                       // RETI
	0xDA: Instruction{Op: "JP", DataType: A16, From: m16, Flag: CARRY},    // JP C, a16
	0xDB: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xDC: Instruction{Op: "CALL", DataType: A16, Flag: CARRY, From: m16},  // CALL C, a16
	0xDD: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xDE: Instruction{Op: "SBC", DataType: N8, To: A},                     // SBC A, n8
	0xDF: Instruction{Op: "RST", DataType: NODATA, Abs: 0x18},             // RST 0x18
	0xE0: Instruction{Op: "LDH", DataType: A8, To: m8, From: A},           // LDH [a8], A (LD ram[0xFF00 + n], A)
	0xE1: Instruction{Op: "POP", DataType: NODATA, To: HL},                // POP HL
	0xE2: Instruction{Op: "LD", DataType: NODATA, To: mC, From: A},        // LD [C], A
	0xE3: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xE4: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xE5: Instruction{Op: "PUSH", DataType: NODATA, From: HL},             // PUSH HL
	0xE6: Instruction{Op: "AND", DataType: N8, To: A},                     // AND A, n8
	0xE7: Instruction{Op: "RST", DataType: NODATA, Abs: 0x20},             // RST 0x20 (CALL, but 1 byte instead of 3)
	0xE8: Instruction{Op: "ADD", DataType: E8, To: SP},                    // ADD SP, e8
	0xE9: Instruction{Op: "JP", DataType: NODATA, From: HL},               // JP HL
	0xEA: Instruction{Op: "LD", DataType: A16, To: m16, From: A},          // LD [a16], A
	0xEB: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xEC: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xED: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xEE: Instruction{Op: "XOR", DataType: N8, To: A},                     // XOR A, n8
	0xEF: Instruction{Op: "RST", DataType: NODATA, Abs: 0x28},             // RST 0x28
	0xF0: Instruction{Op: "LDH", DataType: A8, To: A, From: m8},           // LDH A, [a8]
	0xF1: Instruction{Op: "POP", DataType: NODATA, To: AF},                // POP AF
	0xF2: Instruction{Op: "LD", DataType: NODATA, To: A, From: mC},        // LD A, [C]
	0xF3: Instruction{Op: "DI", DataType: NODATA},                         // Disable Interrupts
	0xF4: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xF5: Instruction{Op: "PUSH", DataType: NODATA, From: AF},             // PUSH AF
	0xF6: Instruction{Op: "OR", DataType: N8, To: A},                      // OR A n8
	0xF7: Instruction{Op: "RST", DataType: NODATA, Abs: 0x30},             // RST 0x30
	0xF8: Instruction{Op: "LD", DataType: E8, To: HL, From: SPe8},         // LD HL, SP + e8
	0xF9: Instruction{Op: "LD", DataType: NODATA, To: SP, From: HL},       // LD SP, HL
	0xFA: Instruction{Op: "LD", DataType: A16, To: A, From: m16},          // LD A, [a16]
	0xFB: Instruction{Op: "EI", DataType: NODATA},                         // Enable Interrupts after next machine cycle
	0xFC: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xFD: Instruction{Op: "ILLEGAL", DataType: NODATA},                    // ILLEGAL
	0xFE: Instruction{Op: "CP", DataType: N8, To: A},                      // CP A, n8 (Sub n8 from A and update flags. Do not update A.)
	0xFF: Instruction{Op: "RST", DataType: NODATA, Abs: 0x38},             // RST 0x38
}

var prefixedLookup map[byte]Instruction = make(map[byte]Instruction)

func populatePrefixLookup() {
	for op := 0x0; op <= 0xFF; op++ {
		prefixedLookup[byte(op)] = getPrefixInstructionFromOp(byte(op))
	}
}

func getPrefixInstructionFromOp(b byte) Instruction {
	r := getReg8ByOp(b % 8)
	if b&0xF < 8 {
		switch b >> 4 {
		case 0x0:
			return Instruction{Op: "RLC", DataType: NODATA, To: r, From: r}
		case 0x1:
			return Instruction{Op: "RL", DataType: NODATA, To: r, From: r}
		case 0x2:
			return Instruction{Op: "SLA", DataType: NODATA, To: r, From: r}
		case 0x3:
			return Instruction{Op: "SWAP", DataType: NODATA, To: r, From: r}
		case 0x4:
			return Instruction{Op: "BIT", DataType: NODATA, To: r, From: r, Bit: 0}
		case 0x5:
			return Instruction{Op: "BIT", DataType: NODATA, To: r, From: r, Bit: 2}
		case 0x6:
			return Instruction{Op: "BIT", DataType: NODATA, To: r, From: r, Bit: 4}
		case 0x7:
			return Instruction{Op: "BIT", DataType: NODATA, To: r, From: r, Bit: 6}
		case 0x8:
			return Instruction{Op: "RES", DataType: NODATA, To: r, From: r, Bit: 0}
		case 0x9:
			return Instruction{Op: "RES", DataType: NODATA, To: r, From: r, Bit: 2}
		case 0xA:
			return Instruction{Op: "RES", DataType: NODATA, To: r, From: r, Bit: 4}
		case 0xB:
			return Instruction{Op: "RES", DataType: NODATA, To: r, From: r, Bit: 6}
		case 0xC:
			return Instruction{Op: "SET", DataType: NODATA, To: r, From: r, Bit: 0}
		case 0xD:
			return Instruction{Op: "SET", DataType: NODATA, To: r, From: r, Bit: 2}
		case 0xE:
			return Instruction{Op: "SET", DataType: NODATA, To: r, From: r, Bit: 4}
		case 0xF:
			return Instruction{Op: "SET", DataType: NODATA, To: r, From: r, Bit: 6}
		}
	} else {
		switch b >> 4 {
		case 0x0:
			return Instruction{Op: "RRC", DataType: NODATA, To: r, From: r}
		case 0x1:
			return Instruction{Op: "RR", DataType: NODATA, To: r, From: r}
		case 0x2:
			return Instruction{Op: "SRA", DataType: NODATA, To: r, From: r}
		case 0x3:
			return Instruction{Op: "SRL", DataType: NODATA, To: r, From: r}
		case 0x4:
			return Instruction{Op: "BIT", DataType: NODATA, To: r, From: r, Bit: 1}
		case 0x5:
			return Instruction{Op: "BIT", DataType: NODATA, To: r, From: r, Bit: 3}
		case 0x6:
			return Instruction{Op: "BIT", DataType: NODATA, To: r, From: r, Bit: 5}
		case 0x7:
			return Instruction{Op: "BIT", DataType: NODATA, To: r, From: r, Bit: 7}
		case 0x8:
			return Instruction{Op: "RES", DataType: NODATA, To: r, From: r, Bit: 1}
		case 0x9:
			return Instruction{Op: "RES", DataType: NODATA, To: r, From: r, Bit: 3}
		case 0xA:
			return Instruction{Op: "RES", DataType: NODATA, To: r, From: r, Bit: 5}
		case 0xB:
			return Instruction{Op: "RES", DataType: NODATA, To: r, From: r, Bit: 7}
		case 0xC:
			return Instruction{Op: "SET", DataType: NODATA, To: r, From: r, Bit: 1}
		case 0xD:
			return Instruction{Op: "SET", DataType: NODATA, To: r, From: r, Bit: 3}
		case 0xE:
			return Instruction{Op: "SET", DataType: NODATA, To: r, From: r, Bit: 5}
		case 0xF:
			return Instruction{Op: "SET", DataType: NODATA, To: r, From: r, Bit: 7}
		}
	}
	return Instruction{}
}

// Opcodes determine the variable register that is used.
// getReg8ByOp uses the lowest nibble, 0-7
// 0=B, 1=C ... 7=A
func getReg8ByOp(loNib byte) register {
	switch loNib {
	case 0:
		return B
	case 1:
		return C
	case 2:
		return D
	case 3:
		return E
	case 4:
		return H
	case 5:
		return L
	case 6:
		return mHL
	case 7:
		return A
	default:
		log.Panicf("invalid nibble value")
		return ""
	}
}

// TODO, remove any processing stuff, set and return Instruction with correct data. May take a while
func lookup(op byte, prefix bool) Instruction {
	var inst Instruction
	var ok bool
	if prefix {
		inst, ok = prefixedLookup[op]
		if !ok {
			log.Panicf("could not find prefix opcode %02X in lookup", op)
		}
	} else {
		inst, ok = mainLookup[op]
		if !ok {
			log.Panicf("could not find opcode %02X in lookup", op)
		}
	}
	if !ok {
		inst.Op = fmt.Sprintf("%02X prefix?: %t", op, prefix)
		inst.DataType = "UNKNOWN"
	}
	return inst
}
