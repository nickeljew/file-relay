package main

import (
	"net"

	. "github.com/nickeljew/file-relay/debug"
)


var (
	dtrace = NewDTrace("sample")
	sthTrace = NewDTrace("sample:sth")
	memTrace = NewDTrace("sample:mem")
)


func main() {
	dtrace.Log("Start service...")

	netType := "tcp"
	port := "10238"
	lis, err := net.Listen(netType, ":"+port)
	if err == nil {
		dtrace.Logf("Listening %s connection on %s", netType, port)
	}
	defer lis.Close()

	doSth("network")
	handleMem(120)
	doSth("file")
	handleMem(300)

	dtrace.Log("Service end.")
}

func doSth(src string) {
	sthTrace.Log(" - doing something here")
	sthTrace.Logf(" recv data from %s for process", src)
}

func handleMem(sz int) {
	memTrace.Logf(" * allocating memory with size %d", sz)
	memTrace.Log(" data stored")
}


