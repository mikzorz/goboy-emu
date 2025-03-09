package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// const DEV = true

const DEV = false

const GAMEBOY_DOCTOR = false

var logfile *os.File

var enableDebugInfo bool

// Big TODO List

// - Y-flip objects
// - Windows might need shifting by a pixel
// - Fix screen tearing (very noticable in Zelda when camera moves to left and right)
// - Fix top row(s)
// - Audio
// - Save data
// - Change default controls
// - Dr Mario freezes after the menus

// - Pass more tests (need to fix timing differences)

// - When test reaches loop at end, pressing [s] freezes emu, requiring forced quit
//    End of test is JR to itself, causing infinite loop

// - My implementation of a bus is completely wrong for a gameboy. The gameboy has 2(?) main buses, 1 goes to vram via ppu? Buses do not have clocks.

// - Finish setting default values
// - Rearrange components to match hardware more closely

// - Check all of the other TODOs

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

// const TRUEWIDTH int32 = 256 // Some test messages are too long to fit on the normal screen.
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

func ReadRomFile(c *Cart, romPath string) {
	data, err := os.ReadFile(romPath)
	if err != nil {
		log.Panic(err)
	}

	// copy cart bytes from file to byte slice
	c.LoadROMData(data)
}

func _init() {
	flag.StringVar(&romPath, "rom", "", "The path to the rom file.")
	flag.Parse()

	// Load ROM
	if romPath == "" {
		fmt.Println("no rom provided")
		os.Exit(1)
	} else {
		// fmt.Println(romPath)
		ReadRomFile(cart, romPath)

		populatePrefixLookup()
		if DEV {
			// instructions = disassemble(disAssembleStart, disAssembleEnd)
			enableDebugInfo = true
		} else {
			gameWindow.x, gameWindow.y = 0, 0
			window.w, window.h = gameWindow.w, gameWindow.h
			enableDebugInfo = false
		}

	}
}

func main() {

	_init()
	if GAMEBOY_DOCTOR {
		var err error
		logfile, err = os.Create("gbdoctor_logfile.log")
		if err != nil {
			log.Fatal(err)
		}
		defer logfile.Close()
		bus.screenDisabled = true
		bus.alwaysVblank = true
		for i := 0; i < 400000; i++ { // Not an endless loop, filled RAM accidentally.
			for cycle := 0; cycle < 4; cycle++ {
				bus.Cycle()
			}
		}
		os.Exit(0)
	}

	rl.InitWindow(window.w, window.h, "Game Boy Emulator made in Go")
	defer rl.CloseWindow()

	if DEV {
		debugFont = rl.LoadFont(debugFontPath)
		defer rl.UnloadFont(debugFont)
	}

	gameScreen = rl.LoadRenderTexture(TRUEWIDTH, TRUEHEIGHT)
	defer rl.UnloadRenderTexture(gameScreen)

	rl.SetTargetFPS(60)

	defer func() {
		// if r := recover(); r != nil {
		// 	log.Printf("Crashed at address 0x%04X\n", bus.cpu.PC)
		// 	log.Printf("OP: 0x%02X %s, opcycle: %d\n", bus.cpu.IR, bus.cpu.inst.Op, bus.cpu.curCycle)
		// }
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
				// disAssembleStart = bus.cpu.PC //
				disAssembleEnd = bus.cpu.PC + instructionsPeekAmount + 10
				// instructions = disassemble(disAssembleStart, disAssembleEnd)
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
