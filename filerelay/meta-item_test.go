package filerelay

import (
	"strconv"
	"testing"
)

const (
	itemCount = 11
)

var itemsEntry *ItemsEntry

func init() {
	itemsEntry = NewItemsEntry(2)
}



func newItem(i int, t *testing.T) *MetaItem {
	key := "HelloTest-" + strconv.Itoa(i) //RandomStr(10, 50)
	t.Log("Creating new item with key: ", key)
	return NewMetaItem(key, 0, 1800, 0)
}


func TestItemsEntry_Add(t *testing.T) {
	for i := 0; i < itemCount; i++ {
		_ = itemsEntry.Add(newItem(i, t))
	}
}

func TestItemsEntry_Remove(t *testing.T) {
	front := itemsEntry.list.Front()
	frontKey := front.Key().(string)
	t.Log("Front element key: ", frontKey)

	itemsEntry.checkpoint = front
	item := itemsEntry.checkpoint.Value.(*MetaItem)

	item2 := itemsEntry.Remove(item.key)
	t.Log("Current checkpoint key: ", itemsEntry.checkpoint.Key().(string), "; item: ", item2)
}

// to ensure the checking will go through all items in skip-list
func TestItemsEntry_ScheduledCheck(t *testing.T) {
	itemsEntry.ScheduledCheck()
	itemsEntry.ScheduledCheck()
	itemsEntry.ScheduledCheck()
	itemsEntry.ScheduledCheck()
	itemsEntry.ScheduledCheck()
}
