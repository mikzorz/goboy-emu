package main

func joinBytes(msb, lsb byte) uint16 {
	return (uint16(msb) << 8) | uint16(lsb)
}

func lsb(nn uint16) byte {
	return uint8(nn & 0xFF)
}

func msb(nn uint16) byte {
	return uint8(nn >> 8)
}

// For relative jumps, need to add signed int e to PC
func addInt8ToUint16(e uint8, a uint16) (result uint16, flags uint8) {
	sign := isBitSet(7, e)
	lo := lsb(a)
	z := e + lo

	carry := z < lo // bool
	var c byte = 0  // used in flags return value
	halfCarry := ((lo ^ e ^ z) & 0x10) >> 4

	w := msb(a)
	if carry {
		c = 1
		if !sign {
			w += 1
		}
	} else if !carry && sign {
		w -= 1
	}
	// result := a + uint16(e&0x7F)
	// if sign {
	// 	result -= 128
	// }
	return joinBytes(w, z), (halfCarry << 5) | (c << 4)
	// return result
}

func isBitSet(col int, b byte) bool {
	return (b>>col)&0x1 == 1
}

func getBit(col int, b byte) byte {
	return (b >> col) & 0x1
}

func setBit(col int, b byte) byte {
	return b | (1 << col)
}

func resetBit(col int, b byte) byte {
	return b & (0xFF ^ (1 << col))
}
