# Game Boy emulator written in Go

> [!WARNING]
> This emulator is a work-in-progress-on-hiatus.

## About

 Goboy is a Game Boy (DMG) emulator written in Go. The purpose of this project was to demystify how emulators work and to see what I could put together just by reading the docs. This emulator is currently not worth using to play games, it is missing features like audio and saving. The features that have been implemented kind of work.  

 At first, I avoided looking at other GB emulators. I wanted to figure out how to code the functionality by myself based on what the pandocs explained about the hardware. By the end, I was reading everyone else's code.

 There was a lot of information to take in all at once and I didn't know how to begin, so I checked out some NES emulator tutorials for some hints. This led to me making wrongful assumptions about the hardware of the GB, which is a bit different to the NES. Not only that, different resources claim different things. I had to start referencing various other docs and forum posts during the development process.

 If I started from scratch, knowing what I do now about the hardware, I would use TDD from the very beginning, for two main reasons:

1) I had a bug with one particular opcode that was used by Blargg's test roms to setup tests. The buggy opcode was causing multiple tests to fail and the test roms weren't able to tell me what the problem was. I scoured through my code looking for a needle in a haystack. Unit tests could have prevented this.

2) TDD would have encouraged me to decouple the hardware components from each other and to use dependency injection.

This is all in hindsight, of course. When I started, I was worried about writing tests based on wrongful assumptions. I just wanted to see *something* working.

## Usage

Build
```
git clone https://github.com/mikzorz/goboy-emu.git
cd goboy-emu
go build .
```

Run  
`./goboy-emu -rom ROM_PATH`

> [!NOTE]
> Hardcoded values

 The debug info is drawn using a hardcoded Noto font. If you don't have it, replace `debugFontPath` in debug.go with a font that you like. At some point, a font may be provided with the emulator.

`testrom_suite_test.go` automatically runs a whole bunch of mooneye acceptance tests.
Mooneye test suite needs to be downloaded separately (should probably include as a git submodule or something).
Set the `path` variable in `testrom_suite_test.go` to the path of the directory containing the acceptance tests.

DEV mode can be toggled in main.go

## TODO

- Y-flip objects (Zelda map has an object that needs flipping)
- Change default controls
- Save data
- Audio
- Windows might need shifting by a pixel
- Fix screen tearing (very noticable in Zelda when camera moves to left and right)
- Fix top row(s)
- Dr Mario & Mario Picross both freeze after the initial menus.
- In Yugioh, when selecting a monster to attack with, a small menu in bottom right is not rendered correctly.

- Pass more tests (need to fix timing differences)

- When test reaches loop at end, pressing [s] freezes emu, requiring forced quit
   End of test is JR to itself, causing infinite loop

- My implementation of a bus is completely wrong for a gameboy. The gameboy has 2(?) main buses, 1 goes to vram via ppu? Buses do not have clocks.

- Finish setting default values
- Rearrange components to match hardware more closely

- Check all of the other TODOs scattered throughout the code
