package main

import (
	"testing"
)

func TestIncrementTIMA(t *testing.T) {
	c := &Clock{}

	c.DIV = 0
	c.TIMA = 0
	c.TAC = 0b101

	for i := 0; i < 15; i++ {
		c.Tick()
		c.IncrementTIMA()
	}

	if c.TIMA != 0 {
		t.Errorf("TIMA should still be 0, got %d", c.TIMA)
	}

	c.Tick()
	c.IncrementTIMA()

	if c.TIMA != 1 {
		t.Errorf("TIMA should have incremented to 1, got %d", c.TIMA)
	}
}
