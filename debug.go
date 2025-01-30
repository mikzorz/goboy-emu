package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	utils "github.com/mikzorz/gameboy-emulator/helpers"
	"log"
	"strings"
)

// TODO, remove hardcoded font or provide one with project.
var debugFontPath = "/usr/share/fonts/noto/NotoSansMono-Regular.ttf"
var debugFont rl.Font

var shouldDrawGame bool = true                                                                              // switch between displaying game screen or memory
var memRowExample = "00000: " + strings.Repeat("00 ", bytesPerRow) + strings.Repeat(" ", (bytesPerRow/4)-1) // -1 because real string has extraneous space. could remove but...
var memRowWidth = int32(((len(memRowExample) * fontSize) / 11) * 5)                                         // approximation
var memSelection = 0
var debugX int32 = max(memRowWidth, gameWindow.w) + 5

// Debug control
var paused = true
var cyclesPerFrame = 8
var breakpoints = map[uint16]bool{}

// int = how many occurrences to skip before pausing.
// multiply by 4, because it decrements per tick, not per cycle
var opOccurrences = map[byte]int{
	// 0x40: 0, // LD B, B (used by mts tests, but also unintentionally matches CB 40)
}

var opsWithArgs = map[byte]byte{}

// Break after X amount of t-cycles
var cyclebreaks = map[int]bool{}
var curCycle = 0

var disAssembleStart, disAssembleEnd uint16 = 0x0000, 0xFFFF

const instructionsPeekAmount = 12 // How many lines above and below current instruction to show?

func handleDebugInput() {
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

	// 10 Ops (varying amount of cycles)
	if rl.IsKeyPressed(rl.KeyD) {
		for i := 0; i < 100; i++ {
			prevOp := bus.cpu.IR
			for prevOp == bus.cpu.IR {
				mcycle()
			}
		}
	}

	if rl.IsKeyPressed(rl.KeySpace) {
		paused = !paused
	}

	if rl.IsKeyPressed(rl.KeyRight) {
		cyclesPerFrame *= 2
	}
	if rl.IsKeyPressed(rl.KeyLeft) {
		cyclesPerFrame = max(cyclesPerFrame/2, 1)
	}

	if rl.IsKeyPressed(rl.KeyLeftControl) {
		enableDebugInfo = !enableDebugInfo
	}
}

func checkBreakpoints() bool {
	if _, ok := breakpoints[bus.cpu.PC]; ok {
		return true
	} else if occ, ok := opOccurrences[bus.cpu.IR]; ok {
		if occ > 0 {
			opOccurrences[bus.cpu.IR]--
		} else {
			return true
		}
	} else if n8, ok := opsWithArgs[bus.cpu.IR]; ok {
		if bus.Read(bus.cpu.PC) == n8 {
			return true
		}
	} else if _, ok := cyclebreaks[curCycle]; ok {
		return true
	}
	return false
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
  drawOAM()

  // Extra info (may or may not be actual registers)
	rl.DrawTextEx(debugFont, fmt.Sprintf("Dot: %d", bus.ppu.dot), rl.Vector2{float32(debugX + 150), float32(100 - fontSize)}, float32(fontSize), 0, rl.LightGray)
	rl.DrawTextEx(debugFont, fmt.Sprintf("X: %d", bus.lcd.x), rl.Vector2{float32(debugX + 150), float32(100 - 2*fontSize)}, float32(fontSize), 0, rl.LightGray)
	rl.DrawTextEx(debugFont, fmt.Sprintf("tcycle: %d", curCycle), rl.Vector2{float32(debugX + 150), float32(100 - 3*fontSize)}, float32(fontSize), 0, rl.LightGray)

	rl.DrawTextEx(debugFont, "<-, -> Change Speed, ^, v Scroll Ram, [Space] Pause/Unpause, [A] 1 M-Cycle, [S] 1 Op, [D] 100 Ops, [M] Toggle Mem/Screen, [LCtrl] Toggle Debugger", rl.Vector2{float32(5), float32(window.h - 5 - int32(fontSize))}, float32(fontSize), 0, rl.Blue)
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

	drawRegister(utils.GetBit(7, cpu.F), "Z", 1, 1)
	drawRegister(utils.GetBit(6, cpu.F), "N", 2, 1)
	drawRegister(utils.GetBit(5, cpu.F), "HC", 3, 1)
	drawRegister(utils.GetBit(4, cpu.F), "C", 4, 1)

	drawRegister(cpu.IE, "IE", 6, 0)
	drawRegister(cpu.IF, "IF", 6, 1)
	drawRegister(cpu.IME, "IME", 6, 2)

}

