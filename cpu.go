package main

import (
	// "fmt"
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

type ALU struct {
	ALUBusy bool
}

func (alu *ALU) ALUInc(val byte) byte {
	return val + 1
}

func (alu *ALU) ALUDec(val byte) byte {
	return val - 1
}

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
	IF                 byte   // Interrupt Flag
	IE                 byte   // Interrupt Enable
	BC, DE, HL, PC, SP uint16 // may need to set sp to start at top of stack range
	WZ                 uint16 // intermediate 16 bit register
}

type ControlUnit struct {
}

type CPU struct {
	ALU
	IDU
	RegisterFile
	ControlUnit
	bus                *Bus
	interruptStep      int
	interruptAddr      uint8
	interruptAddresses []uint8
	// TODO, when I implement accurate(?) M-Cycles, use a function queue that pops a function per cycle. Then cleanup interrupt handling.
	funcQueue []func()
	// these  bools are used to make functions wait for clear data lines / units
	canFetchOp  bool
	addrBusFull bool
	dataBusFull bool
	miscOp      bool

	inst Instruction // current instruction
}

func NewCPU() *CPU {
	return &CPU{
		ALU: ALU{},
		IDU: IDU{},
		// RegisterFile{PC: 0xFF},
		RegisterFile: RegisterFile{
			PC: 0x0100,
			A:  0x01,
			F:  0xB0,
			BC: 0x0013,
			DE: 0x00D8,
			HL: 0x014D,
			SP: 0xFFFE,
		}, // After header
		ControlUnit:        ControlUnit{},
		interruptAddresses: []uint8{0x40, 0x48, 0x50, 0x58, 0x60}, // V-Blank, LCDC, Timer, Serial, Joypad
		funcQueue:          []func(){},
		canFetchOp:         true,
		inst:               lookup(0x00, false),
	}
}

