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

const DEV = false
// const DEV = true

// Big TODO List

// When running Tetris, crash during interrupt.
// I think it's continuing the func queue, but the old instruction has been replaced
// I think the old instr needs to be put back afterwards

// Next tests

// The GB might not have a bus...
//  It does, but it doesn't have a clock or anything. It's just wires. No control.
//  So, bus.cycle doesn't make much sense for a Gameboy
// VRAM & OAM RAM belong to PPU, CPU goes via PPU, PPU can block CPU access to VRAM
// 20 cycles of OAM search, 43 cycles of pixel transfer(drawing), 51 cycles of H-Blank
// Pixel FIFO
//  16 pixel buffer
//  If contains more than 8 pixels, outputs 1 pixel per cycle.
//  Pushes 2 pixels, fetches data. Repeats until new tile row is ready and space in FIFO.
//    Fetch = get tile id, byte 1, byte 2
// Finish setting default values
// Rearrange components to match hardware more closely


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

var gameWinScale int32 = 1

var gameWindow = Screen{
	w: TRUEWIDTH * gameWinScale,
	h: TRUEHEIGHT * gameWinScale,
	x: 0,
	y: 0,
}

// var gameImg *rl.Image
var gameScreen rl.RenderTexture2D

// Debug Info Attributes
var bytesPerRow = 16
var fontSize = 16
var instructions map[uint16]string
var tilePixels []byte
var colours []color.RGBA = []color.RGBA{
	color.RGBA{255, 255, 255, 255},
	color.RGBA{150, 150, 150, 255},
	color.RGBA{60, 60, 60, 255},
	color.RGBA{0, 0, 0, 255},
}
var shouldDrawGame bool = true // switch between displaying game screen or memory
var memRowExample = "00000: " + strings.Repeat("00 ", bytesPerRow) + strings.Repeat(" ", (bytesPerRow/4)-1) // -1 because real string has extraneous space. could remove but...
var memRowWidth = int32(((len(memRowExample) * fontSize) / 11) * 5)                                         // approximation
var memSelection = 0
var debugX int32 = memRowWidth + 5

// Debug control
var paused = true
var cyclesPerFrame = 1
var breakpoints = map[uint16]bool{
	// 0xDEFA: true,
}

// int = how many occurrences to skip before pausing.
var opbreaks = map[byte]int{
	// 0x40: 0, // LD B, B (used by tests, but also unintentionally matches CB 40)
	// 0xFB: 0, // EI
  // 0x06: 1703,
}

var disAssembleStart, disAssembleEnd uint16 = 0x0000, 0xFFFF

const instructionsPeekAmount = 12 // How many lines above and below current instruction to show?

// main window
// game screen size + some space.
var window = Screen{
	w: gameWindow.w * 4 + 10 + 640, // 640 for space to the side
	h: gameWindow.h * 4 + 10,
	x: 0,
	y: 0,
}

// var cpu = NewCPU()
var ppu = NewPPU()
var cart = NewCart()
var bus = NewBus(cart)

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
			log.Panic(err)
		}

		// copy cart bytes from file to byte slice
		// for _, b := range data {
		// 	romData = append(romData, b)
		// }
		cart.LoadROMData(data)

		// readBank(0, 0)
		// readBank(1, 1)

		populatePrefixLookup()
    if DEV {
  		instructions = disassemble(disAssembleStart, disAssembleEnd)
    } else {
      gameWindow.x, gameWindow.y = 0, 0
      window.w, window.h = gameWindow.w, gameWindow.h
    }
	}
}

