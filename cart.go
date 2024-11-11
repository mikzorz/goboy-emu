package main

type Cart struct {
	rom []byte
}

func NewCart() *Cart {
	return &Cart{}
}

func (c *Cart) LoadROMData(data []byte) {
	c.rom = data
}

func (c *Cart) Read(addr uint16) byte {

	return c.rom[addr]
}
