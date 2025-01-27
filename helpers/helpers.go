package helpers

func JoinBytes(msb, lsb byte) uint16 {
	return (uint16(msb) << 8) | uint16(lsb)
}

func LSB(nn uint16) byte {
	return byte(nn & 0xFF)
}

func MSB(nn uint16) byte {
	return byte(nn >> 8)
}

func JoinNibbles(msn, lsn byte) byte {
	return (msn << 4) | lsn
}

// least significant nibble
func LSN(n byte) byte {
	return n & 0xF
}

// most significant nibble
func MSN(n byte) byte {
	return (n & 0xF0) >> 4
}

func IsBitSet(col int, b byte) bool {
	return (b>>col)&0x1 == 1
}

func GetBit(col int, b byte) byte {
	return (b >> col) & 0x1
}

func SetBit(col int, b byte) byte {
	return b | (1 << col)
}

func ResetBit(col int, b byte) byte {
	return b & (0xFF ^ (1 << col))
}
