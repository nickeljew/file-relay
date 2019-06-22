package filerelay

import (
	"container/list"
	"errors"
	"math"
	"sync"
	"time"

	. "github.com/nickeljew/file-relay/debug"
)

const (
	slabsCheckConc = 3
)


type Slab struct {
	slotCap int
	slots *list.List
	checkTime int64
	checkIntv int //in seconds
	sync.Mutex
}

func NewSlab(slotCap, slotCount, checkIntv int) *Slab {
	if checkIntv < SlabCheckInterval {
		checkIntv = SlabCheckInterval
	}
	slab := Slab{
		slotCap: slotCap,
		slots: list.New(),
		checkTime: time.Now().Unix(),
		checkIntv: checkIntv,
	}
	for i := 0; i < slotCount; i++ {
		slab.slots.PushBack( NewSlot(slotCap) )
	}
	return &slab
}

func (s *Slab) Capacity() int {
	return s.slotCap * s.slots.Len()
}

func (s *Slab) FindAvailableSlot() *Slot {
	s.Lock()
	defer s.Unlock()

	elem := s.slots.Front()
	slot := elem.Value.(*Slot)
	if slot.CheckClear() {
		slot.Reserve() //reserve slot for avoiding found by others
		s.slots.MoveToBack(elem)
		return slot
	}

	if pass := int(time.Now().Unix() - s.checkTime); pass >= s.checkIntv {
		n := int(math.Floor(float64(s.slots.Len() / 10)))
		if n <= 0 {
			n = 1
		} else if n > 5 {
			n = 5
		}

		elem = s.tryClearFromLast(n)
		if elem != nil {
			s.slots.MoveToBack(elem)
			slot = elem.Value.(*Slot)
			slot.Reserve() //reserve slot for avoiding found by others
			return slot
		}
		s.checkTime = time.Now().Unix()
	}

	return nil
}

func (s *Slab) tryClearFromLast(n int) (el *list.Element) {
	var elem *list.Element
	var slot *Slot
	for ; n > 0; n-- {
		elem = s.slots.Back()
		slot = elem.Value.(*Slot)
		if slot.CheckClear() {
			s.slots.MoveToFront(elem)
			el = elem
		}
	}
	return
}




type SlabGroup struct {
	slotCap int
	slotSum int
	checkIntv int

	maxStorageSize int
	totalCap int

	slabs *list.List
	sync.Mutex
}

type SlabCh chan *Slab

func NewSlabGroup(slotCap, slabCount, slotCount, checkIntv, maxSize int) *SlabGroup {
	group := SlabGroup{
		slotCap: slotCap,
		checkIntv: checkIntv,
		maxStorageSize: maxSize,
		slabs: list.New(),
	}

	_ = group.AddSlabs(slabCount, slotCount)

	return &group
}

func (g *SlabGroup) AddSlabs(slabCount, slotCount int) (cap int) {
	for i := 0; i < slabCount; i++ {
		s := NewSlab(g.slotCap, slotCount, g.checkIntv)
		g.slotSum += slotCount
		cap += s.Capacity()
		g.slabs.PushBack(s)
	}
	g.totalCap += cap
	return
}

func (g *SlabGroup) SlotSum() int {
	return g.slotSum
}

func (g *SlabGroup) Capacity() int {
	return g.totalCap
}

func (g *SlabGroup) FindAvailableSlots(need, currentTotalCap int) (s []*Slot, e error) {
	if need > g.SlotSum() {
		return nil, errors.New("too many slots to request")
	}

	s = make([]*Slot, 0, need)
	result := make(SlabCh)
	conc, cnt := slabsCheckConc, need
	SlabSum := g.slabs.Len()
	slabsLeft, slotsLeft := SlabSum, g.slotSum
	if conc > need {
		conc = need
	}
	if sl := g.slabs.Len(); conc > sl {
		conc = sl
	}
	Debugf("-- Find for Cap: %d - Need-slots: %d, Total-slabs: %d, Total-slots: %d; conc: %d", g.slotCap, need, slabsLeft, slotsLeft, conc)

	ForEnd:
	for {
		if cnt == 0 || slotsLeft == 0 {
			break
		}
		if slabsLeft == 0 {
			slabsLeft = SlabSum
		}

		slabsLeft = g.doCheck(conc, slabsLeft, result)
		//Debugf("-- Loop for finding slot: %d - Slabs-left: %d, Slots-left: %d, Need-slots-left: %d", g.slotCap, slabsLeft, slotsLeft, cnt)

		select {
		case slab := <- result:
			if slab == nil {
				panic("No slab found")
			}

			conc = 1
			slot := slab.FindAvailableSlot()
			slotsLeft--
			if slot != nil {
				s = append(s, slot)
				cnt--
			}

			got := len(s)
			//Debugf("-- Next for Cap: %d - Need-slots-left: %d, Got-slots: %d", g.slotCap, cnt, got)
			if got >= need {
				//break //Fuck there! can't break the loop inside select
				break ForEnd
			}
		}
	}

	if cnt > 0 {
		e = errors.New("no enough slots")

		if currentTotalCap < g.maxStorageSize {
			//
		}
	}
	Debugf(" - finished for Cap: %d - got: %d, Slots-left: %d (%d)", g.slotCap, len(s), slotsLeft, g.slotSum)
	return
}

func (g *SlabGroup) doCheck(conc, total int, r SlabCh) int {
	left := total
	for i := 0; i < conc; i++ {
		go g.getSlabForCheck(r)
		left--
	}
	return left
}

func (g *SlabGroup) getSlabForCheck(r SlabCh) {
	g.Lock()

	elem := g.slabs.Front()
	slab := elem.Value.(*Slab)
	g.slabs.MoveToBack(elem)
	r <- slab

	g.Unlock()
}