func main() {

	rl.InitWindow(window.w, window.h, "Game Boy Emulator made in Go")
	defer rl.CloseWindow()

	debugFont = rl.LoadFont(debugFontPath)
	defer rl.UnloadFont(debugFont)

	// gameImg = rl.NewImage([]byte{}, TRUEWIDTH, TRUEHEIGHT, 1, rl.UncompressedR8g8b8)
	// gameImg = rl.GenImageColor(int(TRUEWIDTH), int(TRUEHEIGHT), rl.White)
	// defer rl.UnloadImage(gameImg)
	gameScreen = rl.LoadRenderTexture(TRUEWIDTH, TRUEHEIGHT)
	defer rl.UnloadRenderTexture(gameScreen)

	rl.SetTargetFPS(60)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Crashed at address 0x%04X\n", bus.cpu.PC)
			log.Printf("OP: 0x%02X %s\n", bus.cpu.IR, bus.cpu.inst.Op)
		}
	}()

	for !rl.WindowShouldClose() {
		rl.BeginTextureMode(gameScreen)

    if DEV {
		if rl.IsKeyPressed(rl.KeyM) {
        shouldDrawGame = !shouldDrawGame
    }

		if rl.IsKeyPressed(rl.KeyDown) {
			if rl.IsKeyDown(rl.KeyLeftShift) {
				memSelection += 7
			}
			memSelection++ // Change debug mem selection
		}
		if rl.IsKeyPressed(rl.KeyUp) {
			if rl.IsKeyDown(rl.KeyLeftShift) {
				memSelection -= 7
			}
			memSelection--
			if memSelection < 0 {
				memSelection = 0
			}
		}

		// 1 M-Cycle
		if rl.IsKeyPressed(rl.KeyA) {
			mcycle()
		}

		// 1 Op (varying amount of cycles)
		if rl.IsKeyPressed(rl.KeyS) {
			prevOp := bus.cpu.IR
			for prevOp == bus.cpu.IR {
				mcycle()
			}
		}

		if rl.IsKeyPressed(rl.KeySpace) {
			paused = !paused
		}

		if rl.IsKeyPressed(rl.KeyRight) {
			cyclesPerFrame += 100000
		}
		if rl.IsKeyPressed(rl.KeyLeft) {
			cyclesPerFrame -= 100000
		}

		if !paused {
			for i := 0; i < cyclesPerFrame; i++ {
				if _, ok := breakpoints[bus.cpu.PC]; ok {
					paused = true
				} else if occ, ok := opbreaks[bus.cpu.IR]; ok {
					if occ > 0 {
						opbreaks[bus.cpu.IR]--
					} else {
						paused = true
					}
				}
				if !paused {
					bus.Cycle()
				}
			}
		}

		disAssembleStart = bus.cpu.PC - instructionsPeekAmount - 10 // 0x10 margin
		disAssembleEnd = bus.cpu.PC + instructionsPeekAmount + 10
		instructions = disassemble(disAssembleStart, disAssembleEnd)
    } else {
      for i := 0; i < 70224; i++ {
        bus.Cycle()
      }
    }

		rl.EndTextureMode()
		draw()
	}
}

func mcycle() {
	for i := 0; i < 4; i++ {
		bus.Cycle()
	}
}

// draw the window with debug info and update the PPU. PPU handles the game only.
func draw() {
	rl.BeginDrawing()
	rl.ClearBackground(color.RGBA{20, 20, 20, 255})
  if DEV {
  	drawDebugInfo()
  }
  if shouldDrawGame {
	  rl.DrawTexture(gameScreen.Texture, gameWindow.x, gameWindow.y, rl.White)
  } 
	rl.EndDrawing()
}

// Memory, registers, palettes, tiles etc.
func drawDebugInfo() {
  if !shouldDrawGame {
  	drawMem(memSelection*0x200, (memSelection+1)*0x200)
  }
	// drawMem(0x8000, 0xC000)
	drawInstructions()
	drawRegisters()
	drawIO()
	tilePixels = getTileData()
	drawTiles()

	rl.DrawTextEx(debugFont, fmt.Sprintf("Dot: %d", bus.ppu.dot), rl.Vector2{float32(debugX + 150), float32(100 - fontSize)}, float32(fontSize), 0, rl.LightGray)
	rl.DrawTextEx(debugFont, fmt.Sprintf("X: %d", bus.lcd.x), rl.Vector2{float32(debugX + 150), float32(100 - 2*fontSize)}, float32(fontSize), 0, rl.LightGray)

	rl.DrawTextEx(debugFont, "<-, -> Change Speed, ^, v Scroll Ram, [Space] Pause/Unpause, [A] 1 M-Cycle, [S] 1 Op, [M] Toggle Mem/Screen", rl.Vector2{float32(debugX), float32(window.h - 5 - int32(fontSize))}, float32(fontSize), 0, rl.Blue)
}

