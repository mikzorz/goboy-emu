package main

type Pixel struct {
	c          byte // 0-3
	pal        byte // bit 4 OAM byte 3, 0=OBP0, 1=OBP1
	bgPriority byte  // bit 7 OAM byte 3, 0=obj above bg, 1=bg above obj
}

type FIFO []Pixel

func NewFIFO() *FIFO {
	return &FIFO{}
}

func (f FIFO) CanPush() bool {
	if len(f) <= 8 {
		return true
	}
	return false
}

func (f FIFO) CanPushBG() bool {
	if len(f) == 0 {
		return true
	}
	return false
}

// Push new pixels to the object FIFO.
// If new pixels are pushed to a non-empty FIFO, every pixel up to Len() in the FIFO is kept.
// Only the pixels after, and including, Len() are added from the new pixels.
// e.g. If there are 5 pixels in the FIFO, but the last 2 are transparent, and a new 8 pixels are pushed, then only the last 5 of the new pixels are added to the FIFO.
func (f *FIFO) PushObject(data []Pixel) {
  l := f.Len()

  data = data[l:]
  *f = (*f)[:l]
  *f = append(*f, data...)
}

func (f *FIFO) Push(data []Pixel) {
  *f = data
}

func (f FIFO) CanPop() bool {
	if len(f) > 0 {
		return true
	}
	return false
}

func (f *FIFO) Pop() Pixel {
	pix := (*f)[0]
	*f = (*f)[1:]
	return pix
}

func (f *FIFO) Clear() {
	*f = *NewFIFO()
}

// Returns the number of pixels up to and including the last non-transparent pixel.
// Object FIFO only.
func (f *FIFO) Len() int {
  length := len(*f)
  for i := len(*f) - 1; i >= 0; i-- {
    if (*f)[i].c == 0 {
      length--
    } else {
      break
    }
  }
  return length
}
