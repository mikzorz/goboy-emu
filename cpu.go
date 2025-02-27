package main

import (
	"fmt"
	"github.com/mikzorz/gameboy-emulator/alu"
	utils "github.com/mikzorz/gameboy-emulator/helpers"
	"log"
)

const (
	R8       = "R8" // For checking register types
	R16      = "R16"
	DIRECT   = "DIRECT"   // [a16], [a8] etc
	INDIRECT = "INDIRECT" // [HL], [HL+], [BC] etc
	NONE     = "NONE"     // literal value n8/n16

	// Which byte to act upon when doing things like pushing to stack
	LO = "lo"
	HI = "hi"
)

type IDU struct {
	IDUBusy bool
}

func (idu *IDU) IDUInc(val uint16) uint16 {
	idu.IDUBusy = true
	return val + 1
}

func (idu *IDU) IDUDec(val uint16) uint16 {
	idu.IDUBusy = true
	return val - 1
}

type RegisterFile struct {
	IR, IME, A, F      uint8
	IF                 byte // Interrupt Flag
	IE                 byte // Interrupt Enable
	BC, DE, HL, PC, SP uint16
	WZ                 uint16 // intermediate 16 bit register
}

// should decode instructions and handle interrupts
type ControlUnit struct {
}

type CPU struct {
	alu.ALU
	IDU
	RegisterFile
	ControlUnit
	bus                BusI
	interruptAddr      uint8
	interruptAddresses []uint8

	curCycle byte // current cycle of op

	inst        Instruction // current instruction
	instAddr    uint16      // address of opcode, for debugger
	opFunc      func()
	flagMatched bool
	setIME      bool // IME setting is delayed 1 cycle
	untilIME    int
	haltBug     bool
	skipLog     bool // for Gameboy Doctor, to not log after moving to interrupt vector
}

func NewCPU() *CPU {
	c := &CPU{
		ALU: alu.ALU{},
		IDU: IDU{},
		RegisterFile: RegisterFile{
			PC: 0x100,
			A:  0x01,
			F:  0xB0,
			BC: 0x0013,
			DE: 0x00D8,
			HL: 0x014D,
			SP: 0xFFFE,
			IF: 0x01, // During boot rom, vblank requests are made. Set for lack of boot ram.
		},
		ControlUnit:        ControlUnit{},
		interruptAddresses: []uint8{0x40, 0x48, 0x50, 0x58, 0x60}, // V-Blank, LCDC, Timer, Serial, Joypad
		inst:               lookup(0x00, false),                   // NOP
		curCycle:           0xFF,
	}
	c.opFunc = c.NOP
	return c
}

func (c *CPU) Cycle() {
	if !c.bus.isHalted() {

		// Execute next cycle of op
		c.curCycle++
		// if c.setIME {
		// 	c.IME = 1
		// 	c.setIME = false
		// }

		c.opFunc()

		// }
	} else {
		// log.Fatalf("halted, TODO: handle this")
		c.CheckInterrupts()
	}

}

func (c *CPU) DecodePrefix() {
	// Should interrupts be checked after prefix?
	// Wouldn't that cause incorrect functions to run after a RET?
	c.FetchIR(true)

	c.opFunc = func() {
		if c.curCycle == 0 {
			c.Read()
			return
		}

		if c.curCycle == 1 || c.inst.To != mHL {
			switch c.inst.Op {
			case "SWAP":
				c.Swap()
			case "BIT":
				c.Bit()
				c.DecodeOp()
				return
			case "RES":
				c.Res()
			case "SET":
				c.Set()
			case "SRA":
				c.SRA()
			case "SLA":
				c.SLA()
			case "SRL":
				c.SRL()
			case "RR":
				c.RR()
			case "RRC":
				c.RRC()
			case "RL":
				c.RL()
			case "RLC":
				c.RLC()
			default:
				log.Panicf("unimplemented PREFIXED op: %s/0x%02X, dt: %v, to: %s, from: %s, flag: %s", c.inst.Op, c.IR, c.inst.DataType, c.inst.To, c.inst.From, c.inst.Flag)
			}

			if c.inst.To != mHL {
				c.SetRegister()
				c.DecodeOp()
			} else {
				c.Write()

			}
			return
		}

		if c.curCycle == 2 {
			c.DecodeOp()
		}
	}
}

func (c *CPU) CheckInterrupts() (interrupted bool) {
	// According to a reddit comment, normal cpu cycle is T-cycle 1, but interrupt checks are during T3?
	// Check interrupt bytes
	if c.IF != 0 && c.IE != 0 {
		c.bus.setHalt(false)
		if c.IME == 1 {
			for bit := 0; bit <= 4; bit++ {
				if utils.IsBitSet(bit, c.IF) && utils.IsBitSet(bit, c.IE) {
					c.IME = 0
					c.interruptAddr = c.interruptAddresses[bit]
					c.IF = utils.ResetBit(bit, c.IF)

					c.IR = 0xC7 // RST
					c.inst = lookup(c.IR, false)
					c.inst.Op = "INT"
					// c.inst.Abs = c.interruptAddr
					c.WZ = utils.JoinBytes(0x00, c.interruptAddr)
					c.curCycle = 0xFF

					// interrupt transition
					c.opFunc = c.MoveToInterrupt
					// c.MoveToInterrupt()
					return true
				}
			}
		}
	}
	return false
}

func (c *CPU) MoveToInterrupt() {
	// TODO, should Bank 0 be forced back to default?
	switch c.curCycle {
	case 0:
		// NOP
	case 1:
		// NOP
		c.decrementReg(SP)
	case 2:
		c.pushPCToStack(HI)
		c.decrementReg(SP)
	case 3:
		c.pushPCToStack(LO)
	case 4:
		c.SetPC()
		c.skipLog = true
		c.DecodeOp()
	}
}

func (c *CPU) setCPFunc() {
	if c.inst.DataType == N8 {
		c.opFunc = c.CPn
	} else if c.inst.From == mHL {
		c.opFunc = c.CPmhl
	} else {
		c.opFunc = c.CPr
	}
}

// Compare with register
func (c *CPU) CPr() {
	// c.ReadReg()
	c.Read()
	c.compare()
	c.DecodeOp()
}

