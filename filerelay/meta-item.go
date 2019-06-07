package filerelay

import (
	"errors"
	"fmt"
	"sync"
	"time"

	//"github.com/sirupsen/logrus"

	"github.com/huandu/skiplist"
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
	sync.Mutex
}

func NewItemsEntry() *ItemsEntry {
	return &ItemsEntry{
		list: skiplist.New(skiplist.String),
	}
}

func (e *ItemsEntry) get(key string) *MetaItem {
	elem := e.list.Get(key)
	if elem == nil {
		return nil
	}
	return elem.Value.(*MetaItem)
}

func (e *ItemsEntry) remove(key string, clear bool) *MetaItem {
	elem := e.list.Remove(key)
	if elem == nil {
		return nil
	}
	t := elem.Value.(*MetaItem)
	if clear == true {
		t.ClearSlots()
	}
	return t
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
		e.remove(key,true)
		return nil
	}
	return t
}


func (e *ItemsEntry) Set(t *MetaItem) error {
	if elem := e.replace(t); elem != nil {
		return nil
	} else if elem = e.list.Set(t.key, t); elem != nil {
		return nil
	}
	return errors.New("failed to set item into skip-list")
}


func (e *ItemsEntry) Add(t *MetaItem) error {
	if elem := e.list.Get(t.key); elem != nil {
		return errors.New("key already exists")
	} else if elem = e.list.Set(t.key, t); elem != nil {
		return nil
	}
	return errors.New("failed to set item into skip-list")
}


func (e *ItemsEntry) Replace(t *MetaItem) error {
	if elem := e.replace(t); elem != nil {
		return nil
	}
	return errors.New("key not exists")
}