func drawIO() {
	c := bus.clock
	drawRegister(bus.Read(0xFF44), "LY", 1, 2)
	drawRegister(c.DIV, "DIV", 2, 2)
	drawRegister(c.TIMA, "TIMA", 3, 2)
	drawRegister(c.TMA, "TMA", 4, 2)
	drawRegister(c.TAC, "TAC", 5, 2)
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
	var pixels = []byte{} // tiles * 8x8 pixels

  for t := 0; t < tileCount; t++ {
    tilePixels := getTileDataByID(t)
    pixels = append(pixels, tilePixels...)
  }

	return pixels
}

// Return []byte containing 64 colour IDs, row by row.
func getTileDataByID(tileID int) []byte {
  pixels := make([]byte, 64)
  // For each row of 8 pixels
  //  Get the low and high bytes
  //  Pair nth bits of each byte to get colour ID
  //  Save colour ID to a []byte
	for tRow := 0; tRow < 8; tRow++ {
		rAddr := uint16((tileID * 8 + tRow) * 2)
		loByte := bus.ppu.vram[rAddr]
		hiByte := bus.ppu.vram[rAddr+1]
		for b := 7; b >= 0; b-- {
			leftBit := utils.GetBit(b, hiByte)
			rightBit := utils.GetBit(b, loByte)
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
				tileIdx := blockStart + tyOffset + txOffset
				drawTile(tilePixels, tileIdx, debugX+int32(block*(pixPerRow+1)+tx*8), window.h-64-5-int32(fontSize)+int32(ty*8), 0xFF47)
			}
		}
	}
}

func drawTile(pixels []byte, idx int, x, y int32, palAddr uint16) {
	for row := 0; row < 8; row++ {
		for column := 0; column < 8; column++ {
			colourId := pixels[idx+row*8+column]
			c := colours[(bus.Read(palAddr)>>(colourId*2))&0x3]

			rl.DrawPixel(x+int32(column), y+int32(row), c)
		}
	}
}

// Find all current OAM data from IDs in OAM and return as []byte.
func getOAMTileIDs() []byte {
  var objectCount = 40
  // var oamAddr = 0xFE00
  
  // Every 4th tile, starting from FE02, is a tile index
  tileIndices := []byte{}
  for o := 0; o < objectCount; o++ {
    tileIndices = append(tileIndices, bus.dma.oam[o * 4 + 2])
  }

  return tileIndices
}

func getOAMPixels() []byte {
  pixels := []byte{}
  ids := getOAMTileIDs()
  for _, id := range ids {
    tilePixels := getTileDataByID(int(id)) // TODO, account for bit.4 of byte 3 of oam data, DMG palette
    pixels = append(pixels, tilePixels...)
  }
  return pixels
}

func drawOAM() {
  tilesPerRow := 5
  tilesPerColumn := 8
  // pixPerRow := tilesPerRow * 8
  pixPerTile := 64

  pixels := getOAMPixels()

  for ty := 0; ty < tilesPerColumn; ty++ {
    tyOffset := ty * tilesPerRow * pixPerTile
    for tx := 0; tx < tilesPerRow; tx++ {
      txOffset := tx * pixPerTile
      idx := tyOffset + txOffset
      // 3*(16*8+1) puts OAM tiles after tile maps
			drawTile(pixels, idx, debugX+int32(3*(16*8+1)+tx*8), window.h-64-5-int32(fontSize)+int32(ty*8), 0xFF47)
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
				n16 := utils.JoinBytes(hi, lo)
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
				a16 := utils.JoinBytes(hi, lo)

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
