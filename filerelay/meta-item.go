package filerelay

import (
	linkedlist "container/list"
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
	byteLen uint64
	slots []*Slot
}

func NewMetaItem(key string, flags uint32, expiration int64, byteLen uint64) (t *MetaItem) {
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
	metaTrace.Logf(" - check item expiration -> setAt: %v | now: %v | diff: %v", t.setAt, now, diff)
	return diff > t.duration
}




type LRU struct {
	size int
	queue *linkedlist.List
	lookup *skiplist.SkipList
}

func NewLRU(size int) *LRU {
	return &LRU{
		size: size,
		queue: linkedlist.New(),
		lookup: skiplist.New(skiplist.String),
	}
}

func (c *LRU) removeOldest(clear bool) (t *MetaItem) {
	if e := c.queue.Back(); e != nil {
		t = e.Value.(*MetaItem)
		c.queue.Remove(e)

		if clear == true {
			t.ClearSlots()
		}
	}
	return
}

func (c *LRU) removeElement(elem *linkedlist.Element, clear bool) *MetaItem {
	t := elem.Value.(*MetaItem)
	c.queue.Remove(elem)

	if t != nil && clear == true {
		t.ClearSlots()
	}
	return t
}

func (c *LRU) Remove(key string) *MetaItem {
	if el := c.lookup.Remove(key); el != nil {
		elem := el.Value.(*linkedlist.Element)
		return c.removeElement(elem, true)
	}
	return nil
}

func (c *LRU) Purge() {
	c.queue.Init()
	c.lookup.Init()
}

func (c *LRU) Len() int {
	return c.lookup.Len()
}

func (c *LRU) Add(t *MetaItem, noReplace bool) (elem *linkedlist.Element, exceed bool, err error) {
	// Check for existing item
	if el := c.lookup.Get(t.key); el != nil {
		if noReplace {
			err = errors.New("key already exists")
			return
		}
		elem = el.Value.(*linkedlist.Element)
		c.queue.MoveToFront(elem)
		elem.Value = t
		return
	}

	elem = c.queue.PushFront(t)
	if el := c.lookup.Set(t.key, elem); el == nil {
		err = errors.New("failed to add item into skip-list")
		return
	}

	if exceed = c.queue.Len() > c.size; exceed {
		t2 := c.removeOldest(true)
		if t2 != nil {
			c.lookup.Remove(t2.key)
		}
	}
	return
}

func (c *LRU) Replace(t *MetaItem) *linkedlist.Element {
	if el := c.lookup.Get(t.key); el != nil {
		elem := el.Value.(*linkedlist.Element)
		c.queue.MoveToFront(elem)
		elem.Value = t
		return elem
	}
	return nil
}

func (c *LRU) Get(key string) *MetaItem {
	if el := c.lookup.Get(key); el != nil {
		elem := el.Value.(*linkedlist.Element)
		c.queue.MoveToFront(elem)
		return elem.Value.(*MetaItem)
	}
	return nil
}

func (c *LRU) GetOldest() *MetaItem {
	if elem := c.queue.Back(); elem != nil {
		return elem.Value.(*MetaItem)
	}
	return nil
}

// Peek returns the key value without updating recently-used-ness
func (c *LRU) Peek(key string) *MetaItem {
	if el := c.lookup.Get(key); el != nil {
		elem := el.Value.(*linkedlist.Element)
		return elem.Value.(*MetaItem)
	}
	return nil
}

func (c *LRU) Contains(key string) bool {
	if el := c.lookup.Get(key); el != nil {
		return true
	}
	return false
}




//
type ItemsEntry struct {
	lru *LRU
	checkpoint *skiplist.Element
	checkAt time.Time
	checkSteps int
	quit chan byte
	sync.Mutex
}

func NewItemsEntry(lruSize, checkSteps int) *ItemsEntry {
	return &ItemsEntry{
		lru: NewLRU(lruSize),
		checkSteps: checkSteps,
		quit: make(chan byte),
	}
}


func (e *ItemsEntry) StartCheck() {
	go func() {
		t := time.NewTicker(time.Second * 60)
		for {
			select {
			case <-t.C:
				e.ScheduledCheck()
			case <- e.quit:
				t.Stop()
				logger.Info("Quit ItemsEntry")
				return
			}
		}
	}()
}

func (e *ItemsEntry) StopCheck() {
	e.quit <- 0
}

func (e *ItemsEntry) ScheduledCheck() {
	listLen := e.lru.Len()
	if listLen == 0 {
		return
	}

	if e.checkpoint == nil {
		e.checkpoint = e.lru.lookup.Front()
	}

	e.Lock()
	defer e.Unlock()

	steps := e.checkSteps
	if listLen < steps {
		metaTrace.Log("ItemsEntry len: ", listLen)
		steps = listLen
	}

	for {
		next := e.checkpoint.Next()
		if next == nil {
			next = e.lru.lookup.Front()
		}

		elem := e.checkpoint.Value.(*linkedlist.Element)
		if elem != nil {
			item := elem.Value.(*MetaItem)
			metaTrace.Logf("ItemsEntry check steps: %d; key: %s", steps, item.key)

			if item != nil && item.Expired() {
				_ = e.lru.Remove(item.key)
			}
		}

		e.checkpoint = next
		steps--
		if steps == 0 {
			e.checkAt = time.Now()
			return
		}
	}

}


func (e *ItemsEntry) movePoint(key string) {
	if e.checkpoint == nil {
		return
	}

	k := e.checkpoint.Key().(string)
	metaTrace.Log("Current checkpoint key: ", k)
	if k == key {
		e.checkpoint = e.checkpoint.Next()
		if e.checkpoint == nil {
			metaTrace.Log("After removed, checkpoint is nil")
		} else {
			metaTrace.Log("After removed, checkpoint key: ", e.checkpoint.Key().(string))
		}
	}
}


func (e *ItemsEntry) Get(key string) *MetaItem {
	t := e.lru.Get(key)
	if t != nil && t.Expired() {
		e.Lock()
		defer e.Unlock()

		_ = e.lru.Remove(key)
		e.movePoint(key)
		return nil
	}
	return t
}


func (e *ItemsEntry) Remove(key string) *MetaItem {
	e.Lock()
	defer e.Unlock()

	t := e.lru.Remove(key)
	e.movePoint(key)
	return t
}


func (e *ItemsEntry) Set(t *MetaItem) error {
	e.Lock()
	defer e.Unlock()

	if _, _, err := e.lru.Add(t, false); err != nil {
		return err
	}
	return nil
}


func (e *ItemsEntry) Add(t *MetaItem) error {
	e.Lock()
	defer e.Unlock()

	if _, _, err := e.lru.Add(t, true); err != nil {
		return err
	}
	return nil
}


func (e *ItemsEntry) Replace(t *MetaItem) error {
	e.Lock()
	defer e.Unlock()

	if elem := e.lru.Replace(t); elem != nil {
		return nil
	}
	return errors.New("key not exists")
}
