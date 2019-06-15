package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/sirupsen/logrus"

	"github.com/nickeljew/file-relay/filerelay"
)


const defaultConfig = `
# General purpose
#host: 
port: 12721
network-type: tcp
max-routines: 2

# Memory purpose
lru-size: 100000
skiplist-check-step: 20
#min-expiration: 
#slab-check-interval:
#slot-capacity-min: 64
#slot-capacity-max: 4096
#slots-in-slab: 200
#slabs-in-group: 100
`


var (
	logger = logrus.New()
	log = logger.WithFields(logrus.Fields{
		"name": "file-relay",
		"pkg": "main",
	})
)



type Profile struct {
	cpu string
	mem string
}

func getProfile() *Profile {
	var cpuprof = flag.String("cpuprofile", "", "write cpu profile to `file`")
	var memprof = flag.String("memprofile", "", "write memory profile to `file`")

	flag.Parse()

	return &Profile{
		cpu: *cpuprof,
		mem: *memprof,
	}
}


//
func main() {
	fmt.Printf("File-Relay server is running...\n\n")

	p := getProfile()

	if p.cpu != "" {
		f, err := os.Create(p.cpu)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if e := pprof.StartCPUProfile(f); e != nil {
			log.Fatal("could not start CPU profile: ", e)
		}
		defer pprof.StopCPUProfile()
	}

	if p.mem != "" {
		f, err := os.Create(p.mem)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if e := pprof.WriteHeapProfile(f); e != nil {
			log.Fatal("could not write memory profile: ", e)
		}
	}

	code := filerelay.Start(defaultConfig)

	os.Exit(code)

}

