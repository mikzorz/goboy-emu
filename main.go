package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const DEV = true

// const DEV = false

var enableDebugInfo bool

// Big TODO List

// Draw OAM tiles in debug screen.

// Pass more tests
// Currently, instr_timing.gb fails during test setup
//  The timer may trigger an interrupt slightly too late.
// This could be a timer issue, but it could also be another instr taking the wrong amount of m-cycles.
//  Look at the different ops used for timer setup, check their timings.

// When test reaches loop at end, pressing [s] freezes emu, requiring forced quit
// End of test is JR to itself, causing infinite loop

// Try adding basic audio for test beeps.

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
// Block off everything except HRAM during OAM_DMA? Necessary, or just for accuracy?

// I would also make register read/writes less messy, with more descriptive functions.

var joypadMap = map[int32]Button{
	rl.KeyZ:         JoyA,
	rl.KeyX:         JoyB,
	rl.KeyBackspace: JoySelect,
	rl.KeyEnter:     JoyStart,
	rl.KeyL:         JoyRight,
	rl.KeyJ:         JoyLeft,
	rl.KeyI:         JoyUp,
	rl.KeyK:         JoyDown,
}

var romPath string
var romData []byte

type Screen struct {
	w, h int32
	x, y int32
}

const TRUEWIDTH int32 = 160
const TRUEHEIGHT int32 = 144

var gameWinScale int32 = 3

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

// main window
// game screen size + some space.
var window = Screen{
	w: gameWindow.w + 10 + 960,
	h: gameWindow.h + 10 + 320,
	x: 0,
	y: 0,
}

// var cpu = NewCPU()
var ppu = NewPPU()
var cart = NewCart()
var bus = NewBus(cart)

func setup() {
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
		cart.LoadROMData(data)

		populatePrefixLookup()
		if DEV {
			instructions = disassemble(disAssembleStart, disAssembleEnd)
      enableDebugInfo = true
		} else {
			gameWindow.x, gameWindow.y = 0, 0
			window.w, window.h = gameWindow.w, gameWindow.h
      enableDebugInfo = false
		}
	}
}

func main() {

	setup()

	rl.InitWindow(window.w, window.h, "Game Boy Emulator made in Go")
	defer rl.CloseWindow()

	debugFont = rl.LoadFont(debugFontPath)
	defer rl.UnloadFont(debugFont)

	gameScreen = rl.LoadRenderTexture(TRUEWIDTH, TRUEHEIGHT)
	defer rl.UnloadRenderTexture(gameScreen)

	rl.SetTargetFPS(60)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Crashed at address 0x%04X\n", bus.cpu.PC)
			log.Printf("OP: 0x%02X %s, opcycle: %d\n", bus.cpu.IR, bus.cpu.inst.Op, bus.cpu.curCycle)
		}
	}()

	for !rl.WindowShouldClose() {
		rl.BeginTextureMode(gameScreen)

		getJoypadInput()

		if DEV {
			handleDebugInput()

			if !paused {
				for i := 0; i < cyclesPerFrame; i++ {
					paused = checkBreakpoints()

					if !paused {
						tick()
					}
				}
			}

      if enableDebugInfo {
        disAssembleStart = bus.cpu.PC - instructionsPeekAmount - 10 // 10 margin
        disAssembleEnd = bus.cpu.PC + instructionsPeekAmount + 10
        instructions = disassemble(disAssembleStart, disAssembleEnd)
      }
		} else {
			for i := 0; i < 70224; i++ {
				tick()
			}
		}

		rl.EndTextureMode()
		draw()
	}
}

func tick() {
	bus.Cycle()
	curCycle++
}

func mcycle() {
	for i := 0; i < 4; i++ {
		tick()
	}
}

func getJoypadInput() {

	for k, input := range joypadMap {
		if rl.IsKeyDown(k) {
			bus.joypad.Press(input)
		} else {
			bus.joypad.Release(input)
		}
	}

}

// draw the window with debug info and update the PPU. PPU handles the game only.
func draw() {
	rl.BeginDrawing()
	rl.ClearBackground(color.RGBA{20, 20, 20, 255})
	if DEV && enableDebugInfo {
		drawDebugInfo()
	}
	if shouldDrawGame {
		rl.DrawTextureEx(gameScreen.Texture, rl.Vector2{float32(gameWindow.x), float32(gameWindow.y)}, 0, float32(gameWinScale), rl.White)
	}
	rl.EndDrawing()
}