func (c *CPU) Cycle() {
	// According to a reddit comment, normal cpu cycle is T-cycle 1, but interrupt checks are during T3?
	// Check interrupt bytes
	if c.IF != 0 && c.IE != 0 {
		c.bus.halted = false
		if c.IME == 1 {
			for bit := 0; bit <= 4; bit++ {
				if isBitSet(bit, c.IF) && isBitSet(bit, c.IE) {
					// c.interruptStep = 5
					c.IME = 0
					c.interruptAddr = c.interruptAddresses[bit]
					c.IF = resetBit(bit, c.IF)

					// TODO, if interrupted instruction needs to be remembered, this will need to be tweaked.
					c.IR = 0xC7 // RST
					c.inst = lookup(c.IR, false)
          c.inst.Op = "INT"
					// c.inst.Abs = c.interruptAddr
					c.WZ = joinBytes(0x00, c.interruptAddr)

					// prepend interrupt funcs
					c.PushToFuncQueue(
						c.Nop, c.Nop,
						c.PushToStack(HI), c.PushToStack(LO),
						c.SetPC,
					)
					return
					// log.Panicf("interrupted %d", len(c.funcQueue))
				}
			}
		}
	}
	if !c.bus.halted {

		// R8 <- R8 handled by ALU
		// R8 <- mem, mem <- R8 uses data bus, R16 in addr bus
		// R16 +/- 1 uses IDU
		// R16 <- R16 uses "Misc op"
		c.addrBusFull = false
		c.dataBusFull = false
		c.IDUBusy = false
		c.ALUBusy = false
		c.miscOp = false
		c.canFetchOp = true

		// Execute next function
		if len(c.funcQueue) > 0 {
			nextFn := c.funcQueue[0]
			c.funcQueue = c.funcQueue[1:]
			nextFn()

			if c.addrBusFull || c.dataBusFull || c.IDUBusy {
				c.canFetchOp = false // TODO, check docs for what matters
			}
		}

		if len(c.funcQueue) == 0 && c.canFetchOp {
			// log.Panicf("IF: %02X, IE: %02X, IME: %02X, funcQueue len: %d", c.IF, c.IE, c.IME, len(c.funcQueue))
			// } else {
			c.FetchIR()

			switch c.inst.Op {
			case "STOP": 
        // c.bus.halted = true
        // c.bus.DIV = 0
				c.PushToFuncQueue(c.Nop)
      // TODO, STOP does a lot more than NOP, but this is here just to pass blargg's cpu_instrs test. CGB needs it for speed switching, DMG not so much.
			case "NOP":
				c.PushToFuncQueue(c.Nop)
			case "HALT":
				// Pause CPU until interrupt pending.

				// TODO
				// Documented "halt bug",
				// If IME == 0, but IE & IF != 0, halt ends immediately but PC does not increment
				// causing the following instruction to be read twice.
				// If halt comes immediately after ei, the return from the interrupt handler will be the halt command again
				// If halt is followed by rst, rst will return to itself
				c.bus.halted = true
			case "CP":
				if c.inst.DataType == N8 {
					c.PushToFuncQueue(c.Fetch(LO))
				} else if c.inst.From == mHL {
					c.PushToFuncQueue(c.Read)
				} else {
					c.Read()
				}
				c.PushToFuncQueue(c.Compare)
			case "JP":
				// TODO conditional flag
				// jp nn, jp HL, jp cc nn
				switch c.inst.DataType {
				// if jp HL, set c.WZ to c.HL TODO
				case A16:
					c.PushToFuncQueue(c.Fetch(LO), c.Fetch(HI))
					if c.inst.Flag != NOFLAG {
						if c.FlagIsSet() {
							c.PushToFuncQueue(c.SetPC)
						}

					} else {
						c.PushToFuncQueue(c.SetPC)
					}
				case NODATA:
					c.Read()
					c.PushToFuncQueue(c.SetPC)
				default:
					log.Panicf("unhandled JP datatype")
				}
			case "JR":
				c.PushToFuncQueue(c.Fetch(LO))
				if c.flagMatches(c.inst.Flag) {
					c.PushToFuncQueue(c.AddRelPC, c.SetPC)
				}
			case "OR":
				if c.inst.DataType == N8 {
					c.PushToFuncQueue(c.Fetch(LO))
				} else if c.inst.From == mHL {
					c.PushToFuncQueue(c.Read)
				} else {
					c.Read()
				}
				c.PushToFuncQueue(c.Or)
			case "XOR":
				if c.inst.DataType == N8 {
					c.PushToFuncQueue(c.Fetch(LO))
				} else if c.inst.From == mHL {
					c.PushToFuncQueue(c.Read)
				} else {
					c.Read()
				}
				c.PushToFuncQueue(c.Xor)
			case "LD":
				// LD r, r 1 cycle
				// LD r, n8 2 cycles
				// LD rr, n16 3 cycles
				// LD a16, A 4 cycles
				// LD a16, SP 5 cycles
				// LD HL, SP+e 3 cycles
				switch c.inst.DataType {
				case NODATA:
					if c.regType(c.inst.From) == INDIRECT {
						c.PushToFuncQueue(c.Read)
					} else {
						c.Read()
					}
					if c.regType(c.inst.To) == INDIRECT {
						c.PushToFuncQueue(c.Write)
					} else {
						c.PushToFuncQueue(c.SetRegister)
					}
				case N8:
					c.PushToFuncQueue(c.Fetch(LO), c.SetRegister)
				case N16:
					c.PushToFuncQueue(c.Fetch(LO), c.Fetch(HI), c.SetRegister)
				case A16:
					c.PushToFuncQueue(c.Fetch(LO), c.Fetch(HI))
					if c.inst.To == m16 {
						if c.inst.From == SP {
							c.PushToFuncQueue(c.WriteSP(LO), c.WriteSP(HI))
						} else {
							c.PushToFuncQueue(c.Write)
						}
					} else {
						c.PushToFuncQueue(c.Read, c.SetRegister)
					}
				case E8:
					c.PushToFuncQueue(c.Fetch(LO), c.AddRelSP, c.SetRegister)
				default:
					log.Panicf("LD %s %s not implemented", c.inst.To, c.inst.From)
				}
			case "LDH":
				c.PushToFuncQueue(c.Fetch(LO))
				if c.inst.To == m8 {
					c.PushToFuncQueue(c.Write)
				} else if c.inst.From == m8 {
					c.PushToFuncQueue(c.Read, c.SetRegister)
				}
			case "INC":
				if c.inst.To == mHL {
					c.PushToFuncQueue(c.Read, c.INC, c.Write)
				} else {
					c.PushToFuncQueue(c.INC) // 16 bit inc and dec take an extra cycle because pc requires same unit for incrementation. 8 bit incs and decs use ALU, 16 bit uses IDU.
				}
			case "DEC":
				if c.inst.To == mHL {
					c.PushToFuncQueue(c.Read, c.DEC, c.Write)
				} else {
					c.PushToFuncQueue(c.DEC) // 16 bit inc and dec take an extra cycle because pc requires same unit for incrementation. 8 bit incs and decs use ALU, 16 bit uses IDU.
				}
			case "ADD":
				switch c.inst.DataType {
				case E8:
					// 4 cycles, AddRel should take 2, then set SP during PC inc, Nop used to pad
					c.PushToFuncQueue(c.Fetch(LO), c.AddRelSP, c.Nop, c.SetRegister)
				case N8:
					c.PushToFuncQueue(c.Fetch(LO), c.Add)
				default:
					if c.inst.From == mHL {
						c.PushToFuncQueue(c.Read)
					} else if c.inst.To == HL {
						c.PushToFuncQueue(c.AddLo, c.AddHi)
						break
					} else {
						c.Read()
					}
					c.PushToFuncQueue(c.Add)
				}
			case "ADC":
				if c.inst.DataType == N8 {
					c.PushToFuncQueue(c.Fetch(LO))
				} else if c.inst.From == mHL {
					c.PushToFuncQueue(c.Read)
				} else {
					c.Read()
				}
				c.PushToFuncQueue(c.Adc)
			case "SUB":
				// TODO, this check seems to be kinda common, put in a func.
				if c.inst.DataType == N8 {
					c.PushToFuncQueue(c.Fetch(LO))
				} else if c.inst.From == mHL {
					c.PushToFuncQueue(c.Read)
				} else {
					c.Read()
				}
				c.PushToFuncQueue(c.Sub)
			case "SBC":
				if c.inst.DataType == N8 {
					c.PushToFuncQueue(c.Fetch(LO))
				} else if c.inst.From == mHL {
					c.PushToFuncQueue(c.Read)
				} else {
					c.Read()
				}
				c.PushToFuncQueue(c.Sbc)
			case "AND":
				if c.inst.From == mHL {
					c.PushToFuncQueue(c.Read)
				} else if c.inst.DataType == N8 {
					c.PushToFuncQueue(c.Fetch(LO))
				} else {
					c.Read()
				}
				c.PushToFuncQueue(c.AND)
			case "RRA":
				c.PushToFuncQueue(c.RRA)
			case "RRCA":
				c.PushToFuncQueue(c.RRCA)
			case "RLA":
				c.PushToFuncQueue(c.RLA)
			case "RLCA":
				c.PushToFuncQueue(c.RLCA)
			case "CPL":
				c.PushToFuncQueue(c.CPL)
			case "SCF":
				c.PushToFuncQueue(c.SCF)
			case "EI":
				c.PushToFuncQueue(c.SetIME)
			case "DI":
				c.PushToFuncQueue(c.UnsetIME)
			case "CALL":
				c.PushToFuncQueue(c.Fetch(LO), c.Fetch(HI))
				if c.flagMatches(c.inst.Flag) {
					// According to gbctr.pdf, SP decs first cycle, then push and dec 2nd, then push 3rd
					// Currently, i'm DECing then pushing in the same cycle
					c.PushToFuncQueue(c.PushToStack(HI), c.PushToStack(LO), c.SetPC)
				}
			case "RET":
				// TODO, check flag conditional
				// Unlike with CALL, pop then INCing SP in 1st and 2nd cycle is fine.
				if c.flagMatches(c.inst.Flag) {
					c.PushToFuncQueue(c.PopFromStack(LO), c.PopFromStack(HI), c.SetPC)
				}
			case "RETI":
				c.PushToFuncQueue(c.PopFromStack(LO), c.PopFromStack(HI), c.SetPC)
				c.PushToFuncQueue(c.SetIME)
			case "RST":
				c.WZ = joinBytes(0x00, c.inst.Abs)
				c.PushToFuncQueue(c.PushToStack(HI), c.PushToStack(LO), c.SetPC)
			case "PUSH":
				c.PushToFuncQueue(c.PushToStack(HI), c.PushToStack(LO))
			case "POP":
				c.PushToFuncQueue(c.PopFromStack(LO), c.PopFromStack(HI), c.SetRegister)
			case "DAA":

				c.PushToFuncQueue(c.DAA)
			case "CCF":
				c.PushToFuncQueue(c.CCF)
			case "PREFIX":
				c.PushToFuncQueue(c.Prefix)
				// c.handlePrefix()
			default:
				inst := c.inst
				log.Panicf("unimplemented op: %s/0x%02X, dt: %v, to: %s, from: %s, flag: %s", inst.Op, c.IR, inst.DataType, inst.To, inst.From, inst.Flag)
			}
		}
		// }
	} else {
		// log.Fatalf("halted, TODO: handle this")
	}
}