// CP [HL]
func (c *CPU) CPmhl() {
	switch c.curCycle {
	case 0:
		c.writeR8(Z, c.readIndirect(mHL))
	case 1:
		c.compare()
		c.DecodeOp()
	}
}

// CP n8
func (c *CPU) CPn() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.compare()
		c.DecodeOp()
	}
}

// Subtract c.Z from c.A and set flags. Do not store result in c.A.
func (c *CPU) compare() {
	c.sub(c.A, c.readR8(Z), false)
}

func (c *CPU) setJPFunc() {
	switch c.inst.DataType {
	case A16:
		// if c.inst.Flag == NOFLAG {
		// 	c.opFunc = c.JPDirectNoConditional
		// } else {
		// 	c.opFunc = c.JPDirectWithConditional
		// }
		c.opFunc = c.JPDirect
	case NODATA:
		c.opFunc = c.JPhl
	default:
		log.Panicf("unhandled JP datatype")
	}

}

func (c *CPU) JPDirectNoConditional() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.Fetch(HI)
	case 2:
		c.SetPC()
	case 3:
		c.DecodeOp()
	}
}

func (c *CPU) JPDirect() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.Fetch(HI)
		c.CheckFlag()
	case 2:
		if c.FlagMatches() {
			c.SetPC()
		} else {
			c.DecodeOp()
		}
	case 3:
		c.DecodeOp()
	}
}

func (c *CPU) JPhl() {
	c.PC = c.HL
	c.DecodeOp()
}

func (c *CPU) JR() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
		c.CheckFlag()
	case 1:
		if c.FlagMatches() {
			c.AddRelPC()
		} else {
			c.DecodeOp()
		}
	case 2:
		c.SetPC()
		c.DecodeOp()
	}
}

func (c *CPU) setORFunc() {
	if c.inst.DataType == N8 {
		c.opFunc = c.ORn
	} else if c.inst.From == mHL {
		c.opFunc = c.ORmhl
	} else {
		c.opFunc = c.ORr
	}
}

func (c *CPU) ORr() {
	c.Read()
	c.Or()
	c.DecodeOp()
}

func (c *CPU) ORmhl() {
	switch c.curCycle {
	case 0:
		c.Read()
	case 1:
		c.Or()
		c.DecodeOp()
	}
}

func (c *CPU) ORn() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.Or()
		c.DecodeOp()
	}
}

// OR two bytes A and x and store result in A.
func (c *CPU) Or() {
	c.A |= c.readR8(Z)
	c.clearFlags()
	c.setZFlag(c.A)
}

func (c *CPU) setXORFunc() {
	if c.inst.DataType == N8 {
		c.opFunc = c.XORn
	} else if c.inst.From == mHL {
		c.opFunc = c.XORmhl
	} else {
		c.opFunc = c.XORr
	}
}

func (c *CPU) XORr() {
	c.Read()
	c.Xor()
	c.DecodeOp()
}

func (c *CPU) XORmhl() {
	switch c.curCycle {
	case 0:
		c.writeR8(Z, c.readIndirect(mHL))
	case 1:
		c.Xor()
		c.DecodeOp()
	}
}

func (c *CPU) XORn() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.Xor()
		c.DecodeOp()
	}
}

// XOR two bytes A and x and store result in A.
func (c *CPU) Xor() {
	c.A ^= c.readR8(Z)
	c.clearFlags()
	c.setZFlag(c.A)
}

func (c *CPU) setLDFunc() {
	// dt := c.inst.DataType
	t := c.inst.To
	f := c.inst.From

	if c.IR&0xC0 == 0x40 {
		if c.IR&0x7 == 0x6 {
			// 01xxx110
			// LD r (HL)
			c.opFunc = func() {
				switch c.curCycle {
				case 0:
					c.Read()
				case 1:
					c.SetRegister()
					c.DecodeOp()
				}
			}
		} else if (c.IR>>3)&0x7 == 0x6 {
			// 01110xxx
			// LD (HL) r
			c.opFunc = func() {
				switch c.curCycle {
				case 0:
					c.Read()
					c.writeIndirect(mHL, c.readR8(Z))
				case 1:
					c.DecodeOp()
				}
			}
		} else {
			// 01xxxyyy
			// Ld r r'
			c.opFunc = func() {
				c.writeR8(t, c.readR8(f))
				c.DecodeOp()
			}
		}
		return
	} else if c.IR&0xC7 == 0x06 {
		// 00xxx110
		if c.IR == 0x36 {
			// LD (HL) n
			c.opFunc = func() {
				switch c.curCycle {
				case 0:
					c.Fetch(LO)
				case 1:
					c.writeIndirect(mHL, c.readR8(Z))
				case 2:
					c.DecodeOp()
				}
			}
		} else {
			// LD r n
			c.opFunc = c.LDn
		}
		return
	} else if c.IR&0xCF == 0x1 {
		// 00xx0001
		// LD rr nn
		c.opFunc = c.LDnn
		return
	}

	// 0x2 == ld (rr) a
	// 0xa == ld a (rr)
	// if 0x2 || 0xa, then
	//  0x0x == BC
	//  0x1x == DE
	//  0x2x == HL+
	//  0x3x == HL-
	switch c.IR {
	case 0x02:
		// LD (BC) A
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				// (bc) <- a
				c.writeMem(c.BC, c.A)
			case 1:
				c.DecodeOp()
			}
		}
	case 0x08:
		// LD (a16) SP
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				c.Fetch(LO)
			case 1:
				c.Fetch(HI)
			case 2:
				c.writeMem(c.WZ, utils.LSB(c.SP))
				c.WZ = c.IDUInc(c.WZ)
			case 3:
				c.writeMem(c.WZ, utils.MSB(c.SP))
			case 4:
				c.DecodeOp()
			}
		}
	case 0x0A:
		// LD A (BC)
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				c.Read()
			case 1:
				c.SetRegister()
				c.DecodeOp()
			}
		}
	case 0x12:
		// LD (DE) A
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				// (de) <- a
				c.writeMem(c.DE, c.A)
			case 1:
				c.DecodeOp()
			}
		}
	case 0x1A:
		// LD A (DE)
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				c.Read()
			case 1:
				c.SetRegister()
				c.DecodeOp()
			}
		}
	case 0x22:
		// LD (HL+) A
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				c.Read()
				c.Write()
				c.incrementReg(HL)
			case 1:
				c.DecodeOp()
			}
		}
	case 0x2A:
		// LD A (HL+)
		c.opFunc = c.LDHLPlus
	case 0x32:
		// LD (HL-) A
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				c.Read()
				c.Write()
				c.decrementReg(HL)
			case 1:
				c.DecodeOp()
			}
		}
	case 0x3A:
		// LD A (HL-)
		c.opFunc = c.LDHLMinus
	case 0xE2:
		// LD (C) a
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				c.writeIndirect(mC, c.A)
			case 1:
				c.DecodeOp()
			}
		}
	case 0xEA:
		// LD (nn) A
		c.opFunc = c.LDDirectA
	case 0xF2:
		// LD a (C)
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				c.writeR8(Z, c.readIndirect(mC))
			case 1:
				c.A = utils.LSB(c.WZ)
				c.DecodeOp()
			}
		}
	case 0xF8:
		// LD HL SP+e8
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				c.Fetch(LO)
			case 1:
				res, hc, carry := c.AddSignedToUnsigned(utils.LSB(c.SP), utils.LSB(c.WZ))
				c.writeR8(L, res)
				c.clearFlags()
				c.setHalfCarry(hc)
				c.setCarry(carry)

			case 2:
				res := c.Adjust(utils.MSB(c.SP), c.getCarry())
				c.writeR8(H, res)
				c.DecodeOp()
			}
		}
	case 0xF9:
		c.opFunc = c.LDSPHL
	case 0xFA:
		// LD A (nn)
		c.opFunc = func() {
			switch c.curCycle {
			case 0:
				c.Fetch(LO)
			case 1:
				c.Fetch(HI)
			case 2:
				c.Read()
			case 3:
				c.SetRegister()
				c.DecodeOp()
			}
		}
	default:
		log.Panicf("OP: %02X, LD %s %s not implemented", c.IR, c.inst.To, c.inst.From)
	}
}

