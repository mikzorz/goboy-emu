package main

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
		j.Buttons = resetBit(0, j.Buttons)
	case JoyB:
		j.Buttons = resetBit(1, j.Buttons)
	case JoySelect:
		j.Buttons = resetBit(2, j.Buttons)
	case JoyStart:
		j.Buttons = resetBit(3, j.Buttons)
	case JoyRight:
		j.Directions = resetBit(0, j.Directions)
	case JoyLeft:
		j.Directions = resetBit(1, j.Directions)
	case JoyUp:
		j.Directions = resetBit(2, j.Directions)
	case JoyDown:
		j.Directions = resetBit(3, j.Directions)
	}
}

func (j *Joypad) Release(b Button) {
	switch b {
	case JoyA:
		j.Buttons = setBit(0, j.Buttons)
	case JoyB:
		j.Buttons = setBit(1, j.Buttons)
	case JoySelect:
		j.Buttons = setBit(2, j.Buttons)
	case JoyStart:
		j.Buttons = setBit(3, j.Buttons)
	case JoyRight:
		j.Directions = setBit(0, j.Directions)
	case JoyLeft:
		j.Directions = setBit(1, j.Directions)
	case JoyUp:
		j.Directions = setBit(2, j.Directions)
	case JoyDown:
		j.Directions = setBit(3, j.Directions)
	}
}

func (j *Joypad) Read() byte {
	var ret byte = j.JOYP
	if !isBitSet(4, j.JOYP) {
		// Directions
		ret = (ret & 0xF0) | (j.Directions & 0xF)
	} else if !isBitSet(5, j.JOYP) {
		// Buttons
		ret = (ret & 0xF0) | (j.Buttons & 0xF)
	}
	return ret
}

func (j *Joypad) Write(data byte) {
	keySelect := data & 0x30
	j.JOYP = keySelect | (j.JOYP & 0xF) // TODO preserving bottom nibble might not do anything
}
