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

	sysClock               uint
	ticksToDivInc          int
	ticksUntilTIMAOverflow int
	cancelTimerIntr        bool

	TIMAState timaState
}

type timaState int

const (
	TIMA_NO_OVERFLOW timaState = iota
	TIMA_DELAYING
	TIMA_DELAY_FINISHED
  TIMA_RELOADED
)

func NewClock() *Clock {
	return &Clock{
		speed:     4194304,
		DIV:       0xABCC, // according to cycle accurate docs
		TIMAState: TIMA_NO_OVERFLOW,
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
	c.DIV++
}

// Timer notes
// during delay, writing to TIMA prevents interrupts and TMA reload, TIMA = new TIMA
// when load is about to happen, writing to TIMA will be ignored

// Writing to IF overwrites (maybe not necessary to add)

// when load is about to happen, writes to TMA also write to TIMA

func (c *Clock) DecrementCountdown() {
	// c.ticksUntilTIMAOverflow--
}

func (c *Clock) UpdateTIMAState() {
  switch c.TIMAState {
  case TIMA_RELOADED:
		c.TIMAState = TIMA_NO_OVERFLOW
  case TIMA_DELAYING:
		// c.timaOverflow = false
		c.TIMAState = TIMA_RELOADED
		c.TIMA = c.TMA
		// if !c.cancelTimerIntr {
		c.bus.InterruptRequest(TIMER_INTR)
		// }
		c.TIMAState = TIMA_RELOADED
	}

}

func (c *Clock) IncrementTIMA() {

	bitToCheck := divBit[c.TAC&0x3]
	curAND := byte((c.DIV>>bitToCheck)&0x1) & utils.GetBit(2, c.TAC)

	// "falling edge"
	if curAND == 0 && c.prevAND == 1 {
		c.TIMA++
		if c.TIMA == 0x00 {
			// c.timaOverflow = true
			// c.ticksUntilTIMAOverflow = 1
			c.TIMAState = TIMA_DELAYING
			// c.cancelTimerIntr = false
		// c.TIMA = c.TMA // according to sameboy
		}
	}
	c.prevAND = curAND

}
