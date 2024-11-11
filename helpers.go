package main

func joinBytes(msb, lsb byte) uint16 {
	return (uint16(msb) << 8) | uint16(lsb)
}
