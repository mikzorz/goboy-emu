package main

type Cart struct {
	rom  []byte
	bank byte
  secondaryBank byte
	ram  [0x2000]byte
	bus  *Bus

  bankingMode byte // 0 or 1

  numOfBanks byte
}

func NewCart() *Cart {
	return &Cart{
		ram:  [0x2000]byte{},
		bank: 1,
	}
}

func (c *Cart) LoadROMData(data []byte) {
	c.rom = data

  // Trying to access an out of range bank causes a wrap around based on the required number of bits for the amount of banks in the rom.
  // e.g. a 256KiB rom has 16 banks, which is 4 bits.
  // Trying to access bank 16 (0x10) should(?) wrap to bank 0 (0x10 & 0xF == 0x0)
  numOfBanks := len(c.rom) / 0x4000
  if len(c.rom) % 0x4000 != 0 {
    numOfBanks++
  }
  c.numOfBanks = byte(numOfBanks)
}

func (c *Cart) SwitchBank(data byte) {
	bank := data & 0x1F
  if bank == 0x0 {
    bank++
  }
	c.bank = bank
}

func (c *Cart) Read(addr uint16) byte {
	if addr <= 0x3FFF {
    if c.bankingMode == 1 {
      bank := c.secondaryBank << 5
      bank %= c.numOfBanks
  		return c.rom[uint(addr) + c.getBankAddressOffset(bank)]
    } else {
      return c.rom[addr]
    }
	} else if addr <= 0x7FFF {
    bank := c.bank | (c.secondaryBank << 5) // In MBC1 multicart, shift 4 instead of 5, original bit.4 is ignored
    bank %= c.numOfBanks
		return c.rom[uint(addr - 0x4000) + c.getBankAddressOffset(bank)]
	} else {
		return c.ram[addr - 0xA000]
	}
}

func (c *Cart) Write(addr uint16, data byte) {
  switch {
	case (addr >= 0x0000 && addr <= 0x1FFF):
    c.setRAMEnable(data)
	case (addr >= 0x2000 && addr <= 0x3FFF):
		c.SwitchBank(data)
  case (addr >= 0x4000 && addr <= 0x5FFF):
    // RAM bank or Upper bits of ROM bank
    c.secondaryBank = data & 0x3
  case (addr >= 0x6000 && addr <= 0x7FFF):
    // Banking Mode
    c.setBankingMode(data)
	case (addr >= 0xA000 && addr <= 0xBFFF):
	  c.ram[addr-0xA000] = data // TODO ram banking
  }
}

func (c *Cart) setRAMEnable(data byte) {
  if data == 0x0A {
    // TODO
    // enable cart RAM
  } else {
    // disable cart RAM
  }
}

// If mode == 0, 0x0000 - 0x3fff is locked to rom bank 0, A000-BFFF = ram bank 0
// If mode == 1, those banks can be switched 
func (c *Cart) setBankingMode(data byte) {
  c.bankingMode = getBit(0, data)
}

func (c *Cart) getBankAddressOffset(bank byte) uint {
  return uint(bank) * 0x4000
}
