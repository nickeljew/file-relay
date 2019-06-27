package filerelay

import (
	"runtime"
	"sync"
	"time"
)



func timePassed(afterThan time.Time, limits time.Duration) bool {
	now := time.Now()
	diff := now.Sub(afterThan)
	return diff > limits
}




type SI struct {
	mem runtime.MemStats

	checkAt time.Time
	sync.Mutex
}

var si = &SI{}


func throttle() bool {
	if !timePassed(si.checkAt, time.Millisecond) {
		return true
	}
	si.checkAt = time.Now()
	return false
}


func readMemInfo() {
	si.Lock()
	if throttle() {
		si.Unlock()
		return
	}
	runtime.ReadMemStats(&si.mem)
	si.Unlock()
}

func getMemUsed() uint64 {
	readMemInfo()
	return si.mem.Sys
}