func (c *CPU) LDSPHL() {
	switch c.curCycle {
	case 0:
		c.SP = c.HL
	case 1:
		c.DecodeOp()
	}

}

func (c *CPU) LDn() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.SetRegister()
		c.DecodeOp()
	}
}

func (c *CPU) LDnn() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.Fetch(HI)
	case 2:
		c.SetRegister()
		c.DecodeOp()
	}
}

func (c *CPU) LDDirectA() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.Fetch(HI)
	case 2:
		c.writeMem(c.readR16(WZ), c.readR8(A))
	case 3:
		c.DecodeOp()
	}
}

func (c *CPU) LDHLPlus() {
	switch c.curCycle {
	case 0:
		c.Read()
		c.incrementReg(HL)
	case 1:
		c.SetRegister()
		c.DecodeOp()
	}
}

func (c *CPU) LDHLMinus() {
	switch c.curCycle {
	case 0:
		c.Read() // TODO According to some sources, HL- is done BEFORE, HL+ is after. Verify?
		// IDU cant only post-inc/dec, not pre-inc/dec?
		c.decrementReg(HL)
	case 1:
		c.SetRegister()
		c.DecodeOp()
	}
}

func (c *CPU) setLDHFunc() {
	if c.inst.To == m8 {
		c.opFunc = c.LDHToMem
	} else {
		c.opFunc = c.LDHFromMem
	}
}

func (c *CPU) LDHToMem() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.writeMem(utils.JoinBytes(0xFF, c.readR8(Z)), c.readR8(A))
	case 2:
		c.DecodeOp()
	}
}

func (c *CPU) LDHFromMem() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		val := c.readMem(utils.JoinBytes(0xFF, c.readR8(Z)))
		c.writeR8(Z, val)
	case 2:
		c.A = utils.LSB(c.WZ)
		c.DecodeOp()
	}
}

func (c *CPU) INC() {
	if c.inst.To == mHL {
		switch c.curCycle {
		case 0:
			c.writeR8(Z, c.readIndirect(mHL))
		case 1:
			c.incrementReg(Z)
			c.writeIndirect(mHL, c.readR8(Z))
		case 2:
			c.DecodeOp()
		}
	} else if c.regType(c.inst.To) == R16 {
		switch c.curCycle {
		case 0:
			c.writeR16(c.inst.To, c.IDUInc(c.readR16(c.inst.To)))
		case 1:
			c.DecodeOp()
		}
	} else {
		c.incrementReg(c.inst.To)
		c.DecodeOp()
	}
}

func (c *CPU) DEC() {
	if c.inst.To == mHL {
		switch c.curCycle {
		case 0:
			c.writeR8(Z, c.readIndirect(mHL))
		case 1:
			c.decrementReg(Z)
			c.writeIndirect(mHL, c.readR8(Z))
		case 2:
			c.DecodeOp()
		}
	} else if c.regType(c.inst.To) == R16 {
		switch c.curCycle {
		case 0:
			c.writeR16(c.inst.To, c.IDUDec(c.readR16(c.inst.To)))
		case 1:
			c.DecodeOp()
		}
	} else {
		c.decrementReg(c.inst.To)
		c.DecodeOp()
	}
}

func (c *CPU) setADDFunc() {
	hi := c.IR >> 4
	if hi <= 3 {
		// Add HL rr
		c.opFunc = c.AddHLrr
	} else if hi == 8 {
		if c.IR&0xF == 6 {
			// Add A [HL]
			c.opFunc = c.AddmHL
		} else {
			// Add A r
			c.opFunc = c.AddR8
		}
	} else if c.IR == 0xC6 {
		// Add A n8
		c.opFunc = c.Addn
	} else if c.IR == 0xE8 {
		// Add SP e8
		c.opFunc = c.AddSPe8
	} else {
		panic("unhandled ADD")
	}
}

func (c *CPU) AddHLrr() {
	switch c.curCycle {
	case 0:
		// add lo
		// Z Flag must be preserved for tests to pass
		zFlag := c.getZFlag()
		rr := c.readR16(c.inst.From)
		l := c.add(utils.LSB(c.HL), utils.LSB(rr), false)
		c.writeR8(L, l)
		c.setZFlag(zFlag)
	case 1:
		// add hi
		zFlag := c.getZFlag()
		rr := c.readR16(c.inst.From)
		h := c.add(utils.MSB(c.HL), utils.MSB(rr), true)
		c.writeR8(H, h)
		c.setZFlag(zFlag)
		c.DecodeOp()
	}

}

