package list

import (
	"errors"
)

type (
	Entry struct {
		prev    *Entry
		next    *Entry
		element interface{}
	}
	LinkedList struct {
		element *Entry
		length  int
	}
)

var (
	ErrIndexOutOfBound = errors.New("the location out of index range")
)


func New() *LinkedList {
	return &LinkedList{
		element: nil,
	}
}

func (ll *LinkedList) Length() int {
	return ll.length
}

func (ll *LinkedList) Add(location int, v ...interface{}) error {
	length := ll.length
	if location < 0 || location > length {
		return ErrIndexOutOfBound
	}
	elements := ll.newElement(v...)
	prev, next, err := ll.getNodeAtInsertLocation(location)
	if err != nil {
		return err
	}
	first := elements[0]
	if length == 0 {
		ll.element = first
	}
	last := elements[len(elements)-1]
	first.prev = prev
	last.next = next
	if next != nil {
		next.prev = last
	}
	if prev != nil {
		prev.next = first
	}
	if location == 0 {
		ll.element = last
	}
	ll.length += len(elements)
	return nil
}

func (ll *LinkedList) AddFirst(v interface{}) error {
	return ll.Add(0, v)
}

func (ll *LinkedList) Clear() {
	ll.element = nil
	ll.length = 0
}

func (ll *LinkedList) Contains(v interface{}) bool {
	var contain = false
	ll.Iterator(func(index int, entry *Entry) bool {
		if entry.element == v {
			contain = true
			return true
		}
		return false
	})
	return contain
}

func (ll *LinkedList) Get(location int) (*Entry) {
	if location < 0 || location >= ll.length {
		return nil
	}
	var result *Entry
	ll.Iterator(func(index int, entry *Entry) bool {
		if index == location {
			result = entry
			return true
		}
		return false
	})
	return result
}

func (ll *LinkedList) GetFirst() (*Entry) {
	return ll.Get(0)
}

func (ll *LinkedList) GetLast() (*Entry) {
	return ll.Get(ll.Length() - 1)
}

func (ll *LinkedList) Pull() error {
	return ll.Remove(0)
}

func (ll *LinkedList) IndexOf(v interface{}) int {
	var result int
	ll.Iterator(func(index int, entry *Entry) bool {
		if entry.element == v {
			result = index
			return true
		}
		return false
	})
	return result
}

func (ll *LinkedList) Pop() error {
	return ll.Remove(ll.Length() - 1)
}

func (ll *LinkedList) Push(v ...interface{}) error {
	if ll.Length() == 0 {
		return ll.Add(0, v...)
	} else {
		return ll.Add(ll.length, v...)
	}
}

func (ll *LinkedList) Remove(location int) error {
	if location < 0 || location >= ll.length {
		return ErrIndexOutOfBound
	}
	if ll.length == 0 {
		return nil
	}
	if ll.length == 1 {
		ll.Clear()
	}
	var resErr error
	ll.Iterator(func(index int, entry *Entry) bool {
		if index == location {
			if location == 0 {
				el := ll.Get(1)
				el.prev = nil
				ll.element = el
			} else if location == ll.length-1 {
				el := ll.Get(location - 1)
				el.next = nil
			} else {
				prev := ll.Get(location - 1)
				next := ll.Get(location + 1)
				prev.next = next
				next.prev = prev
			}
			ll.length -= 1
			return true
		}
		return false
	})
	return resErr
}

func (ll *LinkedList) Set(locate int, v interface{}) {
	ll.Iterator(func(index int, entry *Entry) bool {
		if index == locate {
			entry.element = v
			return true
		}
		return false
	})
}

func (ll *LinkedList) Iterator(iterator func(index int, entry *Entry) bool) {
	if ll.Length() == 0 {
		return
	}
	var index = 0
	entry := ll.element
	for {
		con := iterator(index, entry)
		if con {
			break
		}
		if entry.HasNext() {
			index++
			entry = entry.Next()
		} else {
			break
		}
	}
}

func (ll *LinkedList) MoveTo(location int, entry *Entry) error {
	if entry == nil {
		return nil
	}
	if location < 0 || location >= ll.length {
		return ErrIndexOutOfBound
	}
	v := entry.element
	ll.Iterator(func(index int, e *Entry) bool {
		if entry.element == e.element {
			ll.Remove(index)
			return true
		}
		return false
	})
	ll.Add(location, v)
	return nil
}

func (ll *LinkedList) MoveFront(entry *Entry) {
	ll.MoveTo(0, entry)
}

func (ll *LinkedList) MoveBack(entry *Entry) {
	ll.MoveTo(ll.length-1, entry)
}

func (ll *LinkedList) newElement(v ...interface{}) []*Entry {
	var elements = make([]*Entry, 0)
	for index, e := range v {
		var prev *Entry
		var element Entry
		element.element = e
		if index != 0 {
			prev = elements[index-1]
			prev.next = &element
			element.prev = prev
		}
		elements = append(elements, &element)
	}
	return elements
}

// returns prev and current element
func (ll *LinkedList) getNodeAtInsertLocation(insertLocation int) (*Entry, *Entry, error) {
	if insertLocation < 0 || insertLocation > ll.Length() {
		return nil, nil, ErrIndexOutOfBound
	}
	if ll.Length() == 0 {
		return nil, nil, nil
	}
	prevIndex := insertLocation - 1
	nextIndex := insertLocation
	var prev *Entry
	var next *Entry

	ll.Iterator(func(index int, entry *Entry) bool {
		if index == prevIndex {
			prev = entry
		}
		if index == nextIndex {
			next = entry
		}
		return false
	})
	return prev, next, nil
}

func (ll *LinkedList) AsList() []interface{} {
	result := make([]interface{}, 0)
	ll.Iterator(func(index int, entry *Entry) bool {
		result = append(result, entry.element)
		return false
	})
	return result
}

func (e *Entry) HasNext() bool {
	return e.next != nil
}

func (e *Entry) Next() *Entry {
	return e.next
}

func (e *Entry) Value() interface{} {
	return e.element
}
