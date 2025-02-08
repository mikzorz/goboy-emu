package main

import (
  // "log"
	"image/color"
	rl "github.com/gen2brain/raylib-go/raylib"
	utils "github.com/mikzorz/gameboy-emulator/helpers"
)

// greyscale
// var colours []color.RGBA = []color.RGBA{
// 	color.RGBA{255, 255, 255, 255},
// 	color.RGBA{150, 150, 150, 255},
// 	color.RGBA{60, 60, 60, 255},
// 	color.RGBA{0, 0, 0, 255},
// }

// green
var colours []color.RGBA = []color.RGBA{
	color.RGBA{155, 188, 15, 255},
	color.RGBA{139, 172, 15, 255},
	color.RGBA{48, 98, 48, 255},
	color.RGBA{15, 56, 15, 255},
}

type LCD struct {
	bus *Bus
  bgFIFO *FIFO
	x, y   byte
  pixelsToDiscard byte
}

func NewLCD() *LCD {
	return &LCD{}
}

		// pixels that are scrolled left of screen are not skipped, they are discarded one dot at a time.

		// For each scanline, during OAM scan, check each object in OAM from FF00-FF9F and compares y values with LY. LCDC Bit.2 for obj size. Up to 10 objects are selected. Off-screen objects count, because x-coord isn't checked.

		// When 2 opaque pixels overlap, for non-CGB, lower x-coord wins. If x-coords match, first object in OAM wins.

		// After an object pixel has been determined, only then is transparency checked.

		// when lcdc bit 0 is cleared, screen becomes blank white. window and sprites may still be displayed depending on bits 1 and 5.

func (l *LCD) Cycle() {
	if utils.IsBitSet(7, l.bus.ppu.LCDC) {
	// crudely get current tile based on x and y coord of current dot, no scrolling
  // TODO, move most/all of the computation to the PPU
  // OAM scan in PPU, FIFO in LCD
	// y := l.bus.ppu.LY
	// tx := uint16(l.x / 8)
	// ty := uint16(y / 8)
	// tileId := uint16(l.bus.ppu.vram[0x1800+ty*32+tx])
	//
	// // Get tile data
	// tileDataAddr := tileId*16 + uint16(y%8)*2
	// if utils.GetBit(4, l.bus.ppu.LCDC) == 0 && tileId < 128 {
	// 	// Only for BG/Window, not OAM
	// 	tileDataAddr += 0x1000
	// }
	//
	//  // 2 bytes for row of 8 pixels
	// tileLo := l.bus.ppu.vram[tileDataAddr]
	// tileHi := l.bus.ppu.vram[tileDataAddr+1]
	//
	//  // Get current pixel position within row
	// column := l.x % 8
	// bit := int(7 - column)
	//
	//  // Merge bit pair, get corresponding colourID from BGP
	// colourId := (utils.GetBit(bit, tileHi) << 1) | utils.GetBit(bit, tileLo)

  // log.Printf("PPU Mode: %s, len(bgFIFO): %d", l.bus.ppu.mode, len(*l.bgFIFO))
    if l.bus.ppu.mode == MODE_DRAWING && l.bgFIFO.CanPopBG() {
      pix := l.bgFIFO.Pop()
      if l.pixelsToDiscard > 0 {
        l.pixelsToDiscard--
      } else {

        paletteIdx := (pix.c * 2)
        pal := bus.Read(0xFF47) // BGP
        c := colours[(pal>>paletteIdx)&0x3]

        // Draw from top-left
        rl.DrawPixel(int32(l.x), TRUEHEIGHT-int32(l.bus.ppu.LY)-1, c)

        l.x++
      }
    }
  }
}
