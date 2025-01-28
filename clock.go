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
	prevAND      byte

	sysClock               uint

	TIMAState timaState
}

type timaState int

const (
	TIMA_NO_OVERFLOW timaState = iota
	TIMA_DELAYING
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

func (c *Clock) UpdateTIMAState() {
  switch c.TIMAState {
  case TIMA_RELOADED:
		c.TIMAState = TIMA_NO_OVERFLOW
  case TIMA_DELAYING:
		c.TIMA = c.TMA
		c.bus.InterruptRequest(TIMER_INTR)
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
			c.TIMAState = TIMA_DELAYING
		}
	}
	c.prevAND = curAND

}
