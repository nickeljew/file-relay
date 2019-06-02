package filerelay

import (
	"time"
	"sync"

	"github.com/nickeljew/file-relay/list"
)


type Stub struct {
	slotCap int
	slots *list.LinkedList
	checkTime int64
	checkIntv int
	sync.Mutex
}

func NewStub(cap, slotCount, checkIntv, minExpiration int) *Stub {
	if (checkIntv < StubCheckInterval) {
		checkIntv = StubCheckInterval
	}
	stub := Stub{
		slotCap: cap,
		slots: list.New(),
		checkTime: time.Now().Unix(),
		checkIntv: checkIntv,
	}
	for i := 0; i < slotCount; i++ {
		stub.slots.Push( NewSlot(cap, minExpiration) )
	}
	return &stub
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
	stubs *list.LinkedList
}

func NewStubGroup(cap, stubCount, slotCount, checkIntv, minExpiration int) *StubGroup {
	group := StubGroup{
		slotCap: cap,
		stubs: list.New(),
	}
	for i := 0; i < stubCount; i++ {
		group.stubs.Push( NewStub(cap, slotCount, checkIntv, minExpiration) )
	}
	return &group
}
