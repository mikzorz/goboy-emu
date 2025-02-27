package main

import (
	// "log"
	utils "github.com/mikzorz/gameboy-emulator/helpers"
)

type DMA struct {
	bus          *Bus
	oam          [0xA0]byte
	dmaRequested bool
	oamDMA       bool // oam dma transfer in progress
	oamSource    byte // high byte of oam source address
	nextSource   byte
	oamTransferI byte // byte to fetch
	oamByte      byte
}

func NewDMA() *DMA {
	return &DMA{
		oam: [0xA0]byte{},
	}
}

func (d *DMA) Cycle() {

	if d.oamDMA {
		d.oamDMA = false // give dma access to memory
		// if d.oamTransferI == 0 {
		// 	srcAddr := utils.JoinBytes(d.oamSource, d.oamTransferI)
		// 	d.oamByte = d.bus.Read(srcAddr)
		// } else {
		// d.oam[d.oamTransferI-1] = d.oamByte
		srcAddr := utils.JoinBytes(d.oamSource, d.oamTransferI)
		d.oamByte = d.bus.Read(srcAddr)
		d.oam[d.oamTransferI] = d.oamByte

		if d.oamTransferI >= 0x9F {
			// d.oamDMA = false
			return
		}
		// } else {
		// 	srcAddr := utils.JoinBytes(d.oamSource, d.oamTransferI)
		// 	d.oamByte = d.bus.Read(srcAddr)
		// }
		// }
		d.oamTransferI++
		d.oamDMA = true // reblock memory from other hardware
	}

	if d.dmaRequested {
		d.oamDMA = true
		d.oamTransferI = 0
		d.oamSource = d.nextSource
		d.dmaRequested = false
	}
}

func (d *DMA) StartOAMTransfer(source byte) {
	d.nextSource = source
	d.dmaRequested = true
}

func (d *DMA) Read(addr uint16) byte {
	if d.oamDMA {
		return 0xFF
	} else {
		return d.oam[addr-0xFE00]
	}
}