func drawMem(start, end int) {
	for row := 0; row <= (end-start)/bytesPerRow; row++ {
		out := fmt.Sprintf("%05X: ", start+row*bytesPerRow)
		for i := 0; i < bytesPerRow; i++ {
			b := start + row*bytesPerRow + i
			if b > 0xFFFF {
				return
			}
			if b >= 0x8000 && b <= 0x9FFF {
				out += fmt.Sprintf("%02X ", bus.ppu.vram[uint16(b-0x8000)])
			} else {
				out += fmt.Sprintf("%02X ", bus.Read(uint16(b)))
			}
			if i%4 == 3 {
				out += " "
			}
		}
		rl.DrawTextEx(debugFont, out, rl.Vector2{float32(gameWindow.x), float32(gameWindow.y + int32(row*fontSize))}, float32(fontSize), 0, rl.LightGray)
	}
}

func drawInstructions() {
	extra := instructionsPeekAmount

	// instructions[PC]
	s, ok := instructions[bus.cpu.PC]
	if ok {
		rl.DrawTextEx(debugFont, s, rl.Vector2{float32(debugX), float32(5 + extra*fontSize)}, float32(fontSize), 0, rl.Magenta)
	} else {

		rl.DrawTextEx(debugFont, fmt.Sprintf("%04X", bus.cpu.PC), rl.Vector2{float32(debugX), float32(5 + extra*fontSize)}, float32(fontSize), 0, rl.Magenta) // Temporary, to reduce stuttering during early testing
	}

	// instructions above
	var found = 0
	addr := bus.cpu.PC
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
	addr = bus.cpu.PC
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
	cpu := bus.cpu
	drawRegister(cpu.IR, "IR", 0, 0)
	drawRegister(cpu.WZ, "WZ", 0, 1)

	drawRegister(cpu.A, "A", 1, 0)
	drawRegister(cpu.BC, "BC", 2, 0)
	drawRegister(cpu.DE, "DE", 3, 0)
	drawRegister(cpu.HL, "HL", 4, 0)
	drawRegister(cpu.SP, "SP", 5, 0)

	drawRegister(getBit(7, cpu.F), "Z", 1, 1)
	drawRegister(getBit(6, cpu.F), "N", 2, 1)
	drawRegister(getBit(5, cpu.F), "HC", 3, 1)
	drawRegister(getBit(4, cpu.F), "C", 4, 1)

	drawRegister(cpu.IE, "IE", 6, 0)
	drawRegister(cpu.IF, "IF", 6, 1)
	drawRegister(cpu.IME, "IME", 6, 2)

}

func drawIO() {
	drawRegister(bus.Read(0xFF44), "LY", 1, 2)
	drawRegister(bus.DIV, "DIV", 2, 2)
	drawRegister(bus.TIMA, "TIMA", 3, 2)
	drawRegister(bus.TMA, "TMA", 4, 2)
}

func drawRegister(r interface{}, name string, col, row int) {
	var s string
	switch r.(type) {
	case byte:
		s = fmt.Sprintf("%2s: %02X", name, r)
	case uint16:
		s = fmt.Sprintf("%2s: %04X", name, r)
	default:
		log.Fatalf("first arg to drawRegister must be either byte or uint16")
	}
	rl.DrawTextEx(debugFont, s, rl.Vector2{float32(debugX + int32(150+col*4*fontSize)), float32(row*fontSize + 5)}, float32(fontSize), 0, rl.LightGray)
}

// Find all current tiledata from IDs in VRAM and return as []byte.
func getTileData() []byte {
	// var tileAddrStart uint16 = 0x8000
	var tileCount = 384                     // DMG
	var pixels = make([]byte, tileCount*64) // tiles * 8x8 pixels

	for tRow := 0; tRow < tileCount*8; tRow++ {
		rAddr := uint16(tRow * 2)
		loByte := bus.ppu.vram[rAddr]
		hiByte := bus.ppu.vram[rAddr+1]
		for b := 7; b >= 0; b-- {
			leftBit := getBit(b, hiByte)
			rightBit := getBit(b, loByte)
			colourID := (leftBit << 1) | rightBit
			pixels[tRow*8+(7-b)] = colourID
		}
	}

	return pixels
}

