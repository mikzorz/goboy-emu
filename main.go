package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TODO
// Restructure everything.
// Implement documentation without misinterpretation. Make no lazy assumptions.
// If they mention it, its probably important.
// Clock cycles, bus, drawing procedure etc.

// Is it possible to draw the game screen to a separate raylib image/texture, then apply the image each frame?
//   To prevent redrawing pixels, while allowing debug info to update?
//    This would allow me to slow down the speed of the game while having fast debug updates
//   Image is CPU/ram, Texture is GPU/OpenGL. Images must be converted to texture for rendering. Use texture directly when possible.

// Step 1. Create debugging screen layout, pause rom execution.
//  Registers
//  Palette
//  Tile Data
//  WRAM/VRAM?
//  Finish filling in opcode structs in opcodes.go
// Step 2. Hotkeys for controlling program exec
// Step 3. the rest
//  What reads opcodes, the bus or the cpu?

// TODO, remove hardcoded font or provide one with project.
var debugFontPath = "/usr/share/fonts/noto/NotoSansMono-Regular.ttf"
var debugFont rl.Font

var romPath string
var romData []byte

type Screen struct {
	w, h int32
	x, y int32
}

const TRUEWIDTH int32 = 160
const TRUEHEIGHT int32 = 144

var gameWinScale int32 = 4

var gameWindow = Screen{
	w: TRUEWIDTH * gameWinScale,
	h: TRUEHEIGHT * gameWinScale,
	x: 5,
	y: 5,
}

// Debug Info Attributes
var bytesPerRow = 16
var fontSize = 16
var debugX int32 = gameWindow.x*2 + gameWindow.w
var instructions map[uint16]string
var tilePixels []byte
var palette []color.RGBA = []color.RGBA{
	color.RGBA{255, 255, 255, 255},
	color.RGBA{150, 150, 150, 255},
	color.RGBA{60, 60, 60, 255},
	color.RGBA{0, 0, 0, 255},
}
var paused = true

var memRowExample = "00000: " + strings.Repeat("00 ", bytesPerRow) + strings.Repeat(" ", (bytesPerRow/4)-1) // -1 because real string has extraneous space. could remove but...
var memRowWidth = int32(((len(memRowExample) * fontSize) / 11) * 5)                                         // approximation

// main window
// game screen size + some space.
var window = Screen{
	w: gameWindow.w + 10 + 640, // 640 for space to the side
	h: gameWindow.h + 10,
	x: 0,
	y: 0,
}

var bus = NewBus()
var cpu = NewCPU()
var ppu = NewPPU()
var cart = NewCart()

func init() {
	flag.StringVar(&romPath, "rom", "", "The path to the rom file.")
	flag.Parse()

	// Load ROM
	if romPath == "" {
		fmt.Println("no rom provided")
		os.Exit(1)
	} else {
		// fmt.Println(romPath)
		data, err := os.ReadFile(romPath)
		if err != nil {
			log.Fatal(err)
		}

		// copy cart bytes from file to byte slice
		// for _, b := range data {
		// 	romData = append(romData, b)
		// }
		cart.LoadROMData(data)

		bus.cart = cart

		// readBank(0, 0)
		// readBank(1, 1)

		instructions = disassemble(0x0150, 0xFFFF)
		tilePixels = getTileData()
	}
}

func main() {

	rl.InitWindow(window.w, window.h, "Game Boy Emulator made in Go")
	defer rl.CloseWindow()

	debugFont = rl.LoadFont(debugFontPath)
	defer rl.UnloadFont(debugFont)
	rl.SetTargetFPS(15)

	for !rl.WindowShouldClose() {
		// EVAL GAME CODE
		// for i := 0; i < cpu.speed / 60; i++ {
		// if !paused {
		// 	for i := 0; i < cpu.speed/60000; i++ {
		// 		cpu.ReadOpCode()
		// 	}
		// }

		if rl.IsKeyPressed(rl.KeyA) {
			cpu.PC++
		}
		if rl.IsKeyPressed(rl.KeySpace) {
			paused = !paused
		}
		if !paused {
			cpu.PC++
		}
		if cpu.PC > 0xFFFF {
			cpu.PC = 0
		}
		draw()
	}
}

// draw the window with debug info and update the PPU. PPU handles the game only.
func draw() {
	rl.BeginDrawing()
	rl.ClearBackground(color.RGBA{20,20,20,255})
	drawDebugInfo()
	// ppu.Draw()
	rl.EndDrawing()
}