func (c *CPU) AddmHL() {
	switch c.curCycle {
	case 0:
		// Z <- mem[HL]
		c.Read()
	case 1:
		// A <- A + Z
		c.A = c.add(c.A, utils.LSB(c.WZ), false)
		c.DecodeOp()
	}

}

func (c *CPU) AddR8() {
	// A <- A + r
	c.Read()
	c.A = c.add(c.A, utils.LSB(c.WZ), false)
	c.DecodeOp()

}

func (c *CPU) Addn() {
	switch c.curCycle {
	case 0:
		// Z <- mem[PC]
		c.Fetch(LO)
	case 1:
		// A <- A + Z
		c.A = c.add(c.A, utils.LSB(c.WZ), false)
		c.DecodeOp()
	}
}

func (c *CPU) AddSPe8() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		res, hc, carry := c.AddSignedToUnsigned(utils.LSB(c.SP), utils.LSB(c.WZ))
		c.writeR8(Z, res)
		c.clearFlags()
		c.setHalfCarry(hc)
		c.setCarry(carry)

	case 2:
		res := c.Adjust(utils.MSB(c.SP), c.getCarry())
		c.writeR8(W, res)
	case 3:
		c.SP = c.WZ
		c.DecodeOp()
	}
}

func (c *CPU) setADCFunc() {
	if c.inst.DataType == N8 {
		c.opFunc = c.adcn
	} else {
		if c.inst.From == mHL {
			c.opFunc = c.adcIndirect
		} else {
			c.opFunc = c.adcr
		}
	}
}

func (c *CPU) adcn() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.A = c.add(c.A, c.readR8(Z), true)
		c.DecodeOp()
	}
}

func (c *CPU) adcIndirect() {
	switch c.curCycle {
	case 0:
		c.writeR8(Z, c.readIndirect(mHL))
	case 1:
		c.A = c.add(c.A, c.readR8(Z), true)
		c.DecodeOp()
	}
}

func (c *CPU) adcr() {
	c.A = c.add(c.A, c.readR8(c.inst.From), true)
	c.DecodeOp()
}

func (c *CPU) setSUBFunc() {
	if c.inst.DataType == N8 {
		c.opFunc = c.subn
	} else {
		if c.inst.From == mHL {
			c.opFunc = c.subIndirect
		} else {
			c.opFunc = c.subr
		}
	}
}

func (c *CPU) subn() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.A = c.sub(c.A, c.readR8(Z), false)
		c.DecodeOp()
	}
}

func (c *CPU) subIndirect() {
	switch c.curCycle {
	case 0:
		c.writeR8(Z, c.readIndirect(mHL))
	case 1:
		c.A = c.sub(c.A, c.readR8(Z), false)
		c.DecodeOp()
	}
}

func (c *CPU) subr() {
	c.A = c.sub(c.A, c.readR8(c.inst.From), false)
	c.DecodeOp()
}

func (c *CPU) setSBCFunc() {
	if c.inst.DataType == N8 {
		c.opFunc = c.sbcn
	} else {
		if c.inst.From == mHL {
			c.opFunc = c.sbcIndirect
		} else {
			c.opFunc = c.sbcr
		}
	}
}

func (c *CPU) sbcn() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.A = c.sub(c.A, c.readR8(Z), true)
		c.DecodeOp()
	}
}

func (c *CPU) sbcIndirect() {
	switch c.curCycle {
	case 0:
		c.writeR8(Z, c.readIndirect(mHL))
	case 1:
		c.A = c.sub(c.A, c.readR8(Z), true)
		c.DecodeOp()
	}
}

func (c *CPU) sbcr() {
	c.A = c.sub(c.A, c.readR8(c.inst.From), true)
	c.DecodeOp()
}

func (c *CPU) setANDFunc() {
	if c.inst.DataType == N8 {
		c.opFunc = c.andn
	} else {
		if c.inst.From == mHL {
			c.opFunc = c.andIndirect
		} else {
			c.opFunc = c.andr
		}
	}
}

func (c *CPU) andn() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.A = c.and(c.A, c.readR8(Z))
		c.DecodeOp()
	}
}

func (c *CPU) andIndirect() {
	switch c.curCycle {
	case 0:
		c.writeR8(Z, c.readIndirect(mHL))
	case 1:
		c.A = c.and(c.A, c.readR8(Z))
		c.DecodeOp()
	}
}

func (c *CPU) andr() {
	c.A = c.and(c.A, c.readR8(c.inst.From))
	c.DecodeOp()
}

// Set intermediate registers W/Z to value from a certain source, usually specified in Instruction
func (c *CPU) Read() {
	switch c.regType(c.inst.From) {
	case R8:
		c.writeR8(Z, c.readR8(c.inst.From))
	case R16:
		c.writeR16(WZ, c.readR16(c.inst.From))
	case DIRECT:
		var addr uint16
		if c.inst.From == m8 {
			addr = utils.JoinBytes(0xFF, c.readR8(Z))
		} else {
			addr = c.readR16(WZ)
		}
		c.writeR8(Z, c.readMem(addr))
	case INDIRECT:
		c.writeR8(Z, c.readIndirect(c.inst.From))
	default:
		log.Panicf("unhandled c.Read %+v", c.inst)
	}
}

// Read next byte and increment PC
func (c *CPU) Fetch(hilo register) {
	if hilo == HI {
		c.writeR8(W, c.imm8())
	} else if hilo == LO {
		c.writeR8(Z, c.imm8())
	}
}

func (c *CPU) FetchIR(prefix bool) (interrupted bool) {
	if GAMEBOY_DOCTOR && !prefix && !c.skipLog {
		// c := b.cpu
		B := utils.MSB(c.BC)
		C := utils.LSB(c.BC)
		D := utils.MSB(c.DE)
		E := utils.LSB(c.DE)
		H := utils.MSB(c.HL)
		L := utils.LSB(c.HL)
		pcm := []byte{}
		for i := 0; i < 4; i++ {
			pcm = append(pcm, c.bus.Read(c.PC+uint16(i)))
		}
		fmt.Fprintf(logfile, "A:%02X F:%02X B:%02X C:%02X D:%02X E:%02X H:%02X L:%02X SP:%04X PC:%04X PCMEM:%02X,%02X,%02X,%02X\n", c.A, c.F, B, C, D, E, H, L, c.SP, c.PC, pcm[0], pcm[1], pcm[2], pcm[3])
	}

	c.curCycle = 0xFF // after fetch, will be incremented to 0

	if !prefix && c.CheckInterrupts() {
		return true
	}

	c.instAddr = c.PC
	c.IR = c.imm8()

	if c.haltBug {
		c.PC--
		c.haltBug = false
	}

	c.inst = lookup(c.IR, prefix)

	c.skipLog = false
	// if c.inst.Op == "NOP" {
	//   c.skipLog = true
	// }

	return false
}

