package main

import (
	//	"image/color"
	//
	// "log"
	utils "github.com/mikzorz/gameboy-emulator/helpers"
)

type PPU struct {
	bus                                                    *Bus
	vram                                                   [0x2000]byte
	oam                                                    [0xA0]byte
	LCDC, STAT, SCX, SCY, LY, LYC, BGP, OBP0, OBP1, WY, WX uint8 // move to LCD?
	x                                                      int
	oamDMA                                                 bool // oam dma transfer in progress
	DMAStart                                               bool // true for first cycle of OAM DMA
	oamSource                                              byte // high byte of oam source address
	oamTransferI                                           byte // byte to fetch
	oamByte                                                byte
	mode                                                   int
	dot                                                    int // current dot of scanline
}

func NewPPU() *PPU {
	return &PPU{
		vram: [0x2000]byte{},
		oam:  [0xA0]byte{},
		LCDC: 0x91,
		BGP:  0xFC,
		OBP0: 0xFF,
		OBP1: 0xFF,
	}
}

func (p *PPU) Cycle() {
	// if LCD/PPU are enabled
	if utils.IsBitSet(7, p.LCDC) {
		if p.oamDMA {
			if p.DMAStart {
				srcAddr := utils.JoinBytes(p.oamSource, p.oamTransferI)
				p.oamByte = p.bus.Read(srcAddr)
				p.DMAStart = false
			} else {
				p.oam[p.oamTransferI] = p.oamByte
				p.oamTransferI++
				srcAddr := utils.JoinBytes(p.oamSource, p.oamTransferI)

				if p.oamTransferI >= 160 {
					p.oamDMA = false
				} else {
					p.oamByte = p.bus.Read(srcAddr)
				}
			}
		}

		if p.LY >= 144 {
			p.mode = 1
		} else {
			if p.dot < 80 {
				p.mode = 2
			} else {
				// TODO if still drawing
				// p.mode = 3
				// else
				// p.mode = 0
				// for now...
				// if p.dot < 252 {
				if p.dot < 80+int(TRUEWIDTH) {
					p.mode = 3
				} else {
					p.mode = 0
					// p.bus.lcd.x=0
				}
			}

		}

		// pixels that are scrolled left of screen are not skipped, they are discarded one dot at a time.

		// For each scanline, during OAM scan, check each object in OAM from FF00-FF9F and compares y values with LY. LCDC Bit.2 for obj size. Up to 10 objects are selected. Off-screen objects count, because x-coord isn't checked.

		// When 2 opaque pixels overlap, for non-CGB, lower x-coord wins. If x-coords match, first object in OAM wins.

		// After an object pixel has been determined, only then is transparency checked.

		// when lcdc bit 0 is cleared, screen becomes blank white. window and sprites may still be displayed depending on bits 1 and 5.

		switch p.mode {
		case 2:
			// OAM Scan
			// TODO

		case 3:
			// Drawing to LCD
			p.bus.lcd.Pix()
		case 0:
			// H-Blank
		case 1:
			// V-Blank
			p.bus.InterruptRequest(VBLANK_INTR)
			p.STAT &= 0xFD // Mode 1
		}

		p.dot++
		if p.dot >= 456 {
			p.dot = 0
			p.LY++
		}

		if p.LY == p.LYC && p.dot == 0 {
			p.STAT = utils.SetBit(2, p.STAT)
			// TODO may need to check STAT bits 6-3 to decide whether to interrupt
			p.bus.InterruptRequest(LCDI_INTR)
		}

		// for monochrome gb, LCD interrupt sometimes triggers during modes 0,1,2 or LY==LYC when writing to STAT (even $00). It behaves as if $FF is written for 1 M-cycle, then the actual written data the next M-cycle.

		if p.LY > 153 {
			p.LY = 0
		}
	} else {
		// TODO blank the screen. keep blank until next frame
	}
}

func (p *PPU) Read(addr uint16) byte {
	// TODO, if mode == oam scan, vram can be read if index 37 has been reached
	if addr >= 0x8000 && addr <= 0x9FFF && p.mode != 3 {
		return p.vram[addr-0x8000]
	} else if addr >= 0xFE00 && addr <= 0xFE9F && p.mode <= 1 {
		return p.oam[addr-0xFE00]
	} else if addr >= 0xFE00 && addr <= 0xFEFF && p.mode == 2 {
		// OAM Corruption Bug
		// If PPU is in mode 2, r/w to FE00-FEFF cause rubbish data (except for FE00 and FE04)
		return 0xFF
	}
	return 0xFF
}

func (p *PPU) Write(addr uint16, data byte) {

	if addr >= 0x8000 && addr <= 0x9FFF && p.mode != 3 {
		p.vram[addr-0x8000] = data
		if addr >= 0x9800 {
			// log.Printf("%04X %02X", addr, data)
		}
	} else if addr >= 0xFE00 && addr <= 0xFE9F && p.mode <= 1 {
		p.oam[addr-0xFE00] = data
	} else if addr >= 0xFE00 && addr <= 0xFEFF && p.mode == 2 {
		// OAM Corruption Bug
		// If PPU is in mode 2, r/w to FE00-FEFF cause rubbish data (except for FE00 and FE04)

	} else {
		// log.Fatalf("%04X %02X mode=%d", addr, data, p.mode)
		// paused=true
	}
}

func (p *PPU) StartOAMTransfer(source byte) {
	p.oamDMA = true
	p.DMAStart = true
	p.oamSource = source
	p.oamTransferI = 0
}