func (c *CPU) handlePrefix() {
	c.FetchIR()
	c.inst = lookup(c.IR, true)

	if c.inst.To == mHL {
		c.PushToFuncQueue(c.Read)
	} else {
    c.Read()
  }

	switch c.inst.Op {
	case "SWAP":
		// Swap the 2 nibbles of 8 bit register
		c.PushToFuncQueue(c.Swap)
	case "BIT":
		c.PushToFuncQueue(c.Bit)
	case "RES":
		c.PushToFuncQueue(c.Res)
	case "SET":
		c.PushToFuncQueue(c.Set)
	case "SRA":
		c.PushToFuncQueue(c.SRA)
	case "SLA":
		c.PushToFuncQueue(c.SLA)
	case "SRL":
		c.PushToFuncQueue(c.SRL)
	case "RR":
		c.PushToFuncQueue(c.RR)
	case "RRC":
		c.PushToFuncQueue(c.RRC)
	case "RL":
		c.PushToFuncQueue(c.RL)
	case "RLC":
		c.PushToFuncQueue(c.RLC)
	default:
		log.Panicf("unimplemented PREFIXED op: %s/0x%02X, dt: %v, to: %s, from: %s, flag: %s", c.inst.Op, c.IR, c.inst.DataType, c.inst.To, c.inst.From, c.inst.Flag)
	}

	if c.inst.To == mHL {
		c.PushToFuncQueue(c.Write)
	} else {
    c.PushToFuncQueue(c.SetRegister)
  }
}