func (c *CPU) DecodeOp() {

	if intr := c.FetchIR(false); intr {
		return
	}

	c.SetOpFunc()

}

func (c *CPU) SetOpFunc() {
	switch c.inst.Op {
	case "STOP":
		// c.bus.halted = true
		// c.bus.DIV = 0
		c.opFunc = c.NOP
		// TODO, STOP does a lot more than NOP, but this is here just to pass blargg's cpu_instrs test. CGB needs it for speed switching, DMG not so much.
	case "NOP":
		c.opFunc = c.NOP
	case "HALT":
		// Pause CPU until interrupt pending.

		// TODO
		// Documented "halt bug",
		// If IME == 0, but IE & IF != 0, halt ends immediately but PC does not increment
		// causing the following instruction to be read twice.
		// If halt comes immediately after ei, the return from the interrupt handler will be the halt command again
		// If halt is followed by rst, rst will return to itself

		c.opFunc = func() {
			c.bus.setHalt(true)
			if c.IME == 0 && (c.IE&c.IF != 0) {
				c.bus.setHalt(false)
				c.haltBug = true
				c.DecodeOp()
			}
		}

	case "CP":
		c.setCPFunc()
	case "JP":
		c.setJPFunc()
	case "JR":
		c.opFunc = c.JR
	case "OR":
		c.setORFunc()
	case "XOR":
		c.setXORFunc()
	case "LD":
		c.setLDFunc()
	case "LDH":
		c.setLDHFunc()
	case "INC":
		c.opFunc = c.INC
	case "DEC":
		c.opFunc = c.DEC
	case "ADD":
		c.setADDFunc()
	case "ADC":
		c.setADCFunc()
	case "SUB":
		c.setSUBFunc()
	case "SBC":
		c.setSBCFunc()
	case "AND":
		c.setANDFunc()
	case "RRA":
		c.opFunc = c.RRA
	case "RRCA":
		c.opFunc = c.RRCA
	case "RLA":
		c.opFunc = c.RLA
	case "RLCA":
		c.opFunc = c.RLCA
	case "CPL":
		c.opFunc = c.CPL
	case "SCF":
		c.opFunc = c.SCF
	case "EI":
		c.opFunc = c.SetIME
	case "DI":
		c.opFunc = c.UnsetIME
	case "CALL":
		c.opFunc = c.CALL
	case "RET":
		c.SetRETFunc()
	case "RETI":
		c.opFunc = c.RETI
	case "RST":
		c.opFunc = c.RST
	case "PUSH":
		c.opFunc = c.PUSH
	case "POP":
		c.opFunc = c.POP
	case "DAA":
		c.opFunc = c.DAA
	case "CCF":
		c.opFunc = c.CCF
	case "PREFIX":
		c.opFunc = c.DecodePrefix
	default:
		inst := c.inst
		log.Panicf("unimplemented op: %s/0x%02X, dt: %v, to: %s, from: %s, flag: %s", inst.Op, c.IR, inst.DataType, inst.To, inst.From, inst.Flag)
	}

}

// Write a byte to a specific memory location.
func (c *CPU) Write() {
	tt := c.regType(c.inst.To)
	if c.inst.To == m8 {
		c.writeMem(utils.JoinBytes(0xFF, c.readR8(Z)), c.readR8(A))
		// } else if c.inst.To == m16 {

		// }
	} else if tt == INDIRECT {
		c.writeIndirect(c.inst.To, c.readR8(Z))
	} else if tt == DIRECT {
		c.writeMem(c.readR16(WZ), c.readR8(A))
	} else {
		log.Panicf("unhandled c.Write %+v", c.inst)
	}
}

func (c *CPU) WriteSP(hilo string) func() {
	return func() {
		if hilo == HI {
			c.writeMem(c.readR16(WZ)+1, utils.MSB(c.SP))
		} else if hilo == LO {
			c.writeMem(c.readR16(WZ), utils.LSB(c.SP))
		}
	}
}

// Set a certain register to the value of W/Z.
func (c *CPU) SetRegister() {

	rt := c.regType(c.inst.To)
	switch rt {
	case R8:
		c.ALUBusy = true
		switch c.regType(c.inst.From) {
		// TODO, delete switch?
		case DIRECT:
			// if c.inst.Op == "LDH" {
			c.writeR8(A, c.readR8(Z))
			// } else {
			// log.Panicf("OP %s wants to set %s from DIRECT", c.inst.Op, c.inst.To)
			// }
		case INDIRECT:
			// c.writeR8(c.inst.To, c.readIndirect(c.inst.From))
			fallthrough
		case R8, NONE:
			c.writeR8(c.inst.To, c.readR8(Z))
		}
	case R16:
		c.writeR16(c.inst.To, c.readR16(WZ))
		// c.WZ = 0
	case INDIRECT:
		c.writeIndirect(c.inst.To, c.readR8(Z))
	default:
		log.Panicf("c.SetRegister unhandled regType %s (%s)", rt, c.inst.To)
	}
}

// Do nothing for one cycle
func (c *CPU) NOP() {
	c.DecodeOp()
}

// Set the PC register to value of WZ.
func (c *CPU) SetPC() {
	c.PC = c.WZ
}

// Add a signed byte to PC.
func (c *CPU) AddRelPC() {
	z, _, carry := c.AddSignedToUnsigned(utils.LSB(c.PC), utils.LSB(c.WZ))
	w := c.Adjust(utils.MSB(c.PC), carry)
	c.WZ = utils.JoinBytes(w, z)
}

// Set IME to 1
func (c *CPU) SetIME() {
	// c.untilIME = 2
	// c.setIME = true
	c.DecodeOp()
	c.IME = 1
}

