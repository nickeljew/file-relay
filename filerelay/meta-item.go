package filerelay

import (
	"errors"
	"fmt"
	"sync"
	"time"

	//"github.com/sirupsen/logrus"
	"github.com/huandu/skiplist"

	. "github.com/nickeljew/file-relay/debug"
)


//
type MetaItem struct {
	key string
	flags uint32
	setAt time.Time
	duration time.Duration
	byteLen int
	slots []*Slot
}

func NewMetaItem(key string, flags uint32, expiration int64, byteLen int) (t *MetaItem) {
	t = &MetaItem{
		key: key,
		flags: flags,
		setAt: time.Now(),
		byteLen: byteLen,
		//slots: make([]*Slot),
	}
	secs := fmt.Sprintf("%ds", expiration)
	t.duration, _ = time.ParseDuration(secs)
	return
}

func (t *MetaItem) ClearSlots() {
	for _, s := range t.slots {
		s.Clear()
	}
	t.slots = make([]*Slot, 0, 0)
}

func (t *MetaItem) Expired() bool {
	now := time.Now()
	diff := now.Sub(t.setAt)
	return diff > t.duration
}



//
type ItemsEntry struct {
	list *skiplist.SkipList
	checkpoint *skiplist.Element
	checkAt time.Time
	checkSteps int
	quit chan byte
	sync.Mutex
}

func NewItemsEntry(checkSteps int) *ItemsEntry {
	return &ItemsEntry{
		list: skiplist.New(skiplist.String),
		checkSteps: checkSteps,
		quit: make(chan byte),
	}
}


func (e *ItemsEntry) StartCheck() {
	t := time.NewTicker(time.Second * 60)

	for {
		select {
		case <-t.C:
			e.ScheduledCheck()
		case <- e.quit:
			log.Info("Quit ItemsEntry")
			return
		}
	}
}

func (e *ItemsEntry) StopCheck() {
	e.quit <- 0
}

func (e *ItemsEntry) ScheduledCheck() {
	if e.checkpoint == nil {
		e.checkpoint = e.list.Front()
		if e.checkpoint == nil {
			return
		}
	}

	e.Lock()
	defer e.Unlock()

	steps := e.checkSteps
	if l := e.list.Len(); l < steps {
		Debug("ItemsEntry len: ", l)
		steps = l
	}

	for {
		item := e.checkpoint.Value.(*MetaItem)
		Debugf("ItemsEntry check steps: %d; key: %s", steps, item.key)
		expired := item.Expired()

		e.checkpoint = e.checkpoint.Next()
		if e.checkpoint == nil {
			e.checkpoint = e.list.Front()
		}

		if expired {
			_ = e.remove(item.key,true)
		}

		steps--
		if steps == 0 {
			e.checkAt = time.Now()
			return
		}
	}

}


func (e *ItemsEntry) get(key string) *MetaItem {
	elem := e.list.Get(key)
	if elem == nil {
		return nil
	}
	return elem.Value.(*MetaItem)
}

func (e *ItemsEntry) remove(key string, clear bool) *skiplist.Element {
	elem := e.list.Remove(key)
	if elem == nil {
		return nil
	}
	if clear == true {
		t := elem.Value.(*MetaItem)
		t.ClearSlots()
	}
	return elem
}

func (e *ItemsEntry) replace(t *MetaItem) *skiplist.Element {
	elem := e.list.Get(t.key)
	if elem != nil {
		old := elem.Value.(*MetaItem)
		old.ClearSlots()
		elem.Value = t
	}
	return elem
}


func (e *ItemsEntry) Get(key string) *MetaItem {
	t := e.get(key)
	if t != nil && t.Expired() {
		_ = e.remove(key,true)
		return nil
	}
	return t
}


func (e *ItemsEntry) Remove(key string) *MetaItem {
	e.Lock()
	defer e.Unlock()

	elem := e.remove(key,true)
	if elem == nil {
		return nil
	}
	t := elem.Value.(*MetaItem)
	if e.checkpoint == nil {
		return t
	}

	k := e.checkpoint.Key().(string)
	Debug("Current checkpoint key: ", k)
	if k == t.key {
		e.checkpoint = e.checkpoint.Next()
		Debug("After removed, checkpoint key: ", e.checkpoint.Key().(string))
	}
	return t
}


func (e *ItemsEntry) Set(t *MetaItem) error {
	e.Lock()
	defer e.Unlock()

	if elem := e.replace(t); elem != nil {
		return nil
	} else if elem = e.list.Set(t.key, t); elem != nil {
		return nil
	}
	return errors.New("failed to set item into skip-list")
}


func (e *ItemsEntry) Add(t *MetaItem) error {
	e.Lock()
	defer e.Unlock()

	if elem := e.list.Get(t.key); elem != nil {
		return errors.New("key already exists")
	} else if elem = e.list.Set(t.key, t); elem != nil {
		return nil
	}
	return errors.New("failed to set item into skip-list")
}


func (e *ItemsEntry) Replace(t *MetaItem) error {
	e.Lock()
	defer e.Unlock()

	if elem := e.replace(t); elem != nil {
		return nil
	}
	return errors.New("key not exists")
}