func (c *CPU) PushToFuncQueue(fns ...func()) {
	c.funcQueue = append(c.funcQueue, fns...)
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
			addr = joinBytes(0xFF, c.readR8(Z))
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
func (c *CPU) Fetch(hilo register) func() {
	return func() {
		if hilo == HI {
			c.writeR8(W, c.imm8())
		} else if hilo == LO {
			c.writeR8(Z, c.imm8())
		}
	}
}

func (c *CPU) FetchIR() {
	c.IR = c.imm8()
	c.inst = lookup(c.IR, false)
}

// Write a byte to a specific memory location.
func (c *CPU) Write() {
	tt := c.regType(c.inst.To)
	if c.inst.To == m8 {
		c.writeMem(joinBytes(0xFF, c.readR8(Z)), c.readR8(A))
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
			c.writeMem(c.readR16(WZ)+1, msb(c.SP))
		} else if hilo == LO {
			c.writeMem(c.readR16(WZ), lsb(c.SP))
		}
	}
}

// Do nothing for one cycle
func (c *CPU) Nop() {
	// Empty
}

// Add c.Z to c.A and set flags. Store result in c.A.
func (c *CPU) Add() {
	c.A = c.addAndSetFlags(c.A, lsb(c.WZ))
}

// Add low byte of rr to low byte of inst.To and set flags.
func (c *CPU) AddLo() {
	aLo := lsb(c.readR16(c.inst.To))
	bLo := lsb(c.readR16(c.inst.From))
	// Replace only low byte of inst.To
	// TODO, docs don't mention Z flag. Preserve it?
	z := getBit(7, c.F)
	c.writeR16(c.inst.To, joinBytes(msb(c.readR16(c.inst.To)), c.addAndSetFlags(aLo, bLo)))
	c.setZFlag(z)
}

