package main

import (
	// "log"
	utils "github.com/mikzorz/gameboy-emulator/helpers"
	"slices"
	"sort"
)

type PPU struct {
	bus                                                    *Bus
	vram                                                   [0x2000]byte
	bgFIFO                                                 *FIFO
	objFIFO                                                *FIFO
	LCDC, STAT, SCX, SCY, LY, LYC, BGP, OBP0, OBP1, WY, WX uint8 // move to LCD?
	x                                                      byte  // tile x, internal to ppu fetcher
	mode                                                   ppuMode
	dot                                                    int    // current dot of scanline
	oamScanI                                               byte   // index of OAM to scan
	savedObjects                                           []byte // object' oam indices, for objects on current scanline
	fetchingObject                                         bool   // currently fetching object pixel data
	objectToFetch                                          byte   // oam index of object to be fetched
	fetchStep                                              int
	fetcherReset                                           bool // for reseting background fetcher at beginning of each scanline
	tileID, tileLow, tileHigh                              byte
	oldConditionState                                      byte
}

type ppuMode string

const (
	MODE_HBLANK  ppuMode = "MODE_HBLANK"
	MODE_VBLANK          = "MODE_VBLANK"
	MODE_OAMSCAN         = "MODE_OAMSCAN"
	MODE_DRAWING         = "MODE_DRAWING"
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

func (p *PPU) Cycle() {
	// if LCD/PPU are enabled
	if utils.IsBitSet(7, p.LCDC) {

		if p.dot == 0 {
			if p.LY == p.LYC {
				p.STAT |= 0b100
			} else {
				p.STAT &= 0xFB
			}

			p.oldConditionState = 0
			p.STATInterrupt() // STATInterrupt will enable or disable LYC bit of STAT as required
		}

		p.setMode()

		switch p.mode {
		case MODE_OAMSCAN:
			// OAM Scan
			if len(p.savedObjects) < 10 && p.objectOnScanline(p.oamScanI, p.LY) {
				p.saveObjectIndex(p.oamScanI)
			}
			p.oamScanI += 4
		case MODE_DRAWING:
			// Drawing to LCD
			// TODO: push pixels to FIFO, not LCD
			// For now, assumes that all tiles are background tiles.
			// TODO
			// Window.
			// Then objects.

			// If there is a sprite at the current x position, reset background fetcher and pause it
			// Pause the FIFO -> LCD pixel shifter
			if i, ok := p.objectAtCurrentX(); ok && !p.fetchingObject {
				p.fetchStep = 0
				p.fetchingObject = true
				p.objectToFetch = i
			}

			if p.fetchingObject {
				p.objectFetcherCycle()
			} else {
				p.bgFetcherCycle()
			}
		case MODE_HBLANK:
			// H-Blank
		case MODE_VBLANK:
			// V-Blank
		}

		// for monochrome gb, LCD interrupt sometimes triggers during modes 0,1,2 or LY==LYC when writing to STAT (even $00). It behaves as if $FF is written for 1 M-cycle, then the actual written data the next M-cycle.

		p.dot += 2

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
		return p.bus.dma.Read(addr)
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
			p.STAT = (p.STAT & 0xFC) | 0x02

			p.savedObjects = []byte{}
			p.oamScanI = 0
			p.STATInterrupt()
		} else if p.dot == 80 {
			p.mode = MODE_DRAWING
			p.STAT = (p.STAT & 0xFC) | 0x03
			p.bus.lcd.SetPixelsToDiscard(p.SCX % 8)
			// } else if p.x >= 32 && p.mode != MODE_HBLANK {
			// } else if p.x >= 21 && p.mode != MODE_HBLANK {
		} else if int32(p.bus.lcd.GetX()) >= TRUEWIDTH && p.mode != MODE_HBLANK { // breaks graphics, for some reason
			p.mode = MODE_HBLANK
			p.STAT = (p.STAT & 0xFC)

			p.bgFIFO.Clear()
			p.objFIFO.Clear()
			p.fetchStep = 0
			p.fetcherReset = false
			p.x = 0
			p.bus.lcd.SetX(0)

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

func (p *PPU) objectOnScanline(oamIndex, scanline byte) bool {
	y := p.bus.dma.oam[oamIndex]

	spriteSize := byte(8)
	if utils.GetBit(2, p.LCDC) == 1 {
		spriteSize = 16
	}

	objLine := p.objectRowOnScanline(y, scanline, p.SCY)
	if objLine >= 0 && objLine < spriteSize {
		return true
	}

	return false
}

func (p *PPU) objectRowOnScanline(objY, scanline, scroll byte) byte {
	return 16 + scanline + scroll - objY
}

func (p *PPU) saveObjectIndex(idx byte) {
	p.savedObjects = append(p.savedObjects, idx)
	sort.Slice(p.savedObjects, func(i, j int) bool {
		return p.bus.dma.oam[p.savedObjects[i]+1] < p.bus.dma.oam[p.savedObjects[j]+1]
	})
}

// Check first object of ppu.savedObjects, if object's X is within current tile, return the oam index of the object and TRUE, else return 0 and FALSE
func (p *PPU) objectAtCurrentX() (index byte, objectFound bool) {
	if len(p.savedObjects) > 0 {
		index := p.savedObjects[0]
		objX := p.bus.dma.oam[index+1]
		if objX >= (p.x)*8 && objX < (p.x+1)*8 {
			p.savedObjects = p.savedObjects[1:]
			return index, true
		}
	}

	return 0, false

}

func (p *PPU) objectFetcherCycle() {
	switch p.fetchStep {
	case 0:
		p.tileID = p.bus.dma.oam[p.objectToFetch+2]

		// Check if LCDC bit 2 is set AND check if currently getting top or bottom tile
		if utils.IsBitSet(2, p.LCDC) {
			y := (p.bus.dma.oam[p.objectToFetch])
			row := p.objectRowOnScanline(y, p.LY, p.SCY)
			if row < 8 {
				// Top tile
				p.tileID &= 0xFE
			} else {
				// Bottom tile
				p.tileID |= 0x01
			}
		}

		p.fetchStep++
	case 1:
		// get lo
		// TODO: Y-flip
		y := (p.bus.dma.oam[p.objectToFetch])
		p.tileLow = p.fetchTileData(p.tileID, p.objectRowOnScanline(y, p.LY, p.SCY), false, true)
		p.fetchStep++
	case 2:
		// get hi
		// TODO: Y-flip
		y := (p.bus.dma.oam[p.objectToFetch])
		p.tileHigh = p.fetchTileData(p.tileID, p.objectRowOnScanline(y, p.LY, p.SCY), true, true)
		p.fetchStep++
	case 3:
		// push to sprite fifo
		pixelData := p.mergeTileBytes(p.tileHigh, p.tileLow)
		attr := p.bus.dma.oam[p.objectToFetch+3]
		if utils.IsBitSet(5, attr) {
			// X-Flip
			slices.Reverse(pixelData)
		}

		if x := p.bus.dma.oam[p.objectToFetch+1]; x < 8 {
			pixToTrim := 8 - x
			pixelData = pixelData[pixToTrim:]
		}
		pixelData = pixelData[len(*p.objFIFO):] // Pixels already in FIFO take priority.

		p.objFIFO.Push(pixelData)
		p.fetchStep = 0
		p.fetchingObject = false
	}
}

func (p *PPU) bgFetcherCycle() {
	switch p.fetchStep {
	case 0:
		// Fetch tile id from map
		p.tileID = p.getTileIDFromMap(p.x, p.LY)
		p.fetchStep++
	case 1:
		// Fetch tile row low
		p.tileLow = p.fetchTileData(p.tileID, p.LY+p.SCY, false, false)
		p.fetchStep++
	case 2:
		// Fetch tile row high
		p.tileHigh = p.fetchTileData(p.tileID, p.LY+p.SCY, true, false)
		// Push background pixels here (according to pandocs)
		// But according to GBEDG, the first time the background fetcher reaches this step per scanline, it resets.
		if !p.fetcherReset {
			p.fetchStep = 0
			p.fetcherReset = true
		} else {
			p.fetchStep++
		}
	case 3:
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

	// if utils.IsBitSet(5, p.LCDC) {
	//   // window tiles
	//   if utils.GetBit(6, p.LCDC) == 1 {
	//     mapAddr += 0x400
	//   }
	// } else {
	// bg tile
	if utils.GetBit(3, p.LCDC) == 1 {
		mapAddr += 0x400
	}
	tilex, tiley := p.getTileCoords(x*8, y, p.SCX, p.SCY)
	// }
	offset := (uint16(tiley)*32 + uint16(tilex)) & 0x3FF
	return p.vram[mapAddr+offset]
}

// Given pixel coordinates x and y, and pixel offsets scx and scy, return the tile's x and y coordinates.
func (p *PPU) getTileCoords(x, y, scx, scy byte) (tx, ty byte) {
	tx = ((x + scx) / 8)
	ty = ((y + scy) / 8)
	return
}

func (p *PPU) fetchTileData(id, y byte, hi, objectTile bool) byte {
	baseAddr := uint16(0x0000)
	if utils.GetBit(4, p.LCDC) == 0 && id < 128 && !objectTile {
		// Only for BG/Window, not OAM
		baseAddr += 0x1000
	}
	tileAddrOffset := uint16(id) * 16  // 16 bytes per tile
	tileRowOffset := uint16((y)%8) * 2 // 2 bytes per row

	tileDataAddr := baseAddr + tileAddrOffset + tileRowOffset
	if hi {
		tileDataAddr++
	}
	return p.vram[tileDataAddr]
}

func (p *PPU) mergeTileBytes(hi, lo byte) []Pixel {
	data := []Pixel{}
	for bit := 7; bit >= 0; bit-- {
		colour := (utils.GetBit(bit, hi) << 1) | utils.GetBit(bit, lo)
		data = append(data, Pixel{c: colour})
	}
	return data
}

// If condition is met when no conditions were met before, trigger STAT interrupt
func (p *PPU) STATInterrupt() {
	var conditionState byte

	if p.LY == p.LYC {
		if utils.IsBitSet(6, p.STAT) {
			conditionState |= (1 << 6)
		}
	}

	switch p.mode {
	case MODE_OAMSCAN:
		if utils.IsBitSet(5, p.STAT) {
			conditionState |= (1 << 5)
		}
	case MODE_HBLANK:
		if utils.IsBitSet(3, p.STAT) {
			conditionState |= (1 << 3)
		}
	case MODE_VBLANK:
		if utils.IsBitSet(4, p.STAT) {
			conditionState |= (1 << 4)
		}
	}

	// Check for rising edge
	if p.oldConditionState == 0 && conditionState > 0 {
		p.bus.InterruptRequest(STAT_INTR)
	}

	p.oldConditionState = conditionState
}
