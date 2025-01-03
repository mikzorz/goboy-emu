package main

import "log"

type Bus struct {
	cart    *Cart
	cpu     *CPU
	ppu     *PPU
	clock   *Clock
	lcd     *LCD
	wram    [0x2000]byte
	hram    [0x7F]byte
	SB      byte // Serial Transfer Data
	SC      byte // Serial Transfer Control
	DIV     byte
	oldTIMA byte
	TIMA    byte
	TMA     byte // Timer Modulo
	TAC     byte
	// Sound
	NR32 byte
	NR50 byte
	NR51 byte
	NR52 byte // sound on/off
	// Joypad
	JOYP   byte
	halted bool
}

func NewBus(cart *Cart) *Bus {
	wram := [0x2000]byte{}
	hram := [0x7F]byte{}
	b := &Bus{
		cart:  cart,
		cpu:   NewCPU(),
		ppu:   NewPPU(),
		clock: NewClock(),
		lcd:   NewLCD(),
		wram:  wram,
		hram:  hram,
	}
	cart.bus = b
	b.cpu.bus = b
	b.ppu.bus = b
	b.clock.bus = b
	b.lcd.bus = b
	return b
}

func (b *Bus) Cycle() {
	if b.clock.GetTicks()%4 == 0 {
		// timers
		oldSysCounter := b.clock.sysCounter
		b.clock.sysCounter++
		b.DIV = byte(b.clock.sysCounter >> 6) // 16384 hz
		if isBitSet(2, b.TAC) {
			if b.TIMA < b.oldTIMA {
				b.TIMA = b.TMA
				b.InterruptRequest(TIMER)
			}
			b.oldTIMA = b.TIMA
			sysCounterDiff := oldSysCounter ^ b.clock.sysCounter
			switch b.TAC & 0x3 {
			case 0:
				b.TIMA += byte((sysCounterDiff >> 8) & 0x1)
			case 1:
				b.TIMA += byte((sysCounterDiff >> 2) & 0x1)
			case 2:
				b.TIMA += byte((sysCounterDiff >> 4) & 0x1)
			case 3:
				b.TIMA += byte((sysCounterDiff >> 6) & 0x1)
			}
		}

		b.cpu.Cycle()
	}
	b.ppu.Cycle()
	b.clock.Tick()

}

func (b Bus) Read(addr uint16) byte {
	switch {
	case addr <= 0x3FFF:
		// 0000-3FFF, cart bank 00
		return b.cart.Read(addr)
	case addr <= 0x7FFF:
		// 4000-7FFF, cart bank 01-NN
		return b.cart.Read(addr)
	case addr <= 0x9FFF:
		// 8000-9FFF, vram
		return b.ppu.Read(addr)
	case addr <= 0xBFFF:
		// A000-BFFF, cart ram
		return b.cart.Read(addr)
	case addr <= 0xDFFF:
		// C000-DFFF, wram
		return b.wram[addr-0xC000]
	case addr <= 0xFDFF:
		// E000-FDFF, echo ram, mirror C000-DDFF
		return b.wram[addr-0xE000]
	case addr <= 0xFE9F:
		// FE00-FE9F, OAM
		return b.ppu.Read(addr)
	case addr <= 0xFEFF:
		// FEA0-FEFF, Unused
		return b.ppu.Read(addr)
	case addr <= 0xFF7F:
		// FF00-FF7F, IO
		return b.IO(addr)
	case addr <= 0xFFFE:
		// FF80-FFFE, hram
		return b.hram[addr-0xFF80]
	case addr <= 0xFFFF:
		// FFFF-FFFF, Interrupt Enable Register (IE)
		return b.cpu.IE
	default:
		log.Panicf("unimplemented mem access 0x%04X", addr)
	}
	return 0
}

func (b *Bus) IO(addr uint16) byte {
	switch addr {
	case 0xFF00:
		// Joypad Buttons
		return b.JOYP
	case 0xFF01:
		return b.SB
	case 0xFF02:
		return b.SC
	case 0xFF04:
		return b.DIV
	case 0xFF05:
		return b.TIMA
	case 0xFF06:
		return b.TMA
	case 0xFF07:
		return b.TAC
	case 0xFF0F:
		return b.cpu.IF
	//  // case 0xFF1C:
	//  //   return b.NR32
	case 0xFF10, 0xFF11, 0xFF12, 0xFF13, 0xFF14:
		// Channel 1 audio
		return 0
	case 0xFF16, 0xFF17, 0xFF18, 0xFF19:
		// Channel 2 audio
		return 0
	case 0xFF1A, 0xFF1B, 0xFF1C, 0xFF1D, 0xFF1E:
		// Channel 3 audio
		return 0
	case 0xFF20, 0xFF21, 0xFF22, 0xFF23:
		// Channel 4 audio
		return 0
	case 0xFF24:
		return b.NR50
	case 0xFF25:
		return b.NR51
	case 0xFF26:
		return b.NR52
	case 0xFF40:
		return b.ppu.LCDC
	case 0xFF41:
		return b.ppu.STAT
	case 0xFF42:
		return b.ppu.SCY
	case 0xFF43:
		return b.ppu.SCX
	case 0xFF44:
		return b.ppu.LY
	case 0xFF45:
		return b.ppu.LYC
	case 0xFF47:
		return b.ppu.BGP
	case 0xFF48:
		return b.ppu.OBP0
	case 0xFF49:
		return b.ppu.OBP1
	case 0xFF4A:
		return b.ppu.WY
	case 0xFF4B:
		return b.ppu.WX // TODO sub 7?
	case 0xFF03, 0xFF08, 0xFF09, 0xFF0A, 0xFF0B, 0xFF0C, 0xFF0D, 0xFF0E, 0xFF15, 0xFF1F, 0xFF46, 0xFF4C:
		// undocumented
		return 0
	default:
		if addr >= 0xFF27 && addr <= 0xFF3F {
			return 0
		}
		if addr >= 0xFF4D {
			// some CGB-only registers
			return 0
		}
		log.Panicf("tried to read unimplemented IO 0x%04X", addr)
		return 0
	}
}

