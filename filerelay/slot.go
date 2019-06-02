package filerelay

import (
	"io"
	"time"
	"strings"
	"fmt"
)


// Error for handling Slot
type SlotError struct {
	info string
}

func (e *SlotError) Error() string {
	arr := []string{"SlotError: ", e.info,}
	return strings.Join(arr, "")
}



type Slot struct {
	cap int
	data []byte
	used int
	setAt time.Time
	duration time.Duration

	minExpiration int
}

func NewSlot(cap, minExpiration int) *Slot {
	if minExpiration < SlotMinExpriration {
		minExpiration = SlotMinExpriration
	}
	return &Slot{
		cap: cap,
		data: make([]byte, cap, cap),
		minExpiration: minExpiration,
	}
}

func (s *Slot) Cap() int {
	return s.cap
}

func (s *Slot) Clear() {
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

func (s *Slot) Data() []byte {
	return s.data[:s.used]
}

func (s *Slot) ReadAndSet(r io.Reader, byteLen int, expiration int) (n int, err error) {
	if s.Occupied() {
		return 0, &SlotError{"Slot is occupied"}
	}
	if byteLen > s.cap {
		return 0, &SlotError{"Invalid byteLen"}
	}
	if expiration < s.minExpiration {
		return 0, &SlotError{"expiratin too short"}
	}

	n = 0
	err = nil

	buf := s.data[:byteLen]
	if n, err = io.ReadFull(r, buf); n > 0 {
		s.used = n
		s.setAt = time.Now()

		secs := fmt.Sprintf("%ds", expiration)
		s.duration, _ = time.ParseDuration(secs)
	}
	return
}
