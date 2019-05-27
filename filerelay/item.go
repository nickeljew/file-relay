package filerelay

import (
	"strconv"
	"strings"
	//"fmt"
)


var (
	Crlf            = []byte("\r\n")
	Space           = []byte(" ")
	ResultOK        = []byte("OK\r\n")
	ResultStored    = []byte("STORED\r\n")
	ResultNotStored = []byte("NOT_STORED\r\n")
	ResultExists    = []byte("EXISTS\r\n")
	ResultNotFound  = []byte("NOT_FOUND\r\n")
	ResultDeleted   = []byte("DELETED\r\n")
	ResultEnd       = []byte("END\r\n")
	ResultOk        = []byte("OK\r\n")
	ResultTouched   = []byte("TOUCHED\r\n")

	ResultClientErrorPrefix = []byte("CLIENT_ERROR ")
)



type Item struct {
	Cmd string

	// Key is the Item's key (250 bytes maximum).
	Key string

	// Value is the Item's value.
	Value []byte

	// Flags are server-opaque flags whose semantics are entirely
	// up to the app.
	Flags uint32

	// Expiration is the cache expiration time, in seconds: either a relative
	// time from now (up to 1 month), or an absolute Unix epoch time.
	// Zero means the Item has no expiration time.
	Expiration int32

	// Compare and swap ID.
	casid uint64
}

//
func (t *Item) parseLine(line []byte) {
	parts := strings.Split(strings.Trim(string(line), " \r\n"), " ")

	if t.Cmd = parts[0]; t.Cmd == "" {
		return
	}
	
	t.handleStoreCmdParts(parts[1:])
}

func (t *Item) handleStoreCmdParts(parts []string) {
	if t.Key = parts[0]; !ValidKey(t.Key) {
		return
	}

	if d, e := strconv.ParseInt(parts[1], 10, 32); e == nil && d >= 0 {
		t.Flags = uint32(d)
	}
	if d, e := strconv.ParseInt(parts[2], 10, 32); e == nil {
		t.Expiration = int32(d)
	}

	if t.Cmd == "cas" {
		if d, e := strconv.ParseInt(parts[3], 10, 64); e == nil && d >= 0 {
			t.casid = uint64(d)
		}
	}
}

//
func ValidKey(key string) bool {
	if l := len(key); l == 0 || l > 250 {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] <= ' ' || key[i] == 0x7f {
			return false
		}
	}
	return true
}
