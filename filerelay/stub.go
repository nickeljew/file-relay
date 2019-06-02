package filerelay

import (
	"time"
	"sync"
	"errors"
	"fmt"

	"github.com/nickeljew/file-relay/list"
)

const (
	stubsCheckConc = 3
)


type Stub struct {
	slotCap int
	slots *list.LinkedList
	checkTime int64
	checkIntv int
	sync.Mutex
}

func NewStub(slotCap, slotCount, checkIntv int) *Stub {
	if (checkIntv < StubCheckInterval) {
		checkIntv = StubCheckInterval
	}
	stub := Stub{
		slotCap: slotCap,
		slots: list.New(),
		checkTime: time.Now().Unix(),
		checkIntv: checkIntv,
	}
	for i := 0; i < slotCount; i++ {
		stub.slots.Push( NewSlot(slotCap) )
	}
	return &stub
}

func (s *Stub) Capacity() int {
	return s.slotCap * s.slots.Length()
}

func (s *Stub) FindAvailableSlot() *Slot {
	s.Lock()
	defer s.Unlock()
	
	entry := s.slots.GetFirst()
	slot := entry.Value().(*Slot)
	if slot.CheckClear() {
		s.slots.MoveBack(entry)
		return slot
	}

	if pass := int(time.Now().Unix() - s.checkTime); pass >= s.checkIntv {
		entry = s.tryClearFromLast(s.slots.Length() - 1)
		if entry != nil {
			s.slots.MoveBack(entry)
			return entry.Value().(*Slot)
		}
	}

	return nil
}

func (s *Stub) tryClearFromLast(n int) *list.Entry {
	var entry, rtv *list.Entry
	var slot *Slot
	for ; n > 0; n-- {
		entry = s.slots.GetLast()
		slot = entry.Value().(*Slot)
		if slot.CheckClear() {
			s.slots.MoveFront(entry)
			rtv = entry
		}
	}
	return rtv
}




type StubGroup struct {
	slotCap int
	slotCount int
	totalCap int

	stubs *list.LinkedList
	sync.Mutex
}

type StubCh chan *Stub

func NewStubGroup(slotCap, stubCount, slotCount, checkIntv int) *StubGroup {
	group := StubGroup{
		slotCap: slotCap,
		slotCount: slotCount,
		stubs: list.New(),
	}
	for i := 0; i < stubCount; i++ {
		group.stubs.Push( NewStub(slotCap, slotCount, checkIntv) )
	}
	return &group
}

func (g *StubGroup) SlotSum() int {
	return g.slotCount * g.stubs.Length()
}

func (g *StubGroup) FindAvailableSlots(cnt int) (s []*Slot, e error) {
	if cnt > g.SlotSum() {
		return nil, errors.New("Too many slots to request")
	}

	s = make([]*Slot, 0, cnt)
	result := make(StubCh)
	conc, did, chkCnt, chkMax := stubsCheckConc, 0, 0, g.stubs.Length()
	for {
		did, cnt = g.doCheck(conc, cnt, result)
		if did <= 0 || cnt <= 0 {
			return
		}

		select {
		case stub := <- result:
			conc = 1
			chkCnt++
			if stub == nil {
				continue
			}
			slot := stub.FindAvailableSlot()
			if slot != nil {
				s = append(s, slot)
			}
			fmt.Printf("-- For Cap: %d - Left: %d, Slots: %d\n", g.slotCap, cnt, len(s))
		}

		if chkCnt >= chkMax && cnt > 0 {
			e = errors.New("No enough slots")
			return
		}
	}
}

func (g *StubGroup) doCheck(conc, total int, r StubCh) (did, left int) {
	left = total
	for {
		if did >= conc || left <= 0 {
			return
		}
		go g.getStubForCheck(r)
		did++
		left--
	}
}

func (g *StubGroup) getStubForCheck(r StubCh) {
	g.Lock()
	defer g.Unlock()
	
	entry := g.stubs.GetFirst()
	stub := entry.Value().(*Stub)
	g.stubs.MoveBack(entry)
	r <- stub
}

// func (g *StubGroup) checkStubForAvailableSlot(r chan *Slot) {
// 	g.Lock()
// 	defer g.Unlock()
	
// 	entry := g.stubs.GetFirst()
// 	stub := entry.Value().(*Stub)
// 	slot := stub.FindAvailableSlot()
// 	g.stubs.MoveBack(entry)
// 	r <- slot
// }
