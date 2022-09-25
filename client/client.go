package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"crypto/md5"
	"hash/fnv"

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

const (
	MaxFileSize = 1024 * 1024 * 10 //10MB
)



type TrialKeyMap struct {
	keys map[string]bool
	sync.Mutex
}
var trialKeyMap = TrialKeyMap{
	keys: make(map[string]bool),
}


//
func main() {
	fmt.Println("File-Relay client *", time.Now())

	//doConcurrentSet(101, "set", "")
	//doConcurrentSet(101, "add", "./files")

	//doAddAndReplace()

	//doSetNGet("test-abc-set-and-get", false)

	data := []byte("These pretzels are making me thirsty.")
	hash := md5.Sum(data)
	fmt.Printf("Hash %T:\n%x\n%v\n\n", hash, hash, hash)

	var hSum uint64
	for _, h := range hash {
		m := hSum << 2
		hSum = m + uint64(h)
		fmt.Printf("----\nSum: %v %v - mod 1024: %v\n", m, hSum, hSum % 1024)
	}

	hashing2 := fnv.New32a()
	hash2 := hashing2.Sum(data)
	fmt.Printf("Hash2 %T:\n%x\n%v\n\n", hash2, hash2, hash2)
	
	os.Exit(0)
}


//
type Client struct {
	nc net.Conn
	rw *bufio.ReadWriter
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


func doConcurrentSet(cnt int, cmd, dirPath string) {
	fin := make(chan int)

	count := 0
	if dirPath == "" {
		for i := 0; i < cnt; i++ {
			count++
			go doStorageInIndex(i, fin, cmd, "", "")
		}
	} else {
		handleFile := func(i int, filepath string) bool {
			fmt.Println("--> File path: ", filepath)
			count++
			go doStorageInIndex(i, fin, cmd, filepath, filepath)
			return true
		}
		readFilesFromDir(dirPath, cnt, handleFile)
	}
	
	for {
		select {
		case <- fin:
			count--
			fmt.Println("- left count: ", count)
			if count == 0 {
				return
			}
		}
	}
}


func doStorageInIndex(idx int, fin chan int, cmd, key, filepath string) {
	fmt.Println("Doing at index: ", idx)
	if err := tryStorage(idx, cmd, key, filepath); err != nil {
		fmt.Printf("Error in %d: %s\n", idx, err.Error())
	} else {
		fmt.Printf("Finish %d\n", idx)
	}
	fin <- idx
}




func doAddAndReplace() {
	key := "add-replace-key"
	filepath1 := "./files/m_1390221121382.jpg"
	filepath2 := "./files/m_1390221149351.jpg"

	if err := tryStorage(0, "add", key, filepath1); err != nil {
		fmt.Printf("Error in adding: %s\n", err.Error())
	} else {
		fmt.Println("Finish adding")
	}

	if e := tryGet(key); e != nil {
		fmt.Printf("Error: %s\n", e.Error())
	}

	if err := tryStorage(0, "replace", key, filepath2); err != nil {
		fmt.Printf("Error in adding: %s\n", err.Error())
	} else {
		fmt.Println("Finish adding")
	}

	if e := tryGet(key); e != nil {
		fmt.Printf("Error: %s\n", e.Error())
	}
}





func doSetNGet(key string, onlyGet bool) {
	if !onlyGet {
		fmt.Println("# Doing set with key: ", key)
		if e := tryStorage(0, "set", key, ""); e != nil {
			fmt.Printf("Error: %s\n", e.Error())
		} else {
			fmt.Println("Finish")
		}
	}

	fmt.Println("# Doing get with key: ", key)
	if e := tryGet(key); e != nil {
		fmt.Printf("Error: %s\n", e.Error())
	}
}

func createKey() string {
	var key string
	for {
		key = "test123" + filerelay.RandomStr(1000, 9999)
		fmt.Println("Trying created key: ", key)
		trialKeyMap.Lock()
		if !trialKeyMap.keys[key] {
			trialKeyMap.keys[key] = true
			trialKeyMap.Unlock()
			break
		}
		trialKeyMap.Unlock()
	}
	return key
}

//
func tryStorage(tryIndex int, cmd, key, filepath string) error {
	client, err := setupClient()
	if err != nil {
		fmt.Printf("Error in setup client for key[%s]: %v", key, err.Error())
		return nil
	}
	defer client.nc.Close()

	if filepath == "" {
		filepath = "./test.txt"
	}
	reqValue, err := ioutil.ReadFile(filepath)
	key = strings.Trim(key, " \f\n\r\t\v")
	if key == "" {
		key = createKey()
	}

	msgline := &filerelay.MsgLine{
		Cmd: cmd,
		Key: key,
		ValueLen: uint64( len(reqValue) ),
		Flags: 1,
		Expiration: 120,
	}

	toSend := fmt.Sprintf("%s %s %d %d %d\r\n",
		msgline.Cmd, msgline.Key, msgline.Flags, msgline.Expiration, msgline.ValueLen)
	fmt.Printf("Sending[%d]:\n>%s", tryIndex, toSend)
	//if _, err := fmt.Fprintf(client.rw, toSend); err != nil {
	//	return err
	//}
	if _, err := client.rw.Write([]byte(toSend)); err != nil {
		fmt.Printf("Error in writing storage cmd-line for key[%s]: %v\n", msgline.Key, err)
		return err
	}

	if _, err := client.rw.Write(reqValue); err != nil {
		fmt.Printf("Error in writing value for key[%s]: %v\n", msgline.Key, err)
		return err
	}
	if _, err := client.rw.Write(filerelay.Crlf); err != nil {
		fmt.Printf("Error in writing END for key[%s]: %v\n", msgline.Key, err)
		return err
	}
	if err := client.rw.Flush(); err != nil {
		fmt.Printf("Error in flushing for key[%s]: %v\n", msgline.Key, err)
		return err
	}

	line, err := client.rw.ReadSlice('\n')
	if err != nil {
		fmt.Printf("Error in read for server for key[%s]: %v\n", msgline.Key, err)
		return err
	}
	fmt.Printf("Response from server[%d] for key[%s]: %s\n", tryIndex, key, strings.Trim(string(line)," \r\n"))

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

	return fmt.Errorf("unexpected response line from %q: %q", msgline.Cmd, string(line))
}




//
func tryGet(key string) error {
	client, err := setupClient()
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	defer client.nc.Close()

	toSend := fmt.Sprintf("get %s\r\n", key)
	fmt.Printf("Sending:\n>%s", toSend)
	if _, err := client.rw.Write([]byte(toSend)); err != nil {
		fmt.Printf("Error in writing retrieval cmd-line for key[%s]: %v\n", key, err)
		return err
	}
	if err := client.rw.Flush(); err != nil {
		fmt.Printf("Error in flushing for key[%s]: %v\n", key, err)
		return err
	}

	line, err := client.rw.ReadSlice('\n')
	if err != nil {
		return err
	}
	fmt.Println("Response from server:", strings.Trim(string(line)," \r\n"))
	//fmt.Println("Response from server:", strings.ReplaceAll(string(line), "\\", "\\\\"))

	ml := &filerelay.MsgLine{}
	pattern := "VALUE %s %d %d %d\r\n"
	dest := []interface{}{&ml.Key, &ml.Flags, &ml.ValueLen, &ml.CasId}
	if bytes.Count(line, filerelay.Space) == 3 {
		pattern = "VALUE %s %d %d\r\n"
		dest = dest[:3]
	}
	if n, e := fmt.Sscanf(string(line), pattern, dest...); e != nil || n != len(dest) {
		return fmt.Errorf("unexpected line in get response: %q", line)
	}

	if ml.ValueLen == 0 {
		return nil
	}

	itemValue := make([]byte, ml.ValueLen + 2)
	if _, e := io.ReadFull(client.rw, itemValue); e != nil {
		client.rw = nil
		return e
	}
	if !bytes.HasSuffix(itemValue, filerelay.Crlf) {
		itemValue = nil
		return fmt.Errorf("corrupt get result read")
	}
	itemValue = itemValue[:ml.ValueLen]
	fmt.Printf("Value:\n%d\n-- END --\n", len(itemValue))
	return nil
}



func readFilesFromDir(dirPath string, count int, handleFile func(idx int, filepath string) bool) {
	dir, err := ioutil.ReadDir(dirPath)
	if err != nil {
		fmt.Println("Error in reading directory")
		return
	}

	sep := string(os.PathSeparator)
	for i, file := range dir {
		if count > 0 && i >= count {
			break
		}
		if !file.IsDir() && file.Size() <= MaxFileSize {
			if !handleFile(i, dirPath + sep + file.Name()) {
				break
			}
		}
	}
}
