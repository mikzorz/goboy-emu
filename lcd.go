package main

import (
	// "log"
	rl "github.com/gen2brain/raylib-go/raylib"
	utils "github.com/mikzorz/goboy-emu/helpers"
	"image/color"
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

type LCDI interface {
	Cycle()
	SetBus(b *Bus)
	SetBgFIFO(f *FIFO)
	SetObjFIFO(f *FIFO)
	GetX() byte
	SetX(byte)
	SetPixelsToDiscard(byte)
}

type LCD struct {
	bus             *Bus
	bgFIFO          *FIFO
	objFIFO         *FIFO
	x, y            byte
	pixelsToDiscard byte
}

func NewLCD() *LCD {
	return &LCD{}
}

// lcd doesn't show image until frame after it is turned on.
// when lcdc.7 is disabled, do values get reset? will that fix the glitchy top row pixels?

func (l *LCD) Cycle() {
	if utils.IsBitSet(7, l.bus.ppu.LCDC) {

		// TODO: don't check if ppu mode == DRAWING, when final pixels are pushed to FIFO, ppu should be able to switch to HBLANK while the LCD keeps drawing
		if l.bus.ppu.mode == MODE_DRAWING && int32(l.x) < TRUEWIDTH && l.bgFIFO.CanPop() && !l.bus.ppu.fetchingObject {
			// Always pop a bg pixel, only pop obj pixel if one exists
			bgPix := l.bgFIFO.Pop()
			objPix := Pixel{}
			if l.objFIFO.CanPop() {
				objPix = l.objFIFO.Pop()
			}

			if l.pixelsToDiscard > 0 {
				l.pixelsToDiscard--
			} else {
				c := l.GetPixelColour(bgPix, objPix)

				// Draw from top-left
				if !l.bus.screenDisabled {
					rl.DrawPixel(int32(l.x), TRUEHEIGHT-int32(l.bus.ppu.LY)-1, c)
				}

				l.x++
			}
		}
	}
}

func (l *LCD) GetPixelColour(bgPix, objPix Pixel) color.RGBA {
	pix := Pixel{}
	var palAddr uint16

	bgWinEnabled := utils.IsBitSet(0, l.bus.ppu.LCDC)
	objEnabled := utils.IsBitSet(1, l.bus.ppu.LCDC)

	if !objEnabled {
		objPix.c = 0
	}

	if !bgWinEnabled {
		// if bg/window is disabled and object is either transparent or disabled, draw a white pixel
		if objPix.c == 0 {
			return colours[0]
		}
		bgPix.c = 0
	}

	if (objPix.bgPriority == 1 && bgPix.c != 0) || objPix.c == 0 {
		pix.c = bgPix.c
		palAddr = 0xFF47
	} else {
		pix.c = objPix.c
		palAddr = 0xFF48 + uint16(objPix.pal)
	}

	paletteIdx := (pix.c * 2)
	pal := bus.Read(palAddr)
	return colours[(pal>>paletteIdx)&0x3]
}

func (l *LCD) SetBus(b *Bus) {
	l.bus = b
}

func (l *LCD) SetBgFIFO(f *FIFO) {
	l.bgFIFO = f
}

func (l *LCD) SetObjFIFO(f *FIFO) {
	l.objFIFO = f
}

func (l *LCD) GetX() byte {
	return l.x
}

func (l *LCD) SetX(val byte) {
	l.x = val
}

func (l *LCD) SetPixelsToDiscard(amount byte) {
	l.pixelsToDiscard = amount
}
