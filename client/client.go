package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nickeljew/file-relay/filerelay"
)


var (
	ErrNoServer = errors.New("no server")

	// ErrCacheMiss means that a Get failed because the data-key wasn't present.
	ErrCacheMiss = errors.New("cache miss")

	// ErrCASConflict means that a CompareAndSwap call failed due to the
	// cached value being modified between the Get and the CompareAndSwap.
	// If the cached value was simply evicted rather than replaced,
	// ErrNotStored will be returned instead.
	ErrCASConflict = errors.New("compare-and-swap conflict")

	// ErrNotStored means that a conditional write operation (i.e. Add or
	// CompareAndSwap) failed because the condition was not satisfied.
	ErrNotStored = errors.New("data not stored")

	// ErrServer means that a server error occurred.
	ErrServerError = errors.New("server error")

	// ErrMalformedKey is returned when an invalid key is used.
	// Keys must be at maximum 250 bytes long and not
	// contain whitespace or control characters.
	ErrMalformedKey = errors.New("malformed: key is too long or contains invalid characters")
)


//
func main() {
	fmt.Println("File-Relay client")

	rand.Seed(time.Now().Unix())

	doConcurrentSet(1)
	
	os.Exit(0)
}


//
type Client struct {
	nc net.Conn
	rw *bufio.ReadWriter
}


func doConcurrentSet(cnt int) {
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
				return
			}
		}
	}
}


func doTry(idx int, fin chan int) {
	fmt.Println("Doing at index: ", idx)
	client, err := setupClient()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := trySet(client); err != nil {
		fmt.Printf("Error in %d: %s\n", idx, err.Error())
	} else {
		fmt.Printf("Finish %d\n", idx)
	}
	fin <- idx
}

func setupClient() (*Client, error) {
	cfg, _ := filerelay.InitClientConfig("localhost")
	nc, err := net.Dial("tcp", cfg.Addr())
	if err != nil {
		fmt.Println("Failed connection to: ", cfg.Addr())
		return nil, ErrNoServer
	}

	return &Client{
		nc: nc,
		rw: bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc)),
	}, nil
}

//
func trySet(client *Client) error {
	reqValue, err := ioutil.ReadFile("./test.txt")
	reqline := &filerelay.ReqLine{
		Cmd: "set",
		Key: "test123" + strconv.Itoa(random(10, 50)),
		ValueLen: len(reqValue),
		Flags: 1,
		Expiration: 1800,
	}

	toSend := fmt.Sprintf("%s %s %d %d %d\r\n",
	reqline.Cmd, reqline.Key, reqline.Flags, reqline.Expiration, reqline.ValueLen)
	fmt.Printf("Sending:\n>%s", toSend)
	if _, err := fmt.Fprintf(client.rw, toSend); err != nil {
		return err
	}

	if _, err := client.rw.Write(reqValue); err != nil {
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
