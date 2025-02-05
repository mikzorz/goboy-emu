package main

import (
  // "log"
	utils "github.com/mikzorz/gameboy-emulator/helpers"
)

type PPU struct {
	bus                                                    *Bus
	vram                                                   [0x2000]byte
  bgFIFO *FIFO
	LCDC, STAT, SCX, SCY, LY, LYC, BGP, OBP0, OBP1, WY, WX uint8 // move to LCD?
	x                                                      int
	mode                                                   ppuMode
	dot                                                    int // current dot of scanline
  oamScanI int // index of OAM to scan
  fetchStep int
  fetcherReset bool // for reseting background fetcher at beginning of each scanline
  tileID, tileLow, tileHigh byte
}

type ppuMode string

const (
  MODE_HBLANK ppuMode = "MODE_HBLANK"
  MODE_VBLANK = "MODE_VBLANK"
  MODE_OAMSCAN = "MODE_OAMSCAN"
  MODE_DRAWING = "MODE_DRAWING"
)

func NewPPU() *PPU {
	return &PPU{
		vram: [0x2000]byte{},
		LCDC: 0x91,
		BGP:  0xFC,
		OBP0: 0xFF,
		OBP1: 0xFF,
	}
}

// I think the VRAM should be external to the PPU, and the OAM DMA should have direct access to it.
// OAM also external to DMA.
// CPU r/w VRAM via PPU.
// PPU and OAM DMA can access OAM. DMA transfers, PPU scans.
// VRAM is blocked from CPU during Mode 3 (draw), OAM is blocked during Modes 2 & 3 (scan, draw).
// All memory below OAM is blocked during OAM transfer. Does OAM DMA control that?
// Pixels FIFOs will also be separate, between PPU and LCD.
// FIFO 4MHz, Fetch 2MHz ? Push (at least) 2 pixels to LCD per fetch?
  // Should PPU run at 2MHz?

// OAM scan, store found tiles in array, stop at 10 or end of OAM

