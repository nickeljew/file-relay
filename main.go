package main

import (
	"os"
	"fmt"
	
	log "github.com/sirupsen/logrus"

	"github.com/nickeljew/file-relay/filerelay"
)

//
func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})
  
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)
  
	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)
  }


//
func main() {
	fmt.Println("File-Relay server is running...\n")
	//os.Exit(0)
	os.Exit(filerelay.Start())
}
