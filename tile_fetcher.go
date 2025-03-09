package main

import (
	utils "github.com/mikzorz/gameboy-emulator/helpers"
	"slices"
)

type Fetcher struct {
	p *PPU
}

type ObjFetcher struct {
	Fetcher
}

func (f *ObjFetcher) Cycle(p *PPU) {
	// If there is a sprite at the current x position, reset background fetcher and pause it
	// Pause the FIFO -> LCD pixel shifter
	if i, ok := p.objectAtCurrentX(); ok && !p.fetchingObject {
		// TODO: Potential minor optimisation
		// Is it more optimal to check fetchingObject first, before calling objectAtCurrentX() ?
		// Or does it not matter? Test, for curiosity's sake.
		p.fetchStep = 0
		p.fetchingObject = true
		p.objectToFetch = i
	}

	if p.fetchingObject {
		f.Step(p)
	}
}

func (f *ObjFetcher) Step(p *PPU) {
	switch p.fetchStep {
	case 1:
		p.tileID = p.bus.dma.oam[p.objectToFetch+2]

		// If LCDC.2 is set, sprites size = 8x16, else 8x8
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

	case 3:
		// get lo
		// TODO: Y-flip
		y := (p.bus.dma.oam[p.objectToFetch])
		p.tileLow = p.fetchTileData(p.tileID, p.objectRowOnScanline(y, p.LY, p.SCY), false, true)
	case 5:
		// get hi
		// TODO: Y-flip
		y := (p.bus.dma.oam[p.objectToFetch])
		p.tileHigh = p.fetchTileData(p.tileID, p.objectRowOnScanline(y, p.LY, p.SCY), true, true)
	case 7:
		// push to sprite fifo
		pixelData := p.mergeTileBytes(p.tileHigh, p.tileLow)
		objFlags := p.bus.dma.oam[p.objectToFetch+3]
		if utils.IsBitSet(5, objFlags) {
			// X-Flip
			slices.Reverse(pixelData)
		}

		// trim pixels that hang off the left side of the screen
		if objX := p.bus.dma.oam[p.objectToFetch+1]; objX < 8 {
			pixToTrim := 8 - objX
			pixelData = pixelData[pixToTrim:]
		}

		bgPriority := utils.GetBit(7, objFlags)
		pal := utils.GetBit(4, objFlags)
		for i, _ := range pixelData {
			pixelData[i].pal = pal
			pixelData[i].bgPriority = bgPriority
		}

		p.objFIFO.PushObject(pixelData)

		p.fetchingObject = false
		p.fetchStep = 0
		return
	}
	p.fetchStep++
}

type BGFetcher struct {
	Fetcher
}

func (f *BGFetcher) Cycle(p *PPU) {
	if !p.fetchingObject {
		f.Step(p)
	}
}

func (f *BGFetcher) Step(p *PPU) {
	switch p.fetchStep {
	case 1:
		// Fetch tile id from map
		if p.fetchingWindow {
			p.tileID = p.getWindowIDFromMap(p.x, p.windowLineCounter)
		} else {
			p.tileID = p.getTileIDFromMap(p.x, p.LY)
		}
	case 3:
		// Fetch tile row low
		if p.fetchingWindow {
			p.tileLow = p.fetchTileData(p.tileID, p.windowLineCounter, false, false)
		} else {
			p.tileLow = p.fetchTileData(p.tileID, p.LY+p.SCY, false, false)
		}
	case 5:
		// Fetch tile row high
		if p.fetchingWindow {
			p.tileHigh = p.fetchTileData(p.tileID, p.windowLineCounter, true, false)
		} else {
			p.tileHigh = p.fetchTileData(p.tileID, p.LY+p.SCY, true, false)
		}
		// Reset fetcher after first fetch of each scanline, as per GBEDG
		if !p.fetcherReset {
			p.x = 0
			p.fetchStep = 0
			p.fetcherReset = true
			return
		}
	case 7:
		if p.bgFIFO.CanPushBG() {
			pixelData := p.mergeTileBytes(p.tileHigh, p.tileLow)
			p.bgFIFO.Push(pixelData)
			p.x += 8
			p.fetchStep = 0
		}
		return
	}

	p.fetchStep++
}
