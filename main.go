package main

import (
	"fmt"
	"os"

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


//
func main() {
	fmt.Printf("File-Relay server is running...\n\n")
	os.Exit(filerelay.Start(defaultConfig))

}