// Memory, registers, palettes, tiles etc.
func drawDebugInfo() {
	drawMem(0x8000, 0x97FF)
	drawInstructions()
  drawRegisters()
	drawTiles()
}

func drawMem(start, end int) {
	for row := 0; row <= (end-start)/bytesPerRow; row++ {
		out := fmt.Sprintf("%05X: ", start+row*bytesPerRow)
		for i := 0; i < bytesPerRow; i++ {
			out += fmt.Sprintf("%02X ", bus.Read(uint16(start+row*bytesPerRow+i)))
			if i%4 == 3 {
				out += " "
			}
		}
		rl.DrawTextEx(debugFont, out, rl.Vector2{float32(gameWindow.x), float32(gameWindow.y + int32(row*fontSize))}, float32(fontSize), 0, rl.LightGray)
	}
}

func drawInstructions() {
	var extra = 12 // How many lines above and below to show?

	// instructions[PC]
	s, ok := instructions[cpu.PC]
	if ok {
		rl.DrawTextEx(debugFont, s, rl.Vector2{float32(debugX), float32(5 + extra*fontSize)}, float32(fontSize), 0, rl.Magenta)
	} else {

		rl.DrawTextEx(debugFont, fmt.Sprintf("%04X", cpu.PC), rl.Vector2{float32(debugX), float32(5 + extra*fontSize)}, float32(fontSize), 0, rl.Magenta) // Temporary, to reduce stuttering during early testing
	}

	// instructions above
	var found = 0
	addr := cpu.PC
	for found < extra {
		if addr == 0 {
			break
		}
		addr--
		s, ok := instructions[addr]
		if !ok {
			continue
		}

		found++
		rl.DrawTextEx(debugFont, s, rl.Vector2{float32(debugX), float32(5 + (extra-found)*fontSize)}, float32(fontSize), 0, rl.LightGray)
	}

	// instructions below
	found = 0
	addr = cpu.PC
	for found < extra {
		if addr == 0xFFFF {
			break
		}
		addr++
		s, ok := instructions[addr]
		if !ok {
			continue
		}

		found++
		rl.DrawTextEx(debugFont, s, rl.Vector2{float32(debugX), float32(5 + (extra+found)*fontSize)}, float32(fontSize), 0, rl.LightGray)
	}
}

func drawRegisters() {
  rl.DrawTextEx(debugFont, fmt.Sprintf("A: %d", cpu.A), rl.Vector2{float32(debugX + 250), 5}, float32(fontSize), 0, rl.LightGray)
  rl.DrawTextEx(debugFont, fmt.Sprintf("BC: %d", cpu.BC), rl.Vector2{float32(debugX + 250 + int32(3 * fontSize)), 5}, float32(fontSize), 0, rl.LightGray)
  rl.DrawTextEx(debugFont, fmt.Sprintf("DE: %d", cpu.DE), rl.Vector2{float32(debugX + 250 + int32(6 * fontSize)), 5}, float32(fontSize), 0, rl.LightGray)
  rl.DrawTextEx(debugFont, fmt.Sprintf("HL: %d", cpu.HL), rl.Vector2{float32(debugX + 250 + int32(9 * fontSize)), 5}, float32(fontSize), 0, rl.LightGray)
  rl.DrawTextEx(debugFont, fmt.Sprintf("SP: %d", cpu.SP), rl.Vector2{float32(debugX + 250 + int32(12 * fontSize)), 5}, float32(fontSize), 0, rl.LightGray)
  rl.DrawTextEx(debugFont, fmt.Sprintf("IE: %d", cpu.IE), rl.Vector2{float32(debugX + 250 + int32(15 * fontSize)), 5}, float32(fontSize), 0, rl.LightGray)

  rl.DrawTextEx(debugFont, fmt.Sprintf("Z: %d", cpu.F >> 7), rl.Vector2{float32(debugX + 250), float32(5 + fontSize)}, float32(fontSize), 0, rl.LightGray)
  rl.DrawTextEx(debugFont, fmt.Sprintf("N: %d", (cpu.F >> 6) & 1), rl.Vector2{float32(debugX + 250 + int32(3 * fontSize)), float32(5 + fontSize)}, float32(fontSize), 0, rl.LightGray)
  rl.DrawTextEx(debugFont, fmt.Sprintf("HC: %d", (cpu.F >> 5) & 1), rl.Vector2{float32(debugX + 250 + int32(6 * fontSize)), float32(5 + fontSize)}, float32(fontSize), 0, rl.LightGray)
  rl.DrawTextEx(debugFont, fmt.Sprintf("C: %d", (cpu.F >> 4) & 1), rl.Vector2{float32(debugX + 250 + int32(9 * fontSize)), float32(5 + fontSize)}, float32(fontSize), 0, rl.LightGray)
}

