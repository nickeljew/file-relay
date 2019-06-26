package filerelay

import (
	"errors"
	"fmt"
	"io"
	"time"
)


const _ReserveLimit = time.Second * 2



//
type Slot struct {
	key string
	capacity uint64
	data []byte
	used uint64
	setAt time.Time
	duration time.Duration

	reservedAt time.Time
}

func NewSlot(capacity uint64) (s *Slot) {
	s = &Slot{
		capacity: capacity,
		data: make([]byte, capacity, capacity),
	}

	return 
}

func (s *Slot) Cap() uint64 {
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

func (s *Slot) Vacant() bool {
	if s.used == 0 || s.duration == 0 {
		return timePassed(s.reservedAt, _ReserveLimit)
	}
	return timePassed(s.setAt, s.duration)
}

func (s *Slot) CheckClear() bool {
	ok := s.Vacant()
	if ok && s.duration > 0 {
		s.Clear()
	}
	return ok
}

func (s *Slot) Reserve() {
	s.reservedAt = time.Now()
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

func (s *Slot) ReadAndSet(key string, r io.Reader, byteLen uint64) (used uint64, err error) {
	if s.Occupied() {
		errInfo := fmt.Sprintf("slot is occupied by key: %s; tried by: %s", s.key, key)
		return 0, errors.New(errInfo)
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
	if n, e := io.ReadFull(r, buf); n > 0 {
		used = uint64(n)
		s.used = used
		err = e
	}
	return
}