func (b *Bus) Write(addr uint16, data byte) {
	switch {
	case (addr >= 0x2000 && addr <= 0x3FFF):
		// switch rom bank 01-1F
		bank := data & 0x1F // lower 5 bits
		switch bank {
		case 0x00, 0x20, 0x40, 0x60:
			bank++
		}
		b.cart.SwitchBank(bank)
	case (addr >= 0x8000 && addr <= 0x9FFF):
		b.ppu.Write(addr, data)
	case (addr >= 0xA000 && addr <= 0xBFFF):
		b.cart.Write(addr, data)
	case (addr >= 0xC000 && addr <= 0xDFFF):
		b.wram[addr-0xC000] = data
	case (addr >= 0xE000 && addr <= 0xFDFF):
		// Echo
		b.wram[addr-0xE000] = data
	case addr >= 0xFE00 && addr <= 0xFE9F:
		// OAM
		b.ppu.Write(addr, data)
	case addr >= 0xFEA0 && addr <= 0xFEFF:
		// Not Usable
		b.ppu.Write(addr, data)
	case addr >= 0xFF80 && addr <= 0xFFFE:
		// FF80-FFFE, hram
		b.hram[addr-0xFF80] = data
	default:
		switch addr {
		case 0xFF00:
			keySelect := data & 0x30
			b.JOYP = (b.JOYP & 0xCF) | keySelect
		case 0xFF01:
			b.SB = data
		case 0xFF02:
			b.SC = data
			if DEV && b.SC == 0x81 {
				log.Println(string(rune(b.SB)))
				b.SC = 0x0
			}
		case 0xFF04:
			b.DIV = 0x00
		case 0xFF05:
			b.TIMA = data
		case 0xFF06:
			b.TMA = data
		case 0xFF07:
			b.TAC = data
		case 0xFF0F:
			b.cpu.IF = data
		case 0xFF10, 0xFF11, 0xFF12, 0xFF13, 0xFF14:
			// Channel 1 audio
			break
		case 0xFF16, 0xFF17, 0xFF18, 0xFF19:
			// Channel 2 audio
			break
		case 0xFF1A, 0xFF1B, 0xFF1C, 0xFF1D, 0xFF1E:
			// Channel 3 audio
			break
		case 0xFF20, 0xFF21, 0xFF22, 0xFF23:
			// Channel 4 audio
			break
		// case 0xFF16, 0xFF17, 0xFF18:
		// 	// TODO Channel 2 sound length/wave pattern duty
		// 	break
		// case 0xFF19:
		// 	// TODO Channel 2 freq hi data
		// 	break
		case 0xFF24:
			b.NR50 = data
		case 0xFF25:
			b.NR51 = data
		case 0xFF26:
			if isBitSet(7, data) {
				b.NR52 = 0
			} else {
				// Bits 0-3 are read only.
				b.NR52 = (data & 0xF0) | (b.NR52 & 0xF)
			}
		case 0xFF40:
			b.ppu.LCDC = data
		case 0xFF41:
			b.ppu.STAT = data
		case 0xFF42:
			b.ppu.SCY = data
		case 0xFF43:
			b.ppu.SCX = data
		case 0xFF44:
			b.ppu.LY = 0
		case 0xFF45:
			b.ppu.LYC = data
		case 0xFF46:
			b.ppu.oamDMA = true
			b.ppu.oamSource = data
			b.ppu.oamTransferI = 0
		case 0xFF47:
			b.ppu.BGP = data
		case 0xFF48:
			b.ppu.OBP0 = data
		case 0xFF49:
			b.ppu.OBP1 = data
			// case addr <= 0xFF7F:
		case 0xFF4A:
			b.ppu.WY = data
		case 0xFF4B:
			b.ppu.WX = data
		case 0xFFFF:
			b.cpu.IE = data
		case 0xFF03, 0xFF08, 0xFF09, 0xFF0A, 0xFF0B, 0xFF0C, 0xFF0D, 0xFF0E, 0xFF15, 0xFF1F, 0xFF4C:
			// undocumented
			break
		default:
			if addr >= 0xFF27 && addr <= 0xFF3F {
				break
			}
			if addr >= 0xFF4D {
				// some CGB-only registers
				break
			}
			log.Panicf("tried to write 0x%02X to unimplemented address 0x%04X", data, addr)
		}
	}
}

type interrupt int

const (
	VBLANK interrupt = iota
	LCDI
	TIMER
	SERIAL
	JOYPAD
)

// I think interrupts are supposed to be direct to CPU. For now, pass through bus.
func (b *Bus) InterruptRequest(intr interrupt) {
	b.cpu.IF = setBit(int(intr), b.cpu.IF)
}
