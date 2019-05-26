package main

import (
	"os"
	"fmt"

	"github.com/nickeljew/file-relay/filerelay"
)

//
func main() {
	fmt.Println("hello world")
	//os.Exit(0)
	os.Exit(filerelay.Start())
}