// Set IME to 0
func (c *CPU) UnsetIME() {
	c.IME = 0
	// c.setIME = false
	c.DecodeOp()
}

func (c *CPU) CALL() {
	switch c.curCycle {
	case 0:
		c.Fetch(LO)
	case 1:
		c.Fetch(HI)
		c.CheckFlag()
	case 2:
		if c.FlagMatches() {
			c.decrementReg(SP)
		} else {
			c.DecodeOp()
		}
	case 3:
		c.pushPCToStack(HI)
		c.decrementReg(SP)
	case 4:
		c.pushPCToStack(LO)
		c.SetPC()
	case 5:
		c.DecodeOp()
	}
}

func (c *CPU) SetRETFunc() {
	if c.inst.Flag == NOFLAG {
		c.opFunc = c.RET
	} else {
		c.opFunc = c.RETWithConditional
	}
}

func (c *CPU) RET() {
	switch c.curCycle {
	case 0:
		c.popFromStack(LO)
		c.incrementReg(SP)
	case 1:
		c.popFromStack(HI)
		c.incrementReg(SP)
	case 2:
		c.SetPC()
	case 3:
		c.DecodeOp()
	}
}

func (c *CPU) RETWithConditional() {
	switch c.curCycle {
	case 0:
		c.CheckFlag()
	case 1:
		if c.FlagMatches() {
			c.popFromStack(LO)
			c.incrementReg(SP)
		} else {
			c.DecodeOp()
		}
	case 2:
		c.popFromStack(HI)
		c.incrementReg(SP)
	case 3:
		c.SetPC()
	case 4:
		c.DecodeOp()
	}
}

func (c *CPU) RETI() {
	switch c.curCycle {
	case 0:
		c.popFromStack(LO)
		c.incrementReg(SP)
	case 1:
		c.popFromStack(HI)
		c.incrementReg(SP)
	case 2:
		c.SetPC()
		c.IME = 1
	case 3:
		c.DecodeOp()
	}
}

func (c *CPU) RST() {
	switch c.curCycle {
	case 0:
		c.decrementReg(SP)
	case 1:
		c.pushPCToStack(HI)
		c.decrementReg(SP)
	case 2:
		c.pushPCToStack(LO)
		c.writeR16(WZ, utils.JoinBytes(0x00, c.inst.Abs))
		c.SetPC()
	case 3:
		c.DecodeOp()
	}
}

func (c *CPU) PUSH() {
	switch c.curCycle {
	case 0:
		c.decrementReg(SP)
	case 1:
		c.pushToStack(HI)
		c.decrementReg(SP)
	case 2:
		c.pushToStack(LO)
	case 3:
		c.DecodeOp()
	}
}

func (c *CPU) POP() {
	switch c.curCycle {
	case 0:
		c.popFromStack(LO)
		c.incrementReg(SP)
	case 1:
		c.popFromStack(HI)
		c.incrementReg(SP)
	case 2:
		c.writeR16(c.inst.To, c.WZ)
		c.DecodeOp()
	}
}

// Decimal Adjust Accumulator
// Adjust binary coded decimal. This is done after an instruction that adds 2 hex numbers.
// In hexadecimal, 0x16 + 0x15 = 0x2B
// In BCD, 0x16 + 0x15 = 0x31
// DAA makes this adjustment.
func (c *CPU) DAA() {
	result, carry := c.DecAdj(c.A, c.F)

	c.setHalfCarry(0)
	c.setCarry(carry)

	c.setZFlag(result)
	c.A = result

	c.DecodeOp()
}

// Complement Carry Flag
func (c *CPU) CCF() {
	c.setNegFlag(0)
	c.setHalfCarry(0)
	c.setCarry(1 - c.getCarry())
	c.DecodeOp()
}

// PREFIX FUNCS

// Swap hi and lo nibbles of byte
func (c *CPU) Swap() {
	r := c.readR8(Z)
	swapped := c.ALUSwap(r)
	c.writeR8(Z, swapped)
	c.clearFlags()
	c.setZFlag(swapped)

}

// Check bit of byte
func (c *CPU) Bit() {
	r := c.readR8(Z)
	b := utils.GetBit(c.inst.Bit, r)
	c.setZFlag(b)
	c.setNegFlag(0)
	c.setHalfCarry(1)

}

// Set bit of byte
func (c *CPU) Set() {
	var r byte
	r = c.readR8(Z)
	r = utils.SetBit(c.inst.Bit, r)
	c.writeR8(Z, r)
}

// Reset bit
func (c *CPU) Res() {
	var r byte
	r = c.readR8(Z)
	r = utils.ResetBit(c.inst.Bit, r)
	c.writeR8(Z, r)
}

// Shift right arithmetic
func (c *CPU) SRA() {
	data := c.readR8(Z)
	carry := utils.GetBit(0, data)
	data >>= 1
	if utils.IsBitSet(6, data) {
		data = utils.SetBit(7, data)
	}
	c.writeR8(Z, data)
	c.clearFlags()
	c.setZFlag(data)
	c.setCarry(carry)
}

// Shift left arithmetic
func (c *CPU) SLA() {
	data := c.readR8(Z)
	carry := utils.GetBit(7, data)
	data <<= 1
	c.writeR8(Z, data)
	c.clearFlags()
	c.setZFlag(data)
	c.setCarry(carry)
}

// Shift right logical, don't wrap bits
func (c *CPU) SRL() {
	data := c.readR8(Z)
	carry := utils.GetBit(0, data)
	result := data >> 1
	c.writeR8(Z, result)
	c.clearFlags()
	c.setZFlag(result)
	c.setCarry(carry)
}

// Rotate right accumulator
func (c *CPU) RRA() {
	data := c.readR8(A)
	oldCarry := c.getCarry()
	carry := utils.GetBit(0, data)
	result := c.rotateRight(data, oldCarry)

	c.writeR8(A, result)
	c.clearFlags() // TODO, docs say Z should be 0. Is this right?
	c.setCarry(carry)

	c.DecodeOp()
}

// Rotate right circular accumulator
func (c *CPU) RRCA() {
	data := c.readR8(A)
	carry := utils.GetBit(0, data)
	result := c.rotateRight(data, carry)

	c.writeR8(A, result)
	c.clearFlags()
	c.setCarry(carry)
	c.DecodeOp()
}

