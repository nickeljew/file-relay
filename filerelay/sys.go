package filerelay

import (
	"runtime"
	"sync"
	//"syscall"
	"time"
)



func timePassed(afterThan time.Time, limits time.Duration) bool {
	now := time.Now()
	diff := now.Sub(afterThan)
	return diff > limits
}




type SI struct {
	mem runtime.MemStats

	uptime time.Duration 	// time since boot
	loads [3]float64  		// 1, 5, and 15 minute load averages, see e.g. UPTIME(1)
	procs uint64        	// number of current processes
	totalRam uint64        	// total usable main memory size [kB]
	freeRam uint64        	// available memory size [kB]
	SharedRam uint64        // amount of shared memory [kB]
	bufferRam uint64        // memory used by buffers [kB]
	totalSwap uint64        // total swap space size [kB]
	freeSwap uint64        	// swap space still available [kB]
	totalHighRam uint64    	// total high memory size [kB]
	freeHighRam uint64     	// available high memory size [kB]

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


func getSysInfo() *SI {
	/*
	   // Note: uint64 is uint32 on 32 bit CPUs
	   type Sysinfo_t struct {
	   	Uptime    int64		// Seconds since boot
	   	Loads     [3]uint64	// 1, 5, and 15 minute load averages
	   	Procs     uint16	// Number of current processes
	   	Totalram  uint64	// Total usable main memory size
	   	Freeram   uint64	// Available memory size
	   	Sharedram uint64	// Amount of shared memory
	   	Bufferram uint64	// Memory used by buffers
	   	Totalswap uint64	// Total swap space size
	   	Freeswap  uint64	// swap space still available
	   	Pad       uint16
	   	Pad_cgo_0 [4]byte
	   	Totalhigh uint64	// Total high memory size
	   	Freehigh  uint64	// Available high memory size
	   	Unit      uint32	// Memory unit size in bytes
	   	X_f       [0]byte
	   	Pad_cgo_1 [4]byte	// Padding to 64 bytes
	   }
	*/

	//info := syscall.Sysinfo_t{}
	//
	//if e := syscall.Sysinfo(info); e != nil {
	//	panic("Error in syscall.Sysinfo:" + e.Error())
	//}
	//scale := 65536.0 // magic
	//
	//si.Lock()
	//
	//unit := uint64(info.Unit) * 1024 // kB
	//
	//si.uptime = time.Duration(info.Uptime) * time.Second
	//si.loads[0] = float64(info.Loads[0]) / scale
	//si.loads[1] = float64(info.Loads[1]) / scale
	//si.loads[2] = float64(info.Loads[2]) / scale
	//si.procs = uint64(info.Procs)
	//
	//si.totalRam = uint64(info.Totalram) / unit
	//si.freeRam = uint64(info.Freeram) / unit
	//si.bufferRam = uint64(info.Bufferram) / unit
	//si.totalSwap = uint64(info.Totalswap) / unit
	//si.freeSwap = uint64(info.Freeswap) / unit
	//si.totalHighRam = uint64(info.Totalhigh) / unit
	//si.freeHighRam = uint64(info.Freehigh) / unit
	//
	//si.Unlock()

	return si
}
