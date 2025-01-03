package main

type Cart struct {
	rom  []byte
	bank byte
	ram  [0x2000]byte
	bus  *Bus
}

func NewCart() *Cart {
	return &Cart{
		ram:  [0x2000]byte{},
		bank: 1,
	}
}

func (c *Cart) LoadROMData(data []byte) {
	c.rom = data
}

func (c *Cart) SwitchBank(bank byte) {
	c.bank = bank
}

func (c *Cart) Read(addr uint16) byte {
	if addr <= 0x3FFF {
		return c.rom[addr]
	} else if addr <= 0x7FFF {
		return c.rom[addr+uint16(c.bank-1)*0x4000]
	} else {
		return c.ram[addr-0xA000]
	}
}

func (c *Cart) Write(addr uint16, data byte) {
	c.ram[addr-0xA000] = data
}