// Add high byte of rr to high byte of inst.To and set flags.
func (c *CPU) AddHi() {
	aHi := msb(c.readR16(c.inst.To))
	bHi := msb(c.readR16(c.inst.From))
	// TODO, same as above
	z := getBit(7, c.F)
	c.writeR16(c.inst.To, joinBytes(c.addWithCarryAndSetFlags(aHi, bHi), lsb(c.readR16(c.inst.To))))
	c.setZFlag(z)
}

// Add c.Z and carry to c.A and set flags. Store result in c.A.
func (c *CPU) Adc() {
	c.A = c.addWithCarryAndSetFlags(c.A, lsb(c.WZ))
}

// Subtract c.Z from c.A and set flags. Store result in c.A.
func (c *CPU) Sub() {
	c.A = c.subAndSetFlags(c.A, lsb(c.WZ))
}

// Subtract c.Z and carry from c.A and set flags. Store result in c.A.
func (c *CPU) Sbc() {
	c.A = c.subWithCarryAndSetFlags(c.A, lsb(c.WZ))
}

// Subtract c.Z from c.A and set flags. Do not store result in c.A.
func (c *CPU) Compare() {
	c.subAndSetFlags(c.A, lsb(c.WZ))
}

// Set the PC register to value of WZ.
func (c *CPU) SetPC() {
	c.PC = c.WZ
	// c.WZ = 0
}

// Add a signed byte to PC.
func (c *CPU) AddRelPC() {
	c.addrBusFull = true
	c.dataBusFull = true
	c.IDUBusy = true
	c.ALUBusy = true
	c.WZ, _ = addInt8ToUint16(c.readR8(Z), c.PC)
}

// Add a signed byte to SP.
func (c *CPU) AddRelSP() {
	c.addrBusFull = true
	c.dataBusFull = true
	c.IDUBusy = true
	c.ALUBusy = true
	res, flags := addInt8ToUint16(c.readR8(Z), c.SP)
	c.WZ = res
	c.clearFlags()
	c.setHalfCarry(getBit(5, flags))
	c.setCarry(getBit(4, flags))
}

// OR two bytes A and x and store result in A.
func (c *CPU) Or() {
	c.A |= c.readR8(Z)
	c.clearFlags()
	c.setZFlag(c.A)
}

// XOR two bytes A and x and store result in A.
func (c *CPU) Xor() {
	c.A ^= c.readR8(Z)
	c.clearFlags()
	c.setZFlag(c.A)
}

// AND two bytes A and x and store result in A.
func (c *CPU) AND() {
	c.A &= c.readR8(Z)
	c.clearFlags()
	c.setZFlag(c.A)
	c.setHalfCarry(1)
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
		c.miscOp = true
		c.writeR16(c.inst.To, c.readR16(WZ))
		// c.WZ = 0
	case INDIRECT:
		c.writeIndirect(c.inst.To, c.readR8(Z))
	default:
		log.Panicf("c.SetRegister unhandled regType %s", rt)
	}
}

// Increment a register or byte in memory
func (c *CPU) INC() {
	if c.inst.To == mHL {
		c.IncR(Z)
	} else {
		c.IncR(c.inst.To)
	}
}

// Decrement a register or byte in memory
func (c *CPU) DEC() {
	if c.inst.To == mHL {
		c.DecR(Z)
	} else {
		c.DecR(c.inst.To)
	}
}

// Set IME to 1
func (c *CPU) SetIME() {
	c.IME = 1
}

// Set IME to 0
func (c *CPU) UnsetIME() {
	c.IME = 0
}

// Push PC register to stack
func (c *CPU) PushToStack(hilo string) func() {
	return func() {
		var data uint16
		var b byte
		if c.inst.DataType == NODATA {
			if c.inst.Op == "RST" {
				// data = c.readR16(WZ)
				data = c.PC
			} else {
				data = c.readR16(c.inst.From)
			}
		} else {
			data = c.PC
		}

		c.SP--
		if hilo == HI {
			b = msb(data)
		} else if hilo == LO {
			b = lsb(data)
		}
		c.writeMem(c.readR16(SP), b)
	}
}

