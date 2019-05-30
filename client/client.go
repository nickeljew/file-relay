package main

import (
	"os"
	"fmt"
	"math/rand"
	"time"
	"net"
	"bufio"
	"strconv"
	"errors"
	"bytes"
	"strings"

	"github.com/nickeljew/file-relay/filerelay"
)


var (
	ErrNoServer = errors.New("No server")

	// ErrCacheMiss means that a Get failed because the data-key wasn't present.
	ErrCacheMiss = errors.New("Cache miss")

	// ErrCASConflict means that a CompareAndSwap call failed due to the
	// cached value being modified between the Get and the CompareAndSwap.
	// If the cached value was simply evicted rather than replaced,
	// ErrNotStored will be returned instead.
	ErrCASConflict = errors.New("Compare-and-swap conflict")

	// ErrNotStored means that a conditional write operation (i.e. Add or
	// CompareAndSwap) failed because the condition was not satisfied.
	ErrNotStored = errors.New("Data not stored")

	// ErrServer means that a server error occurred.
	ErrServerError = errors.New("Server error")

	// ErrMalformedKey is returned when an invalid key is used.
	// Keys must be at maximum 250 bytes long and not
	// contain whitespace or control characters.
	ErrMalformedKey = errors.New("malformed: key is too long or contains invalid characters")
)


//
func main() {
	fmt.Println("File-Relay client")

	rand.Seed(time.Now().Unix())

	cnt := 1
	fin := make(chan int)

	for i := 0; i < cnt; i++ {
		go doTry(i, fin)
	}
	
	for {
		select {
		case <- fin:
			cnt--
			fmt.Println("- left count: ", cnt)
			if cnt == 0 {
				os.Exit(0)
			}
		}
	}
}


//
type Client struct {
	nc net.Conn
	rw *bufio.ReadWriter
}

func doTry(idx int, fin chan int) {
	if err := trySet(); err != nil {
		fmt.Printf("Error in %d: %s\n", idx, err.Error())
	} else {
		fmt.Printf("Finish %d\n", idx)
	}
	fin <- idx
}

//
func trySet() error {
	cfg, _ := filerelay.InitClientConfig("localhost")
	nc, err := net.Dial("tcp", cfg.Addr())
	if err != nil {
		fmt.Println("Failed connection to: ", cfg.Addr())
		return ErrNoServer
	}

	// reader := bufio.NewReader(os.Stdin)
    // fmt.Print("Text to send: ")
	// text, _ := reader.ReadString('\n')

	client := &Client{
		nc: nc,
		rw: bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc)),
	}

	reqValue := []byte("Hello World - " + strconv.Itoa(random(10, 50)))
	reqline := &filerelay.ReqLine{
		Cmd: "set",
		Key: "test123",
		ValueLen: uint64(len(reqValue)),
		Flags: 1,
		Expiration: 1800,
	}

	// if !ValidKey(reqline.Key) {
	// 	return ErrMalformedKey
	// }

	toSend := fmt.Sprintf("%s %s %d %d %d\r\n",
	reqline.Cmd, reqline.Key, reqline.Flags, reqline.Expiration, reqline.ValueLen)
	fmt.Printf("Sending:\n>%s", toSend)
	if _, err = fmt.Fprintf(client.rw, toSend); err != nil {
		return err
	}

	if _, err = client.rw.Write(reqValue); err != nil {
		return err
	}
	if _, err := client.rw.Write(filerelay.Crlf); err != nil {
		return err
	}
	if err := client.rw.Flush(); err != nil {
		return err
	}

	line, err := client.rw.ReadSlice('\n')
	if err != nil {
		return err
	}
	fmt.Println("Response from server:", strings.Trim(string(line), " \r\n"))

	switch {
	case bytes.Equal(line, filerelay.ResultStored):
		return nil
	case bytes.Equal(line, filerelay.ResultNotStored):
		return ErrNotStored
	case bytes.Equal(line, filerelay.ResultExists):
		return ErrCASConflict
	case bytes.Equal(line, filerelay.ResultNotFound):
		return ErrCacheMiss
	}

	return fmt.Errorf("Unexpected response line from %q: %q", reqline.Cmd, string(line))
}


func random(min, max int) int {
	return rand.Intn(max-min) + min
}
