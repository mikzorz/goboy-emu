package main

import (
	"fmt"
	"testing"
)

// TODO, go through source code of failing test.
// Unit test each op

// -- DONE --
// lda addr = ldh a, addr-ff00
// sta addr = ldh addr-ff00, a
// wreg addr, data =
//    ld a, data
//    sta addr

// cpu_instrs, 01-special.gb
// #6 DAA
// DAA has been tested in alu_test.go, no problems

// cpu_instrs, 02-interrupts.gb
// set_test 4,"Timer doesn't work"
// delay 500
// lda  IF
// delay 500
// and  $04
// jp   nz,test_failed
// delay 500
// lda  IF
// and  $04
// jp   z,test_failed
// pop  af

// Other Ops used in macros/functions

// ldi a, (hl)
// delay 500 (cycles) = delay_ 500 & $FFFF, 500>>16
// delay_ low, high = ...
//    there's some push/pop af, some nops, maybe a ret
// test_failed =
//    ld l, a
//    ld h, a
//    ld a, (hl)
//    or a
//    jr z, e
//    call

func TestLD(t *testing.T) {
	t.Run("ld a, n", func(t *testing.T) {
		cpu := NewCPU()

		b := newBusStub()
		b.rom[0x100] = 0x44
		cpu.bus = b

		cpu.IR = 0x3E
		cpu.inst = lookup(cpu.IR, false)
		cpu.SetOpFunc()

		cpu.Cycle()
		cpu.Cycle()

		if cpu.A != 0x44 {
			b.printLogs()
			t.Errorf("cpu.A should be 0x%02X, got 0x%02X", 0x44, cpu.A)
		}
	})

	t.Run("LD HL, SP+e", func(t *testing.T) {
		cpu := NewCPU()

		b := newBusStub()
		b.rom[0x100] = 0xFE
		cpu.bus = b
		cpu.HL = 0x8000
		cpu.SP = 0xE000

		cpu.IR = 0xF8
		cpu.inst = lookup(cpu.IR, false)
		cpu.SetOpFunc()

		cpu.Cycle()
		cpu.Cycle()
		cpu.Cycle()

		if cpu.HL != 0xDFFE {
			b.printLogs()
			t.Errorf("cpu.HL should be 0x%04X, got 0x%04X", 0xDFFE, cpu.HL)
		}
	})
}

func TestLDH(t *testing.T) {
	t.Run("ldh a, (a8)", func(t *testing.T) {
		cpu := NewCPU()

		b := newBusStub()
		b.rom[0x100] = 0x01
		b.rom[0xff01] = 0x51
		cpu.bus = b

		cpu.IR = 0xF0
		cpu.inst = lookup(cpu.IR, false)
		cpu.setLDHFunc()

		// Z <- imm
		cpu.Cycle()
		// Z <- (FF00+Z)
		cpu.Cycle()
		// A <- Z
		cpu.Cycle()

		if cpu.A != 0x51 {
			b.printLogs()
			t.Errorf("cpu.A should be 0x%02X, got 0x%02X", 0x51, cpu.A)
		}

	})

	t.Run("ldh (a8), a", func(t *testing.T) {
		cpu := NewCPU()

		b := newBusStub()
		b.rom[0x100] = 0x02
		cpu.A = 0x77
		cpu.bus = b

		cpu.IR = 0xE0
		cpu.inst = lookup(cpu.IR, false)
		cpu.setLDHFunc()

		cpu.Cycle()
		cpu.Cycle()
		cpu.Cycle()

		if b.rom[0xff02] != 0x77 {
			b.printLogs()
			t.Errorf("0xff02 should be 0x%02X, got 0x%02X", 0x77, b.rom[0xff02])
		}

	})
}

func TestAdd(t *testing.T) {
	t.Run("ADD HL, SP", func(t *testing.T) {
		testCases := []struct {
			hl, sp, want uint16
			hc, c        byte
		}{
			{hl: 0x2000, sp: 0x4050, want: 0x6050, hc: 0, c: 0},
			{hl: 0xFFFF, sp: 0x0001, want: 0x0000, hc: 1, c: 1},
			{hl: 0x00FF, sp: 0x0001, want: 0x0100, hc: 0, c: 0},
			{hl: 0xF000, sp: 0x1000, want: 0x0000, hc: 0, c: 1},
			{hl: 0x0F00, sp: 0x0100, want: 0x1000, hc: 1, c: 0},
		}

		for _, tt := range testCases {
			cpu := NewCPU()
			cpu.bus = newBusStub()

			cpu.HL = tt.hl
			cpu.SP = tt.sp

			cpu.IR = 0x39
			cpu.inst = lookup(cpu.IR, false)
			cpu.SetOpFunc()
			cpu.clearFlags()

			cpu.Cycle()
			cpu.Cycle()

			if cpu.HL != tt.want {
				t.Fatalf("HL should = 0x%04X, got 0x%04X", tt.want, cpu.HL)
			}

			if cpu.getHalfCarry() != tt.hc {
				t.Fatalf("halfcarry is not correct")
			}

			if cpu.getCarry() != tt.c {
				t.Fatalf("carry is not correct")
			}
		}
	})

	t.Run("ADD SP, e", func(t *testing.T) {
		testCases := []struct {
			sp, want uint16
			e        byte
		}{
			{sp: 0x2000, e: 0x01, want: 0x2001},
			{sp: 0x2000, e: 0xFF, want: 0x1FFF},
		}

		for _, tt := range testCases {
			cpu := NewCPU()
			b := newBusStub()
			b.rom[0x100] = tt.e
			cpu.bus = b

			cpu.SP = tt.sp

			cpu.IR = 0xE8
			cpu.inst = lookup(cpu.IR, false)
			cpu.SetOpFunc()

			for i := 0; i < 4; i++ {
				cpu.Cycle()
			}

			if cpu.SP != tt.want {
				t.Fatalf("SP should = 0x%04X, got 0x%04X", tt.want, cpu.SP)
			}
		}

	})
}

type busStub struct {
	log []string
	rom []byte
}

func newBusStub() *busStub {
	return &busStub{
		rom: newTestRom(),
	}
}

func (b *busStub) Read(addr uint16) byte {
	var v byte = b.rom[addr]
	b.log = append(b.log, fmt.Sprintf("read 0x%02X from 0x%04X", v, addr))
	return v

}

func (b *busStub) Write(addr uint16, data byte) {
	b.rom[addr] = data
	b.log = append(b.log, fmt.Sprintf("write 0x%02X to 0x%04X", data, addr))

}

func (b *busStub) isHalted() bool {
	// return b.halted
	return false
}

func (b *busStub) setHalt(v bool) {
	// b.halted = v
}

func (b *busStub) printLogs() {
	for _, s := range b.log {
		fmt.Println(s)
	}
}

func newTestRom() []byte {
	rom := make([]byte, 0x10000)
	// for i := 0; i < 0x100; i++ {
	//   rom[i] = 0x00
	// }
	//
	// for i, v := range data {
	//   rom[i+0x100] = v
	// }
	return rom
}