// Rotate right circular
func (c *CPU) RRC() {
	data := c.readR8(Z)
	carry := utils.GetBit(0, data)
	result := c.rotateRight(data, carry)

	c.writeR8(Z, result)
	c.clearFlags()
	c.setZFlag(result)
	c.setCarry(carry)
}

// Rotate right through carry
func (c *CPU) RR() {
	data := c.readR8(Z)
	oldCarry := c.getCarry()
	carry := utils.GetBit(0, data)
	result := c.rotateRight(data, oldCarry)

	c.writeR8(Z, result)
	c.clearFlags()
	c.setZFlag(result)
	c.setCarry(carry)
}

// Rotate left accumulator
func (c *CPU) RLA() {
	data := c.readR8(A)
	oldCarry := c.getCarry()
	carry := utils.GetBit(7, data)
	result := c.rotateLeft(data, oldCarry)

	c.writeR8(A, result)
	c.clearFlags()
	c.setCarry(carry)
	c.DecodeOp()
}

// Rotate left circular accumulator
func (c *CPU) RLCA() {
	data := c.readR8(A)
	carry := utils.GetBit(7, data)
	result := c.rotateLeft(data, carry)

	c.writeR8(A, result)
	c.clearFlags()
	c.setCarry(carry)
	c.DecodeOp()
}

// Rotate left circular
func (c *CPU) RLC() {
	data := c.readR8(Z)
	carry := utils.GetBit(7, data)
	result := c.rotateLeft(data, carry)

	c.writeR8(Z, result)
	c.clearFlags()
	c.setZFlag(result)
	c.setCarry(carry)
}

// Rotate left
func (c *CPU) RL() {
	data := c.readR8(Z)
	oldCarry := c.getCarry()
	carry := utils.GetBit(7, data)
	result := c.rotateLeft(data, oldCarry)

	c.writeR8(Z, result)
	c.clearFlags()
	c.setZFlag(result)
	c.setCarry(carry)
}

// Complement accumulator
func (c *CPU) CPL() {
	c.writeR8(A, ^c.readR8(A))
	c.setNegFlag(1)
	c.setHalfCarry(1)

	c.DecodeOp()
}

// Set carry flag
func (c *CPU) SCF() {
	c.setNegFlag(0)
	c.setHalfCarry(0)
	c.setCarry(1)
	c.DecodeOp()
}

// Functions used by main op funcs

// Read the memory address at [PC]
func (c *CPU) imm8() byte {
	n8 := c.readMem(c.PC)
	c.incrementReg(PC)
	return n8
}

func (c *CPU) rotateRight(data, carry byte) byte {
	return (data >> 1) | (carry << 7)
}
func (c *CPU) rotateLeft(data, carry byte) byte {
	return (data << 1) | carry
}

// Read the memory address at [addr]
func (c *CPU) readMem(addr uint16) byte {
	return c.bus.Read(addr)
}

// Write byte to memory address [addr]
func (c *CPU) writeMem(addr uint16, data byte) {
	c.bus.Write(addr, data)
}

func (c *CPU) regType(reg register) string {
	switch reg {
	case A, B, C, D, E, H, L, Z:
		return R8
	case BC, DE, HL, SP, PC, AF:
		return R16
	case mBC, mDE, mHL, mHLp, mHLm, mC:
		return INDIRECT
	case m8, m16:
		return DIRECT
	default:
		return NONE
	}
}

func (c *CPU) readR8(reg register) byte {
	switch reg {
	case A:
		return c.A
	case B:
		return utils.MSB(c.BC)
	case C:
		return utils.LSB(c.BC)
	case D:
		return utils.MSB(c.DE)
	case E:
		return utils.LSB(c.DE)
	case H:
		return utils.MSB(c.HL)
	case L:
		return utils.LSB(c.HL)
	case Z:
		return utils.LSB(c.WZ)
	default:
		log.Panicf("op: %02X, tried to get value from unhandled R8 %s", c.IR, reg)
	}
	return 0
}

func (c *CPU) readR16(reg register) uint16 {
	switch reg {
	case BC:
		return c.BC
	case DE:
		return c.DE
	case HL:
		return c.HL
	// case m16:
	// 	return c.imm16()
	case SP:
		return c.SP
	case AF:
		return utils.JoinBytes(c.A, c.F)
	case WZ:
		// ret := c.WZ
		// c.WZ = 0
		return c.WZ
	case PC:
		return c.PC
	default:
		log.Panicf("tried to get value from unhandled reg16 %s", reg)
	}
	return 0
}

func (c *CPU) readIndirect(reg register) byte {
	switch reg {
	case mC:
		return c.bus.Read(utils.JoinBytes(0xFF, utils.LSB(c.BC)))
	case mBC:
		return c.bus.Read(c.BC)
	case mDE:
		return c.bus.Read(c.DE)
	case mHL:
		return c.bus.Read(c.HL)
	case mHLp:
		val := c.bus.Read(c.HL)
		// c.HL = c.IDUInc(c.HL)
		return val
	case mHLm:
		val := c.bus.Read(c.HL)
		// c.HL = c.IDUDec(c.HL)
		return val
	default:
		log.Panicf("op: %02X, tried to get value from unhandled indirect %s", c.IR, reg)
	}
	return 0
}

func (c *CPU) writeR8(reg register, value byte) {
	switch reg {
	case A:
		c.A = value
	case B:
		c.BC = utils.JoinBytes(value, utils.LSB(c.BC))
	case C:
		c.BC = utils.JoinBytes(utils.MSB(c.BC), value)
	case D:
		c.DE = utils.JoinBytes(value, utils.LSB(c.DE))
	case E:
		c.DE = utils.JoinBytes(utils.MSB(c.DE), value)
	case H:
		c.HL = utils.JoinBytes(value, utils.LSB(c.HL))
	case L:
		c.HL = utils.JoinBytes(utils.MSB(c.HL), value)
	case W:
		c.WZ = utils.JoinBytes(value, utils.LSB(c.WZ))
	case Z:
		c.WZ = utils.JoinBytes(utils.MSB(c.WZ), value)
	default:
		log.Panicf("op: %02X, tried to set value for unhandled R8 %s", c.IR, reg)
	}
}