// Pop 16 bit register from stack
func (c *CPU) PopFromStack(hilo string) func() {
	return func() {
		if hilo == HI {
			c.writeR8(W, c.readMem(c.SP))
		} else if hilo == LO {
			c.writeR8(Z, c.readMem(c.SP))
		}
		c.SP++
	}
}

// Decimal Adjust Accumulator
// Adjust binary coded decimal. This is done after an instruction that adds 2 hex numbers.
// In hexadecimal, 0x16 + 0x15 = 0x2B
// In BCD, 0x16 + 0x15 = 0x31
// DAA makes this adjustment.
func (c *CPU) DAA() {
	// lo := c.A & 0xF
	// hi := c.A >> 4

	result := c.A

	if !isBitSet(6, c.F) {
		if result > 0x99 || isBitSet(4, c.F) {
			result += 0x60
			c.setCarry(1)
		}

		if result&0xF > 0x9 || isBitSet(5, c.F) {
			result += 0x6
		}
	} else {
		if isBitSet(4, c.F) {
			result -= 0x60
		}

		if isBitSet(5, c.F) {
			result -= 0x6
		}
	}

	// Check Z and C, set HC to 0
	c.setHalfCarry(0)
	// if result < c.A {
	// 	c.setCarry(1)
	// } else {
	// 	c.setCarry(0)
	// }

	c.setZFlag(result)
	c.A = result
}

// Complement Carry Flag
func (c *CPU) CCF() {
	c.setNegFlag(0)
	c.setHalfCarry(0)
	if c.getCarry() == 1 {
		c.setCarry(0)
	} else {
		c.setCarry(1)
	}
}

// Prefix() is like Cycle() but checks prefix lookup table instead
func (c *CPU) Prefix() {
	c.handlePrefix()
}

// PREFIX FUNCS

// Swap hi and lo nibbles of byte
func (c *CPU) Swap() {
	r := c.readR8(Z)
	swapped := (r << 4) | (r >> 4)
	c.writeR8(Z, swapped)
	c.clearFlags()
	c.setZFlag(swapped)

}

// Check bit of byte
func (c *CPU) Bit() {
	r := c.readR8(Z)
	b := getBit(c.inst.Bit, r)
	c.setZFlag(b)
	c.setNegFlag(0)
	c.setHalfCarry(1)

}

// Set bit of byte
func (c *CPU) Set() {
	var r byte
  r = c.readR8(Z)
  r = setBit(c.inst.Bit, r)
  c.writeR8(Z, r)
}

// Reset bit
func (c *CPU) Res() {
	var r byte
		r = c.readR8(Z)
		r = resetBit(c.inst.Bit, r)
		c.writeR8(Z, r)
}

// Shift right arithmetic
func (c *CPU) SRA() {
	data := c.readR8(Z)
	carry := getBit(0, data)
	data >>= 1
	if isBitSet(6, data) {
		data = setBit(7, data)
	}
	c.writeR8(Z, data)
	c.clearFlags()
	c.setZFlag(data)
	c.setCarry(carry)
}

// Shift left arithmetic
func (c *CPU) SLA() {
	data := c.readR8(Z)
	carry := getBit(7, data)
	data <<= 1
	c.writeR8(Z, data)
	c.clearFlags()
	c.setZFlag(data)
	c.setCarry(carry)
}

// Shift right logical, don't wrap bits
func (c *CPU) SRL() {
	data := c.readR8(Z)
	carry := getBit(0, data)
	result := data >> 1
	c.writeR8(Z, result)
	c.clearFlags()
	c.setZFlag(result)
	c.setCarry(carry)
}

// TODO, rotate circular moves popped bit to other end of byte
// Non-circular rotate uses carry bit
// Carry is set for both

// Rotate right accumulator
func (c *CPU) RRA() {
	data := c.readR8(A)
	oldCarry := c.getCarry()
	carry := getBit(0, data)
	result := c.rotateRight(data, oldCarry)

	c.writeR8(A, result)
	c.clearFlags() // TODO, docs say Z should be 0. Is this right?
	c.setCarry(carry)
}

