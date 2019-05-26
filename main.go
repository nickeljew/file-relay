package main

import (
	"os"
	"fmt"

	"github.com/nickeljew/file-relay/filerelay"
)

//
func main() {
	fmt.Println("File-Relay server is running...")
	//os.Exit(0)
	os.Exit(filerelay.Start())
}
