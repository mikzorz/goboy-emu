package main

import (
	"image/color"
)

var colours []color.RGBA = []color.RGBA{
	color.RGBA{255, 255, 255, 255},
	color.RGBA{150, 150, 150, 255},
	color.RGBA{50, 50, 50, 255},
	color.RGBA{0, 0, 0, 255},
}

type PPU struct {
	LCDC, SCX, SCY uint8
}

func NewPPU() PPU {
	return PPU{}
}

func (p PPU) Draw() {
	// drawScreen()
}

// Where does the screen read from?
//  I think bg and window are in tile maps.
// Registers SCX and SCY define the camera origin.
// WX & WY for window position. (top left corner is WX-7, WY)
// Camera wraps around tile map.
// In non-CGB mode, bg and window can be disabled with LCDC bit-0
// func drawScreen() {
//   var tData int32 = 0x8000 // tile data, 8x8 tiles, each row of tile split into byte pairs. Pair each nth bit (little endian) to get colour id.
//   var tDataSize int32 = 0x1800
//   _ = tData
//   _ = tDataSize
//   var tMap1 uint32 = 0x9800 // tile maps, 32x32 grids, containing 1-byte indexes of tiles (in tData) (so 0x8000 + index * 64?)
//   var tMap2 uint32 = 0x9C00
//   _ = tMap2
//
//   SCY := ram[0xFF42]
//   SCX := ram[0xFF43]
//   // WY := ram[0xFF4A]
//   // WX := ram[0xFF4B] - 7
//
// 	rl.BeginDrawing()
//
//   // For each column of tiles on screen
//   for ty:=0; ty<18; ty++ {
//     wrappedY := (SCY + byte(ty)) % 32
//     // For each row of tiles on screen
//     for tx:=0; tx<20; tx++ {
//       wrappedX := (SCX + byte(tx)) % 32
//       tIndex := ram[tMap1 + uint32(wrappedY * 32) + uint32(wrappedX)]
//       _ = tIndex
//
//       // For each row of pixels in tile
//       for tr:=0; tr<8; tr++ {
//         rightByte := ram[tIndex] // Are these indices just offsets that need to be added to 0x8000, or absolute addresses?
//         leftByte := ram[tIndex+1]
//         tIndex+=2
//
//         // For each pixel in row of tile
//         for p:=7; p>=0; p-- {
//           // pair bits to get palette indices
//           leftBit := (leftByte & (1 << p)) >> p
//           leftBit <<= 1
//           rightBit := (rightByte & (1 << p)) >> p
//           colour := leftBit | rightBit
//           // fmt.Println(colour)
//           // FF47 - FF49 for palette stuff. (I don't understand it currently)
//
//           px := (tx * 8) + (8 - (p + 1))
//           py := (ty * 8) + tr
//           rl.DrawPixel(int32(px), int32(py), colours[colour])
//         }
//       }
//     }
//   }
//
// 	rl.EndDrawing()
// }