// Rotate right circular accumulator
func (c *CPU) RRCA() {
	data := c.readR8(A)
	carry := getBit(0, data)
	result := c.rotateRight(data, carry)

	c.writeR8(A, result)
	c.clearFlags()
	c.setCarry(carry)
}

// Rotate right circular
func (c *CPU) RRC() {
	data := c.readR8(Z)
	carry := getBit(0, data)
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
	carry := getBit(0, data)
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
	carry := getBit(7, data)
	result := c.rotateLeft(data, oldCarry)

	c.writeR8(A, result)
	c.clearFlags()
	c.setCarry(carry)
}

// Rotate left circular accumulator
func (c *CPU) RLCA() {
	data := c.readR8(A)
	carry := getBit(7, data)
	result := c.rotateLeft(data, carry)

	c.writeR8(A, result)
	c.clearFlags()
	c.setCarry(carry)
}

// Rotate left circular
func (c *CPU) RLC() {
	data := c.readR8(Z)
	carry := getBit(7, data)
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
	carry := getBit(7, data)
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
}

// Set carry flag
func (c *CPU) SCF() {
	c.setNegFlag(0)
	c.setHalfCarry(0)
	c.setCarry(1)
}

// Functions used by main op funcs

// Read the memory address at [PC]
func (c *CPU) imm8() byte {
	n8 := c.readMem(c.PC)
	c.IncR(PC)
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
	c.addrBusFull = true
	c.dataBusFull = true
	return c.bus.Read(addr)
}

// Write byte to memory address [addr]
func (c *CPU) writeMem(addr uint16, data byte) {
	c.addrBusFull = true
	c.dataBusFull = true
	c.bus.Write(addr, data)
}

// Increment register
func (c *CPU) IncR(r register) {
	rt := c.regType(r)
	if rt == R8 {
		val := c.readR8(r)
		c.setHalfCarryAdd(val, 1)
		val = c.ALUInc(val)
		c.setZFlag(val)
		c.setNegFlag(0)
		c.writeR8(r, val)
	} else {
		c.writeR16(r, c.IDUInc(c.readR16(r)))
	}
}

// Decrement register
func (c *CPU) DecR(r register) {
	rt := c.regType(r)
	if rt == R8 {
		val := c.readR8(r)
		// if bit[3] carried, H = 1, else 0
		c.setHalfCarrySub(val, 1, false)
		val = c.ALUDec(val)
		c.setZFlag(val)
		c.setNegFlag(1)
		c.writeR8(r, val)
	} else {
		c.writeR16(r, c.IDUDec(c.readR16(r)))
	}
}

// OLD

// Returns true if flag f is set, else returns false.
func (c *CPU) FlagIsSet() bool {
	if c.flagMatches(c.inst.Flag) && c.inst.Flag != NOFLAG {
		return true
	}
	return false
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
		// log.Panicf("invalid register %s", reg)
		return NONE
	}
}

func (c *CPU) readR8(reg register) byte {
	switch reg {
	case A:
		return c.A
	case B:
		return msb(c.BC)
	case C:
		return lsb(c.BC)
	case D:
		return msb(c.DE)
	case E:
		return lsb(c.DE)
	case H:
		return msb(c.HL)
	case L:
		return lsb(c.HL)
	case Z:
		// ret := lsb(c.WZ)
		// c.WZ = joinBytes(msb(c.WZ), 0)
		return lsb(c.WZ)
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
		return joinBytes(c.A, c.F)
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
		return c.bus.Read(joinBytes(0xFF, lsb(c.BC)))
	case mBC:
		return c.bus.Read(c.BC)
	case mDE:
		return c.bus.Read(c.DE)
	case mHL:
		return c.bus.Read(c.HL)
	case mHLp:
		val := c.bus.Read(c.HL)
		c.HL = c.IDUInc(c.HL)
		return val
	case mHLm:
		val := c.bus.Read(c.HL)
		c.HL = c.IDUDec(c.HL)
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
		c.BC = joinBytes(value, lsb(c.BC))
	case C:
		c.BC = joinBytes(msb(c.BC), value)
	case D:
		c.DE = joinBytes(value, lsb(c.DE))
	case E:
		c.DE = joinBytes(msb(c.DE), value)
	case H:
		c.HL = joinBytes(value, lsb(c.HL))
	case L:
		c.HL = joinBytes(msb(c.HL), value)
	case W:
		c.WZ = joinBytes(value, lsb(c.WZ))
	case Z:
		c.WZ = joinBytes(msb(c.WZ), value)
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
		c.A = msb(value)
		c.F = lsb(value) & 0xF0
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
		c.bus.Write(joinBytes(0xFF, lsb(c.BC)), value)
	case mDE:
		c.bus.Write(c.DE, value)
	case mHL:
		c.bus.Write(c.HL, value)
	case mHLp:
		c.bus.Write(c.HL, value)
		c.HL = c.IDUInc(c.HL)
	case mHLm:
		c.bus.Write(c.HL, value)
		c.HL = c.IDUDec(c.HL)
	default:
		log.Panicf("op: %02X, tried to set value for unhandled indirect %s", c.IR, reg)
	}
}

