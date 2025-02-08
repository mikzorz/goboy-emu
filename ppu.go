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
	x                                                      byte // tile x, internal to ppu fetcher
	mode                                                   ppuMode
	dot                                                    int // current dot of scanline
  oamScanI int // index of OAM to scan
  fetchStep int
  fetcherReset bool // for reseting background fetcher at beginning of each scanline
  tileID, tileLow, tileHigh byte
  oldConditionState byte
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

		if p.dot == 0 {
      if p.LY == p.LYC {
        p.STAT |= 0x2
      }

      p.STATInterrupt() // STATInterrupt will enable or disable LYC bit of STAT as required
    }

    p.setMode()

		switch p.mode {
		case MODE_OAMSCAN:
			// OAM Scan
      p.STAT = (p.STAT & 0xFC) | 0x02
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
      
      p.STAT = (p.STAT & 0xFC) | 0x03
      p.fetcherCycle()
		case MODE_HBLANK:
			// H-Blank
      p.STAT = (p.STAT & 0xFC)
		case MODE_VBLANK:
			// V-Blank
			// p.bus.InterruptRequest(VBLANK_INTR)
      p.STAT = (p.STAT & 0xFC) | 0x01
		}

		// for monochrome gb, LCD interrupt sometimes triggers during modes 0,1,2 or LY==LYC when writing to STAT (even $00). It behaves as if $FF is written for 1 M-cycle, then the actual written data the next M-cycle.

		p.dot+=2

		if p.dot >= 456 {
			p.dot = 0
			p.LY++
		}

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
  if p.LY < 144 {
    if p.dot == 0 {
      p.mode = MODE_OAMSCAN
      // p.savedObjects = []objects{} // TODO uncomment
      p.STAT = (p.STAT & 0xFC) | 0x02
      p.STATInterrupt()
    } else if p.dot == 80 {
      p.mode = MODE_DRAWING
      p.STAT = (p.STAT & 0xFC) | 0x03
      p.bus.lcd.pixelsToDiscard = p.SCX % 8
      // TODO: May need to sort objects from left-most x
    } else if p.x >= 32 {
      p.mode = MODE_HBLANK

      p.STAT = (p.STAT & 0xFC)
      p.oamScanI = 0
      p.bgFIFO.Clear()
      p.fetchStep = 0
      p.fetcherReset = false
      p.x = 0
      p.bus.lcd.x = 0

      p.STATInterrupt()
    }
  } else {
    if p.mode != MODE_VBLANK {
      p.mode = MODE_VBLANK
      p.STAT = (p.STAT & 0xFC) | 0x01
      p.STATInterrupt()
      p.bus.InterruptRequest(VBLANK_INTR)
    }

  }
}

// TODO: Have 2 fetchers, one for bg/window, 1 for sprites. Cycle both each ppu cycle.
func (p *PPU) fetcherCycle() {
  switch p.fetchStep {
  case 0:
    // Fetch tile id from map
    x, y := p.x, p.LY // TODO, add scrolling
    p.tileID = p.getTileIDFromMap(x, y)
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

    // Try to push every t-cycle?
    if p.bgFIFO.CanPushBG() {
      pixelData := p.mergeTileBytes(p.tileHigh, p.tileLow)
      p.bgFIFO.Push(pixelData)
      p.fetchStep = 0
      p.x++
    }
  }
}

// Given x (0-31) and y (0-255) coordinates, find the corresponding map tile and return its value.
func (p *PPU) getTileIDFromMap(x, y byte) byte {
  mapAddr := uint16(0x1800)
  if utils.GetBit(3, p.LCDC) == 1 {
    mapAddr += 0x400
  }
  tilex := (x + (p.SCX/8)) & 0x1F
  tiley := ((y/8)+(p.SCY/8))
  offset := (uint16(tiley)*32 + uint16(tilex)) & 0x3FF
  return p.vram[mapAddr + offset]
}

func (p *PPU) fetchTileDataLow(id, y byte) byte {
  baseAddr := uint16(0x0000)
	if utils.GetBit(4, p.LCDC) == 0 && id < 128 {
		// Only for BG/Window, not OAM
		baseAddr += 0x1000
	}
  tileAddrOffset := uint16(id) * 16 // 16 bytes per tile
  tileRowOffset := uint16((y+p.SCY) % 8) * 2 // 2 bytes per row
  return p.vram[baseAddr + tileAddrOffset + tileRowOffset]
}

func (p *PPU) fetchTileDataHigh(id, y byte) byte {
  baseAddr := uint16(0x0000)
	if utils.GetBit(4, p.LCDC) == 0 && id < 128 {
		// Only for BG/Window, not OAM
		baseAddr += 0x1000
	}
  tileAddrOffset := uint16(id) * 16 // 16 bytes per tile
  tileRowOffset := uint16((y+p.SCY) % 8) * 2 // 2 bytes per row
  return p.vram[baseAddr + tileAddrOffset + tileRowOffset + 1]
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

// If condition is met when no conditions were met before, trigger STAT interrupt
func (p *PPU) STATInterrupt() {
  var conditionState byte

  if p.LY == p.LYC {
    p.STAT = utils.SetBit(2, p.STAT)
    conditionState = utils.SetBit(6, conditionState)
  }

  switch p.mode {
  case MODE_OAMSCAN:
  conditionState = utils.SetBit(5, conditionState)
  case MODE_HBLANK:
  conditionState = utils.SetBit(3, conditionState)
  case MODE_VBLANK:
  conditionState = utils.SetBit(4, conditionState)
  }

  // Rising edge
  if p.oldConditionState == 0 && conditionState > 0 {
    p.bus.InterruptRequest(STAT_INTR)
  }

  p.oldConditionState = conditionState
}
