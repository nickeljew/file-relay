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
			return elem.Value.(*Slot)
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
	slotCount int
	totalCap int

	slabs *list.List
	sync.Mutex
}

type SlabCh chan *Slab

func NewSlabGroup(slotCap, slabCount, slotCount, checkIntv int) *SlabGroup {
	group := SlabGroup{
		slotCap: slotCap,
		slotCount: slotCount,
		slabs: list.New(),
	}
	for i := 0; i < slabCount; i++ {
		group.slabs.PushBack( NewSlab(slotCap, slotCount, checkIntv) )
	}
	return &group
}

func (g *SlabGroup) SlotSum() int {
	return g.slotCount * g.slabs.Len()
}

func (g *SlabGroup) FindAvailableSlots(need int) (s []*Slot, e error) {
	if need > g.SlotSum() {
		return nil, errors.New("too many slots to request")
	}

	s = make([]*Slot, 0, need)
	result := make(SlabCh)
	conc, did, cnt, left := slabsCheckConc, 0, need, g.slabs.Len()
	if conc > need {
		conc = need
	}
	//Debugf("-- Find for Cap: %d - Need: %d, Total: %d", g.slotCap, need, left)

	ForEnd:
	for {
		if cnt > 0 && left > 0 {
			did, left = g.doCheck(conc, left, result)
			cnt -= did
		}

		select {
		case slab := <- result:
			conc = 1
			if slab == nil {
				continue
			}
			slot := slab.FindAvailableSlot()
			if slot != nil {
				s = append(s, slot)
			}
			//Debugf("-- Next for Cap: %d - Need-left: %d, Slots: %d", g.slotCap, cnt, len(s))

			if len(s) >= need {
				//break //Fuck there! can't break the loop inside select
				break ForEnd
			}
		}
	}

	if cnt > 0 {
		e = errors.New("no enough slots")
	}
	Debugf(" - finished for Cap: %d - got: %d, Total-slots-left: %d", g.slotCap, len(s), left)
	return
}

func (g *SlabGroup) doCheck(conc, total int, r SlabCh) (did, left int) {
	left = total
	for {
		if did >= conc || left <= 0 {
			return
		}
		go g.getSlabForCheck(r)
		did++
		left--
	}
}

func (g *SlabGroup) getSlabForCheck(r SlabCh) {
	g.Lock()
	defer g.Unlock()
	
	elem := g.slabs.Front()
	slab := elem.Value.(*Slab)
	g.slabs.MoveToBack(elem)
	r <- slab
}
