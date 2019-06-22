package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/sirupsen/logrus"

	"github.com/nickeljew/file-relay/filerelay"
	. "github.com/nickeljew/file-relay/debug"
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
	logger = logrus.New().WithFields(logrus.Fields{
		"name": "file-relay",
		"pkg": "main",
	})


	dtrace = NewDTrace("main")
)



type Profile struct {
	cpu string
	mem string
	cfg string
}

func getProfile() *Profile {
	var cpuprof = flag.String("cpuprofile", "", "write cpu profile to `file`")
	var memprof = flag.String("memprofile", "", "write memory profile to `file`")
	var cfg = flag.String("cfgfile", "", "yaml file to read configuration")

	flag.Parse()

	p := &Profile{
		cpu: *cpuprof,
		mem: *memprof,
		cfg: *cfg,
	}

	if p.cfg == "" {
		p.cfg = "filerelay.yaml"
	}

	return p
}


//
func main() {
	fmt.Printf("File-Relay server is running...\n\n")

	p := getProfile()

	if p.cpu != "" {
		f, err := os.Create(p.cpu)
		if err != nil {
			logger.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if e := pprof.StartCPUProfile(f); e != nil {
			logger.Fatal("could not start CPU profile: ", e)
		}
		defer pprof.StopCPUProfile()
	}

	if p.mem != "" {
		f, err := os.Create(p.mem)
		if err != nil {
			logger.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if e := pprof.WriteHeapProfile(f); e != nil {
			logger.Fatal("could not write memory profile: ", e)
		}
	}

	config := defaultConfig

	if p.cfg != "" {
		cfg, err := ioutil.ReadFile(p.cfg)
		if err != nil {
			logger.Fatal("could not read yaml configuration file: ", err)
		}
		dtrace.Logf("Configuration from file:\n- - - - - - - - - - - - -\n%s\n- - - - - - - - - - - - -", cfg)
		config = string(cfg)
	}

	code := filerelay.Start(config)

	os.Exit(code)

}

