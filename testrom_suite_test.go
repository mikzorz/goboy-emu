package main

import (
	// "fmt"
	"testing"
)

// Test rom automation

// For mts, if op == 0x40 (ld b, b) and NOT prefix, check reg values for pass/fail
// Pass: B/C/D/E/H/L = 3/5/8/13/21/34 (not hex)
// Fail: all = 0x42
// To speed things up, LY and SC should always return 0xFF

var path = "roms/mts-20240926-1737-443f6e1/acceptance"

// Start with 2 roms that currently pass and fail respectively.
var roms = []string{
	"/instr/daa.gb",
	// "/ret_cc_timing.gb", // never ends
	"/rapid_di_ei.gb",
	// "/rst_timing.gb",
	"/boot_hwio-dmgABCmgb.gb",
	"/ppu/lcdon_write_timing-GS.gb",
	// "/ppu/intr_2_mode3_timing.gb", // gets stuck
	// "/ppu/intr_2_mode0_timing_sprites.gb",
	// "/ppu/intr_2_mode0_timing.gb",
	"/ppu/stat_lyc_onoff.gb",
	"/ppu/lcdon_timing-GS.gb",
	// "/ppu/intr_2_0_timing.gb",
	"/ppu/hblank_ly_scx_timing-GS.gb",
	"/ppu/stat_irq_blocking.gb",
	"/ppu/vblank_stat_intr-GS.gb",
	"/ppu/intr_1_2_timing-GS.gb",
	// "/ppu/intr_2_oam_ok_timing.gb",
	"/ld_hl_sp_e_timing.gb",
	"/halt_ime1_timing2-GS.gb",
	"/oam_dma_restart.gb",
	"/if_ie_registers.gb",
	"/oam_dma_timing.gb",
	"/push_timing.gb",
	"/call_timing2.gb",
	"/boot_regs-dmgABC.gb",
	"/bits/mem_oam.gb",
	"/bits/reg_f.gb",
	"/bits/unused_hwio-GS.gb",
	"/reti_intr_timing.gb",
	"/halt_ime1_timing.gb",
	"/jp_timing.gb",
	"/di_timing-GS.gb",
	// "/ret_timing.gb",
	"/interrupts/ie_push.gb",
	"/ei_timing.gb",
	"/oam_dma_start.gb",
	"/timer/tim10.gb",
	"/timer/tim11.gb",
	"/timer/tim00.gb",
	"/timer/rapid_toggle.gb",
	"/timer/tim11_div_trigger.gb",
	"/timer/tim10_div_trigger.gb",
	"/timer/tim01_div_trigger.gb",
	"/timer/tma_write_reloading.gb",
	"/timer/tima_write_reloading.gb",
	"/timer/tima_reload.gb",
	"/timer/tim00_div_trigger.gb",
	"/timer/tim01.gb",
	"/timer/div_write.gb",
	"/oam_dma/sources-GS.gb",
	"/oam_dma/reg_read.gb",
	"/oam_dma/basic.gb",
	"/boot_div-dmgABCmgb.gb",
	"/serial/boot_sclk_align-dmgABCmgb.gb",
	"/pop_timing.gb",
	"/div_timing.gb",
	"/add_sp_e_timing.gb",
	"/ei_sequence.gb",
	"/jp_cc_timing.gb",
	"/call_cc_timing.gb",
	"/halt_ime0_nointr_timing.gb",
	"/call_timing.gb",
	"/call_cc_timing2.gb",
	"/intr_timing.gb",
	"/halt_ime0_ei.gb",
	// "/reti_timing.gb",
}

func TestRoms(t *testing.T) {

	for _, rom := range roms {
		t.Run(rom, func(t *testing.T) {

			cart := NewCart()
			bus := NewBus(cart)
			bus.screenDisabled = true

			ReadRomFile(cart, path+rom)
			populatePrefixLookup()

			cpu := bus.cpu

			for {
				for cycle := 0; cycle < 4; cycle++ {
					bus.Cycle()
				}

				// LD B, B
				if cpu.inst.Op == "LD" && cpu.IR == 0x40 {
					// Check register values
					if cpu.BC == 0x4242 && cpu.DE == 0x4242 && cpu.HL == 0x4242 {
						// Test failed
						t.Fail()
						break
					}

					if cpu.BC == 0x0305 && cpu.DE == 0x080D && cpu.HL == 0x1522 {
						// Test passed
						break
					}
				}
			}
		})
	}
}