func (c *CPU) flagMatches(f string) bool {
	c.miscOp = true
	switch f {
	case ZERO:
		return isBitSet(7, c.F)
	case NZ:
		return !isBitSet(7, c.F)
	case CARRY:
		return isBitSet(4, c.F)
	case NC:
		return !isBitSet(4, c.F)
	case NOFLAG:
		c.miscOp = false
		return true
	default:
		log.Panicf("unhandled flag check for flag %s", f)
		return false
	}
}

func (c *CPU) clearFlags() {
	c.F &= 0x00
}

func (c *CPU) addAndSetFlags(a, b byte) byte {
	result := a + b
	// set Z flag
	c.setZFlag(result)

	// set N to 0
	c.setNegFlag(0)

	// if bit[3] carried, H = 1, else 0
	c.setHalfCarryAdd(a, b)

	// if bit[7] carried, C = 1, else 0
	if result < a {
		c.setCarry(1)
	} else {
		c.setCarry(0)
	}

	return result
}

func (c *CPU) addWithCarryAndSetFlags(a, b byte) byte {
	result := a + b + c.getCarry()
	// set Z flag
	c.setZFlag(result)

	// set N to 0
	c.setNegFlag(0)

	// if bit[3] carried, H = 1, else 0
	halfCarry := ((a ^ b ^ c.getCarry() ^ result) & 0x10) >> 4
	c.setHalfCarry(halfCarry)

	// if bit[7] carried, C = 1, else 0
	if result < a || (result == a && c.getCarry() == 1) {
		c.setCarry(1)
	} else {
		c.setCarry(0)
	}

	return result
}

func (c *CPU) subAndSetFlags(a, b byte) byte {
	result := a - b
	c.setZFlag(result)
	c.setNegFlag(1)

	// if bit[3] carried, H = 1, else 0
	c.setHalfCarrySub(a, b, false)

	// if bit[7] carried, C = 1, else 0
	if result > a {
		c.setCarry(1)
	} else {
		c.setCarry(0)
	}

	return result
}

func (c *CPU) subWithCarryAndSetFlags(a, b byte) byte {
	result := a - b - c.getCarry()
	c.setZFlag(result)
	c.setNegFlag(1)

	// if bit[3] carried, H = 1, else 0
	c.setHalfCarrySub(a, b, true)

	// if bit[7] carried, C = 1, else 0
	if result > a || (result == a && c.getCarry() == 1) {
		c.setCarry(1)
	} else {
		c.setCarry(0)
	}

	return result
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

func (c *CPU) setNegFlag(to byte) {
	if to == 0 {
		c.F &= 0xBF
	} else {
		c.F |= 0x40
	}
}

func (c *CPU) setHalfCarryAdd(a, b byte) {
	result := a + b
	halfCarry := ((a ^ b ^ result) & 0x10) >> 4
	c.setHalfCarry(halfCarry)
}

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

func (c *CPU) setCarry(setTo byte) {
	if setTo != 0 {
		c.F |= 0x10
	} else {
		c.F &= 0xEF
	}
}

func (c *CPU) getCarry() byte {
	return getBit(4, c.F)
}