func (p *PPU) Cycle() {
	// if LCD/PPU are enabled
	if utils.IsBitSet(7, p.LCDC) {

    p.setMode()

		switch p.mode {
		case MODE_OAMSCAN:
			// OAM Scan
      // TODO: uncomment
      // y := p.oam[p.oamScanI]
      // if len(p.savedObjects) < 10 && p.objectOnScanline(y) {
      //   p.SaveObjectIndex(p.oamScanI)
      // }
      // p.oamScanI+=4
		case MODE_DRAWING:
			// Drawing to LCD
      // TODO: push pixels to FIFO, not LCD
      // For now, just assume that all tiles are background tiles.
        // Then window.
        // Then objects.
      
      p.fetcherCycle()
		case MODE_HBLANK:
			// H-Blank
		case MODE_VBLANK:
			// V-Blank
			p.bus.InterruptRequest(VBLANK_INTR)
			p.STAT &= 0xFD // Mode 1
		}

		p.dot+=2
		if p.dot >= 456 {
			p.dot = 0
			p.LY++
      p.bus.lcd.x = 0
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
	if addr >= 0x8000 && addr <= 0x9FFF && p.mode != MODE_DRAWING {
		return p.vram[addr-0x8000]
	} else if addr >= 0xFE00 && addr <= 0xFE9F && (p.mode == MODE_HBLANK || p.mode == MODE_VBLANK) {
		return p.bus.dma.oam[addr-0xFE00]
	} else if addr >= 0xFE00 && addr <= 0xFEFF && p.mode == MODE_OAMSCAN {
		// OAM Corruption Bug
		// If PPU is in mode 2, r/w to FE00-FEFF cause rubbish data (except for FE00 and FE04)
		return 0x00 // DMG
	}
	return 0xFF
}

func (p *PPU) Write(addr uint16, data byte) {

	if addr >= 0x8000 && addr <= 0x9FFF && p.mode != MODE_DRAWING {
		p.vram[addr-0x8000] = data
		if addr >= 0x9800 {
			// log.Printf("%04X %02X", addr, data)
		}
	} else if addr >= 0xFE00 && addr <= 0xFE9F && (p.mode == MODE_HBLANK || p.mode == MODE_VBLANK) {
		p.bus.dma.oam[addr-0xFE00] = data
	} else if addr >= 0xFE00 && addr <= 0xFEFF && p.mode == MODE_OAMSCAN {
		// OAM Corruption Bug
		// If PPU is in mode 2, r/w to FE00-FEFF cause rubbish data (except for FE00 and FE04)

	} else {
		// log.Fatalf("%04X %02X mode=%d", addr, data, p.mode)
		// paused=true
	}
}

func (p *PPU) setMode() {
  if p.LY >= 144 {
    p.mode = MODE_VBLANK
  } else if int32(p.bus.lcd.x) >= TRUEWIDTH {
    p.mode = MODE_HBLANK
  } else {
    if p.dot == 0 {
      p.mode = MODE_OAMSCAN
      // p.savedObjects = []objects{} // TODO uncomment
      p.fetcherReset = false
    } else if p.dot == 80 {
      p.mode = MODE_DRAWING
      p.oamScanI = 0
      // TODO: May need to sort objects from left-most x
      p.bgFIFO.Clear()
      p.fetchStep = 0
    }
  }
}

func (p *PPU) fetcherCycle() {
  switch p.fetchStep {
  case 0:
    // Fetch tile id from map
    sx, sy := byte(p.bus.lcd.x), p.LY // TODO, add scrolling
    p.tileID = p.getTileIDFromMap(sx, sy)
    p.fetchStep++
  case 1:
    // Fetch tile row low
    p.tileLow = p.fetchTileDataLow(p.tileID, p.LY) // TODO, change p.LY
    p.fetchStep++
  case 2:
    // Fetch tile row high
    p.tileHigh = p.fetchTileDataHigh(p.tileID, p.LY)
    // Push background pixels here (according to pandocs)
    // But according to GBEDG, the first time the background fetcher reaches this step per scanline, it resets.
    if !p.fetcherReset {
      p.fetchStep = 0
      p.fetcherReset = true
    } else {
      p.fetchStep++
    }
  case 3:
    // For non-bg, wait until FIFO has <= 8 pixels
    // For bg, wait until FIFO is empty
    if p.bgFIFO.CanPushBG() {
      pixelData := p.mergeTileBytes(p.tileHigh, p.tileLow)
      p.bgFIFO.Push(pixelData)
      p.fetchStep = 0
    }
  }
}

// Given x and y (0-255) coordinates, find the corresponding map tile and return its value.
func (p *PPU) getTileIDFromMap(x, y byte) byte {
  mapAddr := uint16(0x1800)
  if utils.GetBit(3, p.LCDC) == 1 {
    mapAddr += 0x400
  }
  tilex := x/8
  tiley := y/8
  // return p.bus.Read(0x9800 + uint16(x + (y/8)*32))
  return p.vram[mapAddr + uint16(tilex) + uint16(tiley)*32]
}

func (p *PPU) fetchTileDataLow(id, y byte) byte {
  // baseAddr := uint16(0x8000)
  baseAddr := uint16(0x0000)
	if utils.GetBit(4, p.LCDC) == 0 && id < 128 {
		// Only for BG/Window, not OAM
		baseAddr += 0x1000
	}
  tileAddrOffset := uint16(id) * 16 // 16 bytes per tile
  tileRowOffset := uint16(y % 8) * 2 // 2 bytes per row
  // return p.bus.Read(tileRowAddr)
  return p.vram[baseAddr + tileAddrOffset + tileRowOffset]
}

func (p *PPU) fetchTileDataHigh(id, y byte) byte {
  // baseAddr := uint16(0x8000)
  baseAddr := uint16(0x0000)
	if utils.GetBit(4, p.LCDC) == 0 && id < 128 {
		// Only for BG/Window, not OAM
		baseAddr += 0x1000
	}
  tileBaseAddr := baseAddr + uint16(id) * 16
  tileRowAddr := tileBaseAddr + uint16(y % 8) * 2
  // return p.bus.Read(tileRowAddr + 1)
  return p.vram[tileRowAddr+1]
}

func (p *PPU) mergeTileBytes(hi, lo byte) []Pixel {
  data := []Pixel{}
  for bit := 7; bit >= 0; bit-- {
    colour := (utils.GetBit(bit, hi) << 1) | utils.GetBit(bit, lo)
    data = append(data, Pixel{c: colour})
    // pal := 
  }
  return data
}
