package alu

import (
  "testing"
)

func TestALUAdd(t *testing.T) {
  alu := &ALU{}

  testCases := []struct{
    a, b, oldC byte
    result, hc, c byte
  }{
    {a: 0xFF, b: 0x01, oldC: 0, result: 0x00, hc: 1, c: 1},
    {a: 0xFE, b: 0x01, oldC: 0, result: 0xFF, hc: 0, c: 0},
    {a: 0x0F, b: 0x01, oldC: 0, result: 0x10, hc: 1, c: 0},
    {a: 0xF0, b: 0x10, oldC: 0, result: 0x00, hc: 0, c: 1},
    {a: 0x01, b: 0xFF, oldC: 0, result: 0x00, hc: 1, c: 1},
    {a: 0xFF, b: 0xFF, oldC: 0, result: 0xFE, hc: 1, c: 1},
    {a: 0xFE, b: 0x01, oldC: 1, result: 0x00, hc: 1, c: 1},
    {a: 0xFF, b: 0x00, oldC: 1, result: 0x00, hc: 1, c: 1},
  }


  for i, tt := range testCases {
    result, hc, c := alu.ALUAdd(tt.a, tt.b, tt.oldC)

    if result != tt.result {
      t.Errorf("test %d, result: want 0x%02X, got 0x%02X", i, tt.result, result)
    }

    if hc != tt.hc {
      t.Errorf("test %d, half carry: want %d, got %d", i, tt.hc, hc)
    }

    if c != tt.c {
      t.Errorf("test %d, carry: want %d, got %d", i, tt.c, c)
    }
  }
}

func TestALUSub(t *testing.T) {
  alu := &ALU{}

  testCases := []struct{
    a, b, oldC byte
    result, hc, c byte
  }{
    {a: 0x00, b: 0x01, oldC: 0, result: 0xFF, hc: 1, c: 1},
    {a: 0xFF, b: 0x01, oldC: 0, result: 0xFE, hc: 0, c: 0},
    {a: 0x10, b: 0x01, oldC: 0, result: 0x0F, hc: 1, c: 0},
    {a: 0x00, b: 0x10, oldC: 0, result: 0xF0, hc: 0, c: 1},
    {a: 0x00, b: 0xFF, oldC: 0, result: 0x01, hc: 1, c: 1},
    {a: 0xFE, b: 0xFF, oldC: 0, result: 0xFF, hc: 1, c: 1},
    {a: 0x00, b: 0x01, oldC: 1, result: 0xFE, hc: 1, c: 1},
    {a: 0x00, b: 0x00, oldC: 1, result: 0xFF, hc: 1, c: 1},
  }


  for i, tt := range testCases {
    result, hc, c := alu.ALUSub(tt.a, tt.b, tt.oldC)

    if result != tt.result {
      t.Errorf("test %d, result: want 0x%02X, got 0x%02X", i, tt.result, result)
    }

    if hc != tt.hc {
      t.Errorf("test %d, half carry: want %d, got %d", i, tt.hc, hc)
    }

    if c != tt.c {
      t.Errorf("test %d, carry: want %d, got %d", i, tt.c, c)
    }
  }
}

func TestALUInc(t *testing.T) {
  alu := &ALU{}

  testCases := []struct{
    in, result, hc byte
  }{
    {in:0x00, result:0x01, hc:0},
    {in:0x0F, result:0x10, hc:1},
    {in:0xF0, result:0xF1, hc:0},
    {in:0xFF, result:0x00, hc:1},
    {in:0x07, result:0x08, hc:0},
  }

  for i, tt := range testCases {
    result, hc := alu.ALUInc(tt.in)

    if result != tt.result {
      t.Errorf("test %d, result: want 0x%02X, got 0x%02X", i, tt.result, result)
    }

    if hc != tt.hc {
      t.Errorf("test %d, half carry: want %d, got %d", i, tt.hc, hc)
    }
  }
}

func TestALUDec(t *testing.T) {
  alu := &ALU{}

  testCases := []struct{
    in, result, hc byte
  }{
    {in:0x01, result:0x00, hc:0},
    {in:0x10, result:0x0F, hc:1},
    {in:0xF1, result:0xF0, hc:0},
    {in:0x00, result:0xFF, hc:1},
    {in:0x08, result:0x07, hc:0},
  }

  for i, tt := range testCases {
    result, hc := alu.ALUDec(tt.in)

    if result != tt.result {
      t.Errorf("test %d, result: want 0x%02X, got 0x%02X", i, tt.result, result)
    }

    if hc != tt.hc {
      t.Errorf("test %d, half carry: want %d, got %d", i, tt.hc, hc)
    }
  }
}

