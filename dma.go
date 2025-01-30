package main

import (
  // "log"
	utils "github.com/mikzorz/gameboy-emulator/helpers"
)

type DMA struct {
	bus                                                    *Bus
	oam                                                    [0xA0]byte
	oamDMA                                                 bool // oam dma transfer in progress
	oamSource                                              byte // high byte of oam source address
	oamTransferI                                           byte // byte to fetch
	oamByte                                                byte
}

func NewDMA() *DMA {
  return &DMA{
		oam:  [0xA0]byte{},
  }
}

func (d *DMA) Cycle() {
  if d.oamDMA {
    if d.oamTransferI == 0 {
      srcAddr := utils.JoinBytes(d.oamSource, d.oamTransferI)
      d.oamByte = d.bus.Read(srcAddr)
    } else {
      d.oam[d.oamTransferI-1] = d.oamByte

      if d.oamTransferI >= 160 {
        d.oamDMA = false
      } else {
        srcAddr := utils.JoinBytes(d.oamSource, d.oamTransferI)
        d.oamByte = d.bus.Read(srcAddr)
      }
    }
    d.oamTransferI++
  }
}

func (d *DMA) StartOAMTransfer(source byte) {
  d.oamDMA = true
  d.oamSource = source
  d.oamTransferI = 0
}
