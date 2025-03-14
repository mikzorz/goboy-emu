Game Boy emulator written in Go


The debug info is drawn using a hardcoded Noto font. If you don't have it, replace `debugFontPath` in debug.go with a font that you like. At some point, a font may be provided with the emulator.

The automated tests expect the test roms to be in a certain location.

Big TODO List

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

- Check all of the other TODOs
