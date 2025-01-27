package main

import (
    utils "github.com/mikzorz/gameboy-emulator/helpers"

)

type Clock struct {
	bus          *Bus
	speed        uint   // 4194304 Hz / 2^22 Hz
	DIV          uint16 // increments at 1048576 Hz / 16
	TIMA         byte
	TMA          byte // Timer Modulo
	TAC          byte
	timaOverflow bool
	prevAND      byte

	sysClock      uint
	ticksToDivInc int
}

func NewClock() *Clock {
	return &Clock{
		speed: 4194304,
		// DIV: 0xABCC, // according to one github repo
	}
}

var divBit = []int{
	9, // 4096 Hz
	3, // 262144 Hz
	5, // 65536 Hz
	7, // 16384 Hz
}

func (c *Clock) Tick() {
	c.sysClock++
	// c.ticksToDivInc++

	// if c.ticksToDivInc == 256 {
		c.DIV++
		// c.ticksToDivInc = 0
	// }

	if c.timaOverflow {
		c.timaOverflow = false
		c.TIMA = c.TMA
		c.bus.InterruptRequest(TIMER_INTR)
	}

	bitToCheck := divBit[c.TAC&0x3]
	curAND := byte((c.DIV>>bitToCheck)&0x1) & utils.GetBit(2, c.TAC)

	// "falling edge"
	if curAND == 0 && c.prevAND == 1 {
		c.TIMA++
		if c.TIMA == 0x00 {
			c.timaOverflow = true
		}
	}

	c.prevAND = curAND
}
