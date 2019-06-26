package filerelay

import (
	"container/list"
	"errors"
	"math"
	"sync"
	"time"
)

const (
	_SlabsCheckConc = 3
)


type Slab struct {
	slotCap uint64
	slots *list.List
	checkTime int64
	checkIntv int //in seconds
	sync.Mutex
}

func NewSlab(slotCap uint64, slotCount, checkIntv int) *Slab {
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

func (s *Slab) SlotCount() int {
	return s.slots.Len()
}

func (s *Slab) Capacity() uint64 {
	return s.slotCap * uint64(s.slots.Len())
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
	slotCap uint64
	initSlabCount int
	slotNumInSlab int
	slotSum int
	checkIntv int

	maxStorageSize uint64
	totalCap uint64

	slabs *list.List
	sync.Mutex
}

type SlabCh chan *Slab

func NewSlabGroup(slotCap uint64, slabCount, slotCount, checkIntv int, maxSize uint64) *SlabGroup {
	group := SlabGroup{
		slotCap: slotCap,
		initSlabCount: slabCount,
		slotNumInSlab: slotCount,
		checkIntv: checkIntv,
		maxStorageSize: maxSize,
		slabs: list.New(),
	}

	_ = group.AddSlabs(slabCount)

	return &group
}

func (g *SlabGroup) AddSlabs(slabCount int) (cap uint64) {
	cap = 0
	for i := 0; i < slabCount; i++ {
		s := NewSlab(g.slotCap, g.slotNumInSlab, g.checkIntv)
		g.slotSum += s.SlotCount()
		cap += s.Capacity()
		g.slabs.PushFront(s)
	}
	g.totalCap += cap
	return
}

func (g *SlabGroup) SlotSum() int {
	return g.slotSum
}

func (g *SlabGroup) Capacity() uint64 {
	return g.totalCap
}

func (g *SlabGroup) slabCheckConcurrency(slabCount int) int {
	if slabCount < 30 {
		return _SlabsCheckConc
	}
	return int( math.Round(float64(slabCount) / 10 + 0.5) )
}

func (g *SlabGroup) FindAvailableSlots(key string, need int, currentTotalCap uint64) ([]*Slot, uint64, error) {
	if need > g.SlotSum() {
		return nil, 0, errors.New("too many slots to request")
	}

	slots := make([]*Slot, 0, need)
	result := make(SlabCh)
	cnt := need

	var slabsLeft, slotsLeft int

	startCheck := func() {
		slabSum := g.slabs.Len()
		conc := g.slabCheckConcurrency(slabSum)
		slabsLeft, slotsLeft = slabSum, g.slotSum
		if conc > cnt {
			conc = cnt
		}
		if sl := g.slabs.Len(); conc > sl {
			conc = sl
		}
		memTrace.Logf("-- <%s> Find for Cap: %d - Need-slots: %d, Total-slabs: %d, Total-slots: %d; conc: %d", key, g.slotCap, cnt, slabsLeft, slotsLeft, conc)

		ForEnd:
		for {
			if cnt == 0 || slotsLeft == 0 {
				break
			}
			if slabsLeft == 0 {
				slabsLeft = slabSum
			}

			slabsLeft = g.doCheck(key, conc, slabsLeft, result)
			//memTrace.Logf("-- <%s> In-loop for finding slot: %d - Slabs-left: %d, Slots-left: %d, Need-slots-left: %d", key, g.slotCap, slabsLeft, slotsLeft, cnt)

			select {
			case slab := <- result:
				got := len(slots)
				if got >= need {
					if got > need {
						slots = slots[0:need]
					}
					break ForEnd
				}

				if slab == nil {
					panic("No slab found")
				}

				conc = 1
				slot := slab.FindAvailableSlot()
				slotsLeft--
				if slot != nil {
					slots = append(slots, slot)
					cnt--
					got++
				}

				//memTrace.Logf("-- <%s> Next for Cap: %d - Need-slots-left: %d, Got-slots: %d", key, g.slotCap, cnt, got)
				if got == need {
					break ForEnd
				}
			}
		}
	}

	startCheck()

	var cap uint64
	if cnt > 0 {
		if currentTotalCap < g.maxStorageSize {
			g.Lock()

			ext := int( math.Round(float64(g.initSlabCount) / 2) )
			if ext < cnt {
				ext = cnt
			}
			var extSz uint64
			for ; ext > 0; ext-- {
				extSz = uint64(ext) * uint64(g.slotNumInSlab) * g.slotCap
				if sumCap := currentTotalCap + extSz; sumCap < g.maxStorageSize {
					break
				}
			}
			memTrace.Logf("mark memory[%d] - total: %d, need: %d; max-limit: %d",
				g.slotCap, currentTotalCap, extSz, g.maxStorageSize)
			cap = g.AddSlabs(ext)

			g.Unlock()

			startCheck()
			if len(slots) < need {
				return nil, 0, errors.New("no enough slots")
			}
		} else {
			return nil, 0, errors.New("storage full")
		}

	}

	memTrace.Logf(" - finished for Cap: %d - got: %d, Slots-left: %d (%d)", g.slotCap, len(slots), slotsLeft, g.slotSum)
	return slots, cap, nil
}

func (g *SlabGroup) doCheck(key string, conc, total int, r SlabCh) int {
	//memTrace.Logf("-- <%s> Check group[%d] with %d slabs; conc: %d, total: %d", key, g.slotCap, g.slabs.Len(), conc, total)
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

	g.Unlock() //unlock before send it to chan to allow next round to proceed
	r <- slab
}
