package filerelay

import (
	"strconv"
	"strings"
	//"time"
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


// Error for paring ReqLine
type ReqLineError struct {
	field string
	info string
}

func (e *ReqLineError) Error() string {
	arr := []string{"Parsing fail at field '", e.field, "'",}
	if e.info != "" {
		arr = append(arr, ": ", e.info)
	}
	return strings.Join(arr, "")
}



type ReqLine struct {
	Cmd string

	// Key is the ReqLine's key (250 bytes maximum).
	Key string

	// Flags are server-opaque flags whose semantics are entirely
	// up to the app.
	Flags uint32

	// Expiration is the cache expiration time, in seconds: either a relative
	// time from now (up to 1 month), or an absolute Unix epoch time.
	// Zero means the ReqLine has no expiration time.
	Expiration int64//time.Duration

	ValueLen int

	// Compare and swap ID.
	casid uint64
}

func (rl *ReqLine) String() string {
	arr := []string{
		"&{ ",
		"Cmd:", rl.Cmd, ", ",
		"Key:", rl.Key, ", ",
		"Flags:", strconv.FormatUint(uint64(rl.Flags), 10), ", ",
		"Expiration:", strconv.FormatInt(rl.Expiration, 10), ", ",
		"ValueLen:", strconv.FormatInt(int64(rl.ValueLen), 10), ", ",
		"}",
	}
	return strings.Join(arr, "")
}

//
func (rl *ReqLine) parseLine(line []byte) {
	parts := strings.Split(strings.Trim(string(line), " \r\n"), " ")

	if rl.Cmd = parts[0]; rl.Cmd == "" {
		return
	}
	
	var err error
	if parts, err = rl.handleStoreCmdParts(parts[1:]); err != nil {
		return
	}
	if rl.Cmd == "cas" && len(parts) > 0 {
		if parts, err = rl.handleCasCmdParts(parts[1:]); err != nil {
			return
		}
	}

}

func (rl *ReqLine) handleStoreCmdParts(parts []string) ([]string, error) {
	i := 0
	if rl.Key = parts[i]; !ValidKey(rl.Key) {
		return nil, &ReqLineError{"key", ""}
	}
	i++

	if d, e := strconv.ParseInt(parts[i], 10, 32); e == nil && d >= 0 {
		rl.Flags = uint32(d)
	} else if e != nil {
		return nil, e
	} else {
		return nil, &ReqLineError{"flags", "negative number"}
	}
	i++

	if d, e := strconv.ParseInt(parts[i], 10, 32); e == nil {
		rl.Expiration = d
	} else {
		return nil, e
	}
	i++

	if d, e := strconv.ParseInt(parts[i], 10, 64); e == nil && d >= 0 {
		rl.ValueLen = int(d)
	} else if e != nil {
		return nil, e
	} else {
		return nil, &ReqLineError{"bytes len", "negative number"}
	}
	i++

	return parts[i:], nil
}

func (rl *ReqLine) handleCasCmdParts(parts []string) ([]string, error) {
	i := 0
	if d, e := strconv.ParseInt(parts[i], 10, 64); e == nil && d >= 0 {
		rl.casid = uint64(d)
	} else if e != nil {
		return nil, e
	} else {
		return nil, &ReqLineError{"cas unique", "negative number"}
	}
	i++
	return parts[i:], nil
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
