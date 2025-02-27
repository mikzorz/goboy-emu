package main

type Pixel struct {
	c          byte // 0-3
	pal        byte // bit 4 OAM byte 3, 0=OBP0, 1=OBP1
	bgPriority int  // bit 7 OAM byte 3, 0=obj above bg, 1=bg above obj
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

func (f *FIFO) Push(data []Pixel) {
	for _, p := range data {
		*f = append(*f, p)
	}
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
