package main

// import "log"
import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type LCD struct {
	bus *Bus
	x   byte
}

func NewLCD() *LCD {
	return &LCD{}
}

func (l *LCD) Pix() {
	// crudely get current tile based on x and y coord of current dot, no scrolling
	y := l.bus.ppu.LY
	tx := uint16(l.x / 8)
	ty := uint16(y / 8)
	tileId := uint16(l.bus.ppu.vram[0x1800+ty*32+tx])

	// Get tile data
	tileDataAddr := tileId*16 + uint16(y%8)*2
	if getBit(4, l.bus.ppu.LCDC) == 0 && tileId < 128 {
		// Only for BG/Window
		tileDataAddr += 0x1000
	}

	tileLo := l.bus.ppu.vram[tileDataAddr]
	tileHi := l.bus.ppu.vram[tileDataAddr+1]

	column := l.x % 8
	bit := int(7 - column)

	colourId := (getBit(bit, tileHi) << 1) | getBit(bit, tileLo)
	c := colours[(bus.Read(0xFF47)>>(colourId*2))&0x3]

	rl.DrawPixel(int32(l.x), TRUEHEIGHT-int32(y), c)
	l.x++

	if int32(l.x) >= TRUEWIDTH {
		l.x = 0
	}

}

type Pixel struct {
	c          int // 0-3
	pal        int // bit 4 OAM byte 3, 0=OBP0, 1=OBP1
	bgPriority int // bit 7 OAM byte 3, 0=obj above bg, 1=bg above obj
}

type FIFO [16]Pixel

func (f *FIFO) GetTileID() {

}

func (f *FIFO) GetTileByte(hilo string) {

}

func (f *FIFO) Push() {

}