// Draw tile data image in debug area.
func drawTiles() {
	tilesPerRow := 16
	tilesPerColumn := 8
	pixPerRow := tilesPerRow * 8
	pixPerTile := 64

	for block := 0; block < 3; block++ {
		blockStart := block * tilesPerRow * tilesPerColumn * pixPerTile
		for ty := 0; ty < 8; ty++ {
			tyOffset := ty * tilesPerRow * pixPerTile
			for tx := 0; tx < 16; tx++ {
				txOffset := tx * pixPerTile
				tileAddr := blockStart + tyOffset + txOffset
				drawTile(tileAddr, debugX+int32(block*(pixPerRow+1)+tx*8), window.h-64-5-int32(fontSize)+int32(ty*8))
			}
		}
	}
}

func drawTile(addr int, x, y int32) {
	for row := 0; row < 8; row++ {
		for column := 0; column < 8; column++ {
			colourId := tilePixels[addr+row*8+column]
			c := colours[(bus.Read(0xFF47)>>(colourId*2))&0x3]

			rl.DrawPixel(x+int32(column), y+int32(row), c)
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

		// cartridge header
		// if addr >= 0x100 && addr <= 0x14F {
		// 	addr++
		// 	instructions[lineAddr] = instStr + "HEADER"
		// 	continue
		// }

		// Read byte from bus
		opcode := bus.Read(addr)
		addr++

		// Lookup opcode
		inst := lookup(opcode, false)
		instStr += inst.Op

		if inst.Op == "PREFIX" {
			instructions[lineAddr] = instStr
			instStr = fmt.Sprintf("%04X ", addr)
			lineAddr = addr
			opcode = bus.Read(addr)
			addr++
			inst = lookup(opcode, true)
			instStr += inst.Op

			if (opcode >> 4) > 3 {
				instStr += fmt.Sprintf(" %d", inst.Bit)
			}
			instStr += " " + string(inst.To)
		} else {

			// Depending on inst type, read X amount of bytes and perform a certain action on them
			switch inst.DataType {
			case NODATA:
				if inst.To != "" {
					instStr += " " + string(inst.To)
				}
				if inst.From != "" {
					instStr += fmt.Sprintf(", %s", inst.From)
				}
				if inst.Op == "RST" {
					instStr += fmt.Sprintf(" %02X", inst.Abs)
				} else if inst.Op == "RET" {
					instStr += fmt.Sprintf(" %s", inst.Flag)
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

				if inst.To == m8 {
					instStr += fmt.Sprintf(" [0xFF%02X]", a8)
				} else if inst.To != "" {
					instStr += " " + string(inst.To)
				}

				if inst.From == m8 {
					instStr += fmt.Sprintf(", [0xFF%02X]", a8)
				} else if inst.From != "" {
					instStr += fmt.Sprintf(", %s", inst.From)
				}
			case A16:
				lo := bus.Read(addr)
				addr++
				hi := bus.Read(addr)
				addr++
				a16 := joinBytes(hi, lo)

				if inst.To == m16 {
					instStr += fmt.Sprintf(" [0x%04X]", a16)
				} else if inst.To != "" {
					instStr += " " + string(inst.To) + ","
				}

				if inst.From == m16 {
					if inst.Op == "CALL" {
						instStr += fmt.Sprintf(" 0x%04X", a16)
					} else {
						if inst.Flag != NOFLAG {
							instStr += fmt.Sprintf(" %s,", inst.Flag)
						}
						instStr += fmt.Sprintf(" [0x%04X]", a16)
					}
				} else if inst.From != "" {
					instStr += fmt.Sprintf(" %s", inst.From)
				}
			case E8:
				n8 := bus.Read(addr)
				addr++
				e8 := int8(n8)
				if inst.Flag != NOFLAG && inst.Flag != "" {
					instStr += fmt.Sprintf(" %s,", inst.Flag)
				}
				if inst.To != "" {
					// HL or SP
					instStr += fmt.Sprintf(" %s,", inst.To)
				}
				instStr += fmt.Sprintf(" 0x%02X (%d)", n8, e8)
			default:
				log.Printf("Op: %s, disassembled unexpected DataType %v\n", inst.Op, inst.DataType)
				// continue
			}
		}

		// Add string to map
		instructions[lineAddr] = instStr
	}

	return instructions
}