// Copy tile data from vram to an rl.Image.
// I would like to find tile data in rom, but I don't think Gameboy games have consistent tile data storage methods.
func getTileData() []byte {
	var tileAddrStart = 0x8000
	var tileCount = 384                     // DMG
	var pixels = make([]byte, tileCount*64) // tiles * 8x8 pixels

	for tileI := 0; tileI < tileCount; tileI++ {
		tileOffset := tileI * 16
		tileAddr := uint16(tileAddrStart + tileOffset)
		for row := 0; row < 8; row++ {
			lsb := bus.Read(tileAddr)
			tileAddr++
			msb := bus.Read(tileAddr)
			tileAddr++

			for b := 7; b >= 0; b-- {
				leftBit := (msb >> b) << 1
				rightBit := lsb >> b
				pix := leftBit & rightBit
				pixels[tileI*64+row*8+(7-b)] = pix
			}
		}
	}

	// draw pixels here, create rl.Image, save that TODO
	return pixels
}

// Draw tile data image in debug area.
func drawTiles() {
	// Will probably split into 3 blocks of 16x8 tiles

	// for p := 0; p < len(tiles); p++ {
	for tile := 0; tile < 384; tile++ {
		for row := 0; row < 8; row++ {
			for col := 0; col < 8; col++ {
				pix := tilePixels[tile*64+row*8+col]
				px := int32((tile%16)*8 + col + (tile/128)*129)
				py := int32((tile/16)*8 + row - (tile/128)*64)
				rl.DrawPixel(px+debugX, window.h-64-5+py, palette[pix])
			}
		}
	}
}

// Convert bytes to instruction strings, add them to map, ready for printing
// Converted from Javidx9's nes emu tutorial code.
func disassemble(startAddr, endAddr uint16) map[uint16]string {
	var instructions = make(map[uint16]string)

	addr := startAddr
	for addr < endAddr {
		instStr := fmt.Sprintf("%04X ", addr)
		lineAddr := addr

		// Read byte from bus
		opcode := bus.Read(addr)
		addr++

		// Lookup opcode
		inst := lookup(opcode, false)
		instStr += inst.Op

		// Depending on inst type, read X amount of bytes and perform a certain action on them
		switch inst.DataType {
		case NODATA:
			if inst.To != "" {
				instStr += " " + inst.To
			}
			if inst.From != "" {
				instStr += fmt.Sprintf(", %s", inst.From)
			}
		case N8:
			n8 := bus.Read(addr)
			addr++
			instStr += fmt.Sprintf(" %s, 0x%02X", inst.To, n8)
		case N16:
			lo := bus.Read(addr)
			addr++
			hi := bus.Read(addr)
			addr++
			n16 := joinBytes(hi, lo)
			instStr += fmt.Sprintf(" %s, 0x%04X", inst.To, n16)
		case A8:
			a8 := bus.Read(addr)
			addr++
			val := bus.Read(joinBytes(0xFF, a8))

			if inst.To == m8 {
				instStr += fmt.Sprintf(" [0xFF%02X]", a8)
			} else if inst.To != "" {
				instStr += " " + inst.To
			}

			if inst.From == m8 {
				instStr += fmt.Sprintf(", [0xFF%02X] (0x%02X)", a8, val)
			} else if inst.From != "" {
				instStr += fmt.Sprintf(", %s", inst.From)
			}
		case A16:
			lo := bus.Read(addr)
			addr++
			hi := bus.Read(addr)
			addr++
			a16 := joinBytes(hi, lo)
			val := bus.Read(a16)

			if inst.To == m16 {
				instStr += fmt.Sprintf(" [0x%04X]", a16)
			} else if inst.To != "" {
				instStr += " " + inst.To + ","
			}

			if inst.From == m16 {
				instStr += fmt.Sprintf(" [0x%04X] (0x%02X)", a16, val)
			} else if inst.From != "" {
				instStr += fmt.Sprintf(" %s", inst.From)
			}
		case E8:
			n8 := bus.Read(addr)
			addr++
			e8 := int8(n8)
			instStr += fmt.Sprintf(" 0x%02X (%d)", n8, e8)
		default:
			log.Printf("Op: %s, disassembled unexpected DataType %v\n", inst.Op, inst.DataType)
			// continue
		}

		// Add string to map
		instructions[lineAddr] = instStr
	}

	return instructions
}
