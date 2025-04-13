package main

import (
	utils "github.com/mikzorz/goboy-emu/helpers"
)

type Joypad struct {
	JOYP       byte
	Directions byte
	Buttons    byte
}

func NewJoypad() *Joypad {
	return &Joypad{
		Directions: 0xFF,
		Buttons:    0xFF,
	}
}

type Button int

const (
	JoyA Button = iota
	JoyB
	JoySelect
	JoyStart
	JoyRight
	JoyLeft
	JoyUp
	JoyDown
)

func (j *Joypad) Press(b Button) {
	switch b {
	case JoyA:
		j.Buttons = utils.ResetBit(0, j.Buttons)
	case JoyB:
		j.Buttons = utils.ResetBit(1, j.Buttons)
	case JoySelect:
		j.Buttons = utils.ResetBit(2, j.Buttons)
	case JoyStart:
		j.Buttons = utils.ResetBit(3, j.Buttons)
	case JoyRight:
		j.Directions = utils.ResetBit(0, j.Directions)
	case JoyLeft:
		j.Directions = utils.ResetBit(1, j.Directions)
	case JoyUp:
		j.Directions = utils.ResetBit(2, j.Directions)
	case JoyDown:
		j.Directions = utils.ResetBit(3, j.Directions)
	}
}

func (j *Joypad) Release(b Button) {
	switch b {
	case JoyA:
		j.Buttons = utils.SetBit(0, j.Buttons)
	case JoyB:
		j.Buttons = utils.SetBit(1, j.Buttons)
	case JoySelect:
		j.Buttons = utils.SetBit(2, j.Buttons)
	case JoyStart:
		j.Buttons = utils.SetBit(3, j.Buttons)
	case JoyRight:
		j.Directions = utils.SetBit(0, j.Directions)
	case JoyLeft:
		j.Directions = utils.SetBit(1, j.Directions)
	case JoyUp:
		j.Directions = utils.SetBit(2, j.Directions)
	case JoyDown:
		j.Directions = utils.SetBit(3, j.Directions)
	}
}

func (j *Joypad) Read() byte {
	var ret byte = j.JOYP
	if !utils.IsBitSet(4, j.JOYP) {
		// Directions
		ret = (ret & 0xF0) | (j.Directions & 0xF)
	} else if !utils.IsBitSet(5, j.JOYP) {
		// Buttons
		ret = (ret & 0xF0) | (j.Buttons & 0xF)
	}
	return ret | 0xC0 // bits 6 and 7 always return 1
}

func (j *Joypad) Write(data byte) {
	keySelect := data & 0x30
	j.JOYP = keySelect | (j.JOYP & 0xF) // TODO preserving bottom nibble might not do anything
}