func (c *CPU) writeR16(reg register, value uint16) {
	switch reg {
	case BC:
		c.BC = value
	case DE:
		c.DE = value
	case HL:
		c.HL = value
	case SP:
		c.SP = value
	case PC:
		c.PC = value
	case AF:
		c.A = utils.MSB(value)
		c.F = utils.LSB(value) & 0xF0
	case WZ:
		c.WZ = value
	default:
		log.Panicf("tried to set value for unhandled reg16 %s", reg)
	}
}

func (c *CPU) writeIndirect(reg register, value uint8) {
	switch reg {
	case mBC:
		c.bus.Write(c.BC, value)
	case mC:
		c.bus.Write(utils.JoinBytes(0xFF, utils.LSB(c.BC)), value)
	case mDE:
		c.bus.Write(c.DE, value)
	case mHL:
		c.bus.Write(c.HL, value)
	case mHLp:
		c.bus.Write(c.HL, value)
		// c.HL = c.IDUInc(c.HL)
	case mHLm:
		c.bus.Write(c.HL, value)
		// c.HL = c.IDUDec(c.HL)
	default:
		log.Panicf("op: %02X, tried to set value for unhandled indirect %s", c.IR, reg)
	}
}

// Check if inst.Flag is set, save result for future cycles
func (c *CPU) CheckFlag() {
	switch c.inst.Flag {
	case ZERO:
		c.flagMatched = utils.IsBitSet(7, c.F)
	case NZ:
		c.flagMatched = !utils.IsBitSet(7, c.F)
	case CARRY:
		c.flagMatched = utils.IsBitSet(4, c.F)
	case NC:
		c.flagMatched = !utils.IsBitSet(4, c.F)
	case NOFLAG:
		c.flagMatched = true
	default:
		log.Panicf("unhandled flag check for flag %s", c.inst.Flag)
	}
}

// Return value set by CheckFlag()
func (c *CPU) FlagMatches() bool {
	return c.flagMatched
}

func (c *CPU) clearFlags() {
	c.F &= 0x00
}

// Set Zero flag based on given value
// If value == 0, Z = 1, else Z = 0
func (c *CPU) setZFlag(value byte) {
	if value == 0 {
		c.F |= 0x80
	} else {
		c.F &= 0x7F
	}
}

func (c *CPU) getZFlag() byte {
	return utils.GetBit(7, c.F)
}

func (c *CPU) setNegFlag(to byte) {
	if to == 0 {
		c.F &= 0xBF
	} else {
		c.F |= 0x40
	}
}

// func (c *CPU) setHalfCarryAdd(a, b byte) {
// 	result := a + b
// 	halfCarry := ((a ^ b ^ result) & 0x10) >> 4
// 	c.setHalfCarry(halfCarry)
// }

func (c *CPU) setHalfCarrySub(a, b byte, checkCarry bool) {
	// result := a - b
	aNib := a & 0xF
	bNib := b & 0xF
	sum := aNib - bNib
	if checkCarry {
		sum -= c.getCarry()
	}
	halfCarry := sum & 0x10
	// halfCarry := ((a ^ -b ^ result) & 0x10) >> 4
	c.setHalfCarry(halfCarry >> 4)
}

func (c *CPU) setHalfCarry(to byte) {
	if to == 1 {
		c.F |= 0x20
	} else {
		c.F &= 0xDF
	}
}

func (c *CPU) getHalfCarry() byte {
	return utils.GetBit(5, c.F)
}

func (c *CPU) setCarry(setTo byte) {
	if setTo != 0 {
		c.F |= 0x10
	} else {
		c.F &= 0xEF
	}
}

func (c *CPU) getCarry() byte {
	return utils.GetBit(4, c.F)
}

func (c *CPU) pushToStack(hiOrLo string) {
	if hiOrLo == HI {
		c.writeMem(c.SP, utils.MSB(c.readR16(c.inst.From)))
	} else {
		c.writeMem(c.SP, utils.LSB(c.readR16(c.inst.From)))
	}
}

func (c *CPU) popFromStack(hiOrLo string) {
	if hiOrLo == HI {
		c.writeR8(W, c.readMem(c.SP))
	} else {
		c.writeR8(Z, c.readMem(c.SP))
	}
}

func (c *CPU) pushPCToStack(hiOrLo string) {
	if hiOrLo == HI {
		c.writeMem(c.SP, utils.MSB(c.PC))
	} else {
		c.writeMem(c.SP, utils.LSB(c.PC))
	}
}

func (c *CPU) incrementReg(r register) {
	rt := c.regType(r)
	if rt == R8 {
		in := c.readR8(r)
		result, hc := c.ALUInc(in)
		c.setZFlag(result)
		c.setNegFlag(0)
		c.setHalfCarry(hc)
		c.writeR8(r, result)
	} else if rt == R16 {
		c.writeR16(r, c.IDUInc(c.readR16(r)))
	}
}

func (c *CPU) decrementReg(r register) {
	rt := c.regType(r)
	if rt == R8 {
		in := c.readR8(r)
		result, hc := c.ALUDec(in)
		c.setZFlag(result)
		c.setNegFlag(1)
		c.setHalfCarry(hc)
		c.writeR8(r, result)
	} else if rt == R16 {
		c.writeR16(r, c.IDUDec(c.readR16(r)))
	}
}

func (c *CPU) add(a, b byte, useCarry bool) (result byte) {
	var hc, newCarry byte

	if useCarry {
		result, hc, newCarry = c.ALUAdd(a, b, c.getCarry())
	} else {
		result, hc, newCarry = c.ALUAdd(a, b, 0)
	}

	c.setZFlag(result)
	c.setNegFlag(0)
	c.setHalfCarry(hc)
	c.setCarry(newCarry)

	return
}

func (c *CPU) sub(a, b byte, useCarry bool) (result byte) {
	carry := c.getCarry()
	var hc, newCarry byte

	if useCarry {
		result, hc, newCarry = c.ALUSub(a, b, carry)
	} else {
		result, hc, newCarry = c.ALUSub(a, b, 0)
	}

	c.setZFlag(result)
	c.setNegFlag(1)
	c.setHalfCarry(hc)
	c.setCarry(newCarry)

	return
}

func (c *CPU) and(a, b byte) (result byte) {
	result = c.ALUAnd(a, b)
	c.setZFlag(result)
	c.setNegFlag(0)
	c.setHalfCarry(1)
	c.setCarry(0)

	return
}
