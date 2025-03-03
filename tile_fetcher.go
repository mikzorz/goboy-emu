package main

import (
  "slices"
	utils "github.com/mikzorz/gameboy-emulator/helpers"
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
	case 0:
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

		p.fetchStep++
	case 2:
		// get lo
		// TODO: Y-flip
		y := (p.bus.dma.oam[p.objectToFetch])
		p.tileLow = p.fetchTileData(p.tileID, p.objectRowOnScanline(y, p.LY, p.SCY), false, true)
		p.fetchStep++
	case 4:
		// get hi
		// TODO: Y-flip
		y := (p.bus.dma.oam[p.objectToFetch])
		p.tileHigh = p.fetchTileData(p.tileID, p.objectRowOnScanline(y, p.LY, p.SCY), true, true)
		p.fetchStep++
	case 6:
		// push to sprite fifo
		pixelData := p.mergeTileBytes(p.tileHigh, p.tileLow)
		attr := p.bus.dma.oam[p.objectToFetch+3]
		if utils.IsBitSet(5, attr) {
			// X-Flip
			slices.Reverse(pixelData)
		}

    // trim pixels that hang off the left side of the screen
		if objX := p.bus.dma.oam[p.objectToFetch+1]; objX < 8 {
			pixToTrim := 8 - objX
			pixelData = pixelData[pixToTrim:]
		}

		p.objFIFO.PushObject(pixelData)
		p.fetchStep = 0
		p.fetchingObject = false
  default:
    p.fetchStep++
	}
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
	case 0:
		// Fetch tile id from map
		p.tileID = p.getTileIDFromMap(p.x, p.LY)
		p.fetchStep++
	case 2:
		// Fetch tile row low
		p.tileLow = p.fetchTileData(p.tileID, p.LY+p.SCY, false, false)
		p.fetchStep++
	case 4:
		// Fetch tile row high
		p.tileHigh = p.fetchTileData(p.tileID, p.LY+p.SCY, true, false)
    // Reset fetcher after first fetch of each scanline, as per GBEDG
		if !p.fetcherReset {
			p.fetchStep = 0
			p.fetcherReset = true
		} else {
			p.fetchStep++
		}
	case 6:
		if p.bgFIFO.CanPushBG() {
			pixelData := p.mergeTileBytes(p.tileHigh, p.tileLow)
			p.bgFIFO.Push(pixelData)
			p.fetchStep = 0
			p.x++
		}
  default:
    p.fetchStep++
	}
}
