package filerelay

import (
	"errors"
	"io"
	"time"
)



type Slot struct {
	key string
	capacity int
	data []byte
	used int
	setAt time.Time
	duration time.Duration
}

func NewSlot(capacity int) (s *Slot) {
	s = &Slot{
		capacity: capacity,
		data: make([]byte, capacity, capacity),
	}

	return 
}

func (s *Slot) Cap() int {
	return s.capacity
}

func (s *Slot) Clear() {
	s.key = ""
	s.used = 0
	s.duration = 0
}

func (s *Slot) Occupied() bool {
	return s.used > 0 && s.duration > 0
}

func (s *Slot) Available() bool {
	if s.used == 0 || s.duration == 0 {
		return true
	}

	now := time.Now()
	diff := now.Sub(s.setAt)
	return diff > s.duration
}

func (s *Slot) CheckClear() bool {
	ok := s.Available()
	if ok && s.duration > 0 {
		s.Clear()
	}
	return ok
}

func (s *Slot) Key() string {
	return s.key
}

func (s *Slot) Data() []byte {
	return s.data[:s.used]
}

func (s *Slot) SetInfoWithItem(t *MetaItem) {
	s.setAt = t.setAt
	s.duration = t.duration
}

func (s *Slot) ReadAndSet(key string, r io.Reader, byteLen int) (n int, err error) {
	if s.Occupied() {
		return 0, errors.New("slot is occupied")
	}
	if l := len(key); l == 0 || l > KeyMax {
		return 0, errors.New("key too long")
	}
	if byteLen > s.capacity {
		//return 0, errors.New("Invalid byteLen")
		byteLen = s.capacity
	}

	s.key = key
	buf := s.data[:byteLen]
	if n, err = io.ReadFull(r, buf); n > 0 {
		s.used = n
	}
	return
}
