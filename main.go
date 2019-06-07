package main

import (
	"fmt"
	"os"

	"github.com/nickeljew/file-relay/filerelay"
)


//
func main() {
	fmt.Printf("File-Relay server is running...\n\n")
	//os.Exit(0)
	os.Exit(filerelay.Start())
}