func TestAddSignedToUnsigned(t *testing.T) {
  alu := &ALU{}

  testCases := []struct{
    a uint16
    e byte
    want uint16
    hc, c byte
  }{
    {a: 0x0000, e: 0x01, want: 0x0001, hc: 0, c: 0},
    {a: 0x0000, e: 0xFF, want: 0xFFFF, hc: 0, c: 0},
    {a: 0x0100, e: 0xFF, want: 0x00FF, hc: 0, c: 0},
    {a: 0x00FF, e: 0xFF, want: 0x00FE, hc: 1, c: 1},
    {a: 0x01FF, e: 0x01, want: 0x0200, hc: 1, c: 1},
  }

  for _, tt := range testCases {
    // low byte
    lo, hc, c := alu.AddSignedToUnsigned(byte(tt.a & 0xFF), tt.e)
    // high byte
    var got uint16 = uint16(alu.Adjust(byte(tt.a >> 8), c)) << 8 | uint16(lo)

    if got != tt.want {
      t.Errorf("result: got %04X, want %04X", got, tt.want)
    }

    if hc != tt.hc {
      t.Errorf("hc: got %d, want %d", hc, tt.hc)
    }

    if c != tt.c {
      t.Errorf("c: got %d, want %d", c, tt.c)
    }
  }
}

func TestDecimalAdjust(t *testing.T) {
  alu := ALU{}

  testCases := []struct{
    a, f byte
    want, c byte
  }{
    // Comments describe flags before daa
    {a: 0xA3, f: 0x00, want: 0x03, c: 1}, // add, no carries
    {a: 99, f: 0x10, want: 195, c: 1}, // add, carry
    {a: 0x9C, f: 0x20, want: 0x02, c: 1}, // add, half carry
    {a: 0x00, f: 0x30, want: 0x66, c: 1}, // add, half carry and carry
    {a: 0x7F, f: 0x30, want: 0xE5, c: 1}, // add, both carries
    {a: 0x85, f: 0x30, want: 0xEB, c: 1}, // add, both carries
    {a: 0xF3, f: 0x40, want: 0xF3, c: 0}, // sub, no carries
    {a: 0x6D, f: 0x40, want: 0x6D, c: 0}, // sub, no carries
    {a: 149, f: 0x50, want: 53, c: 1}, // sub, carry
    {a: 219, f: 0x60, want: 213, c: 0}, // sub, half carry
    {a: 96, f: 0x70, want: 250, c: 1}, // sub, both carries
    {a: 0xD7, f: 0x80, want: 0x37, c: 1}, // add, no carries
    {a: 180, f: 0x90, want: 20, c: 1}, // add, carry
    {a: 0x5F, f: 0xA0, want: 0x65, c: 0}, // add, half carry
    {a: 0x10, f: 0xB0, want: 118, c: 1}, // add, both carries
    {a: 0x04, f: 0xC0, want: 0x04, c: 0}, // sub, no carries
    {a: 0x51, f: 0xD0, want: 0xF1, c: 1}, // sub, carry
    {a: 0x31, f: 0xD0, want: 0xD1, c: 1}, // sub, carry
    {a: 0x9E, f: 0xE0, want: 0x98, c: 0}, // sub, half carry
    {a: 79, f: 0xF0, want: 233, c: 1}, // sub, both carries
  }

  for i, tt := range testCases {
    got, c := alu.DecAdj(tt.a, tt.f)

    if got != tt.want {
      t.Errorf("test %d, daa: got %02X, want %02X", i, got, tt.want)
    }

    if c != tt.c {
      t.Errorf("test %d, carry: got %d, want %d", i, c, tt.c)
    }
  }
}

func TestSwap(t *testing.T) {
  alu := ALU{}

  testCases := []struct{
    in byte
    want byte
  }{
    {in: 0x01, want: 0x10},
    {in: 0x00, want: 0x00},
  }

  for i, tt := range testCases {
    got := alu.ALUSwap(tt.in)

    if got != tt.want {
      t.Errorf("ALU Swap, test %d: got %02X, want %02X", i, got, tt.want)
    }
  }
}
