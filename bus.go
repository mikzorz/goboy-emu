package main

type Bus struct {
	cart *Cart
	cpu  CPU
	ppu  PPU
	vram [0x2000]byte
	wram [0x2000]byte
}

func NewBus() Bus {
	vram := [0x2000]byte{}
	wram := [0x2000]byte{}
	return Bus{vram: vram, wram: wram}
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
		return b.vram[addr-0x8000]
	case addr <= 0xBFFF:
		// A000-BFFF, cart ram
		break
	case addr <= 0xDFFF:
		// C000-DFFF, wram
		return b.wram[addr-0xC000]
	default:
		// E000-FDFF, echo ram, mirror C000-DDFF
		// FE00-FE9F, OAM
		// FEA0-FEFF, Unused
		// FF00-FF7F, IO
		// FF80-FFFE, hram
		// FFFF-FFFF, Interrupt Enable Register (IE)
	}
	return 0
}
