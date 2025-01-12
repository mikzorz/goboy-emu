package main

type Clock struct {
	bus          *Bus
	speed        uint   // 4194304 Hz / 2^22 Hz
	DIV          uint16 // increments at 1048576 Hz
	TIMA         byte
	TMA          byte // Timer Modulo
	TAC          byte
	timaOverflow bool
	prevAND      byte
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
	c.DIV++

	if c.timaOverflow {
		c.timaOverflow = false
		c.TIMA = c.TMA
		c.bus.InterruptRequest(TIMER_INTR)
	}

	bitToCheck := divBit[c.TAC&0x3]
	curAND := byte((c.DIV>>bitToCheck)&0x1) & getBit(2, c.TAC)

	// "falling edge"
	if curAND == 0 && c.prevAND == 1 {
		if c.TIMA == 0xFF {
			c.timaOverflow = true
			c.TIMA = 0
		} else {
			c.TIMA++
		}
	}

	c.prevAND = curAND
}
