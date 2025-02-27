package main

import (
	"testing"
)

func TestGetMapTileCoords(t *testing.T) {
	// Given an x, y, scx & scy, get tileX, tileY.
	testCases := []struct {
		x, y, scx, scy byte
		tx, ty         byte
	}{
		{x: 0, y: 0, scx: 0, scy: 0, tx: 0, ty: 0},
		{x: 1, y: 0, scx: 0, scy: 0, tx: 0, ty: 0},
		{x: 8, y: 0, scx: 0, scy: 0, tx: 1, ty: 0},
		{x: 255, y: 0, scx: 1, scy: 0, tx: 0, ty: 0}, //wrap x
		{x: 0, y: 0, scx: 16, scy: 0, tx: 2, ty: 0},
		{x: 0, y: 8, scx: 0, scy: 0, tx: 0, ty: 1},
		{x: 0, y: 8, scx: 0, scy: 8, tx: 0, ty: 2},
		{x: 0, y: 255, scx: 0, scy: 1, tx: 0, ty: 0},
		{x: 24, y: 24, scx: 8, scy: 8, tx: 4, ty: 4},
		{x: 255, y: 255, scx: 255, scy: 255, tx: 31, ty: 31}, // wrap x and y
	}

	ppu := NewPPU()

	for _, tt := range testCases {
		tx, ty := ppu.getTileCoords(tt.x, tt.y, tt.scx, tt.scy)

		if tx != tt.tx {
			t.Errorf("wrong tileX: got %d, want %d", tx, tt.tx)
		}

		if ty != tt.ty {
			t.Errorf("wrong tileY: got %d, want %d", ty, tt.ty)
		}
	}
}
