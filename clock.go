package main

type Clock struct {
	bus        *Bus
	speed      uint // 4194304 Hz
	ticks      uint
	sysCounter uint
}

func NewClock() *Clock {
	return &Clock{
		speed: 4194304,
	}
}

func (c *Clock) Tick() {
	c.ticks++
}

func (c *Clock) GetTicks() uint {
	return c.ticks
}
