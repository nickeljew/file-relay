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


// Error for paring MsgLine
type MsgLineError struct {
	field string
	info string
}

func (e *MsgLineError) Error() string {
	arr := []string{"Parsing fail at field '", e.field, "'",}
	if e.info != "" {
		arr = append(arr, ": ", e.info)
	}
	return strings.Join(arr, "")
}



type MsgLine struct {
	Cmd string

	// Key is the MsgLine's key (250 bytes maximum).
	Key string

	// Flags are server-opaque flags whose semantics are entirely
	// up to the app.
	Flags uint32

	// Expiration is the cache expiration time, in seconds: either a relative
	// time from now (up to 1 month), or an absolute Unix epoch time.
	// Zero means the MsgLine has no expiration time.
	Expiration int64//time.Duration

	ValueLen uint64

	// Compare and swap ID.
	CasId uint64
}

func (ml *MsgLine) String() string {
	arr := []string{
		"&{ ",
		"Cmd:", ml.Cmd, ", ",
		"Key:", ml.Key, ", ",
		"Flags:", strconv.FormatUint(uint64(ml.Flags), 10), ", ",
		"Expiration:", strconv.FormatInt(ml.Expiration, 10), ", ",
		"ValueLen:", strconv.FormatInt(int64(ml.ValueLen), 10), ", ",
		"}",
	}
	return strings.Join(arr, "")
}

//
func (ml *MsgLine) parseLine(line []byte) {
	parts := strings.Split(strings.Trim(string(line), " \r\n"), " ")

	if ml.Cmd = parts[0]; ml.Cmd == "" {
		return
	}
	
	var err error
	if parts, err = ml.handleStoreCmdParts(parts[1:]); err != nil {
		return
	}
	if ml.Cmd == "cas" && len(parts) > 0 {
		if parts, err = ml.handleCasCmdParts(parts[1:]); err != nil {
			return
		}
	}

}

func (ml *MsgLine) handleStoreCmdParts(parts []string) ([]string, error) {
	i := 0
	if ml.Key = parts[i]; !ValidKey(ml.Key) {
		return nil, &MsgLineError{"key", ""}
	}
	i++

	if len(parts[i:]) == 0 {
		return parts[i:], nil
	}

	if d, e := strconv.ParseInt(parts[i], 10, 32); e == nil && d >= 0 {
		ml.Flags = uint32(d)
	} else if e != nil {
		return nil, e
	} else {
		return nil, &MsgLineError{"flags", "negative number"}
	}
	i++

	if d, e := strconv.ParseInt(parts[i], 10, 32); e == nil {
		ml.Expiration = d
	} else {
		return nil, e
	}
	i++

	if d, e := strconv.ParseUint(parts[i], 10, 64); e == nil && d >= 0 {
		ml.ValueLen = d
	} else if e != nil {
		return nil, e
	} else {
		return nil, &MsgLineError{"bytes len", "negative number"}
	}
	i++

	return parts[i:], nil
}

func (ml *MsgLine) handleCasCmdParts(parts []string) ([]string, error) {
	i := 0
	if d, e := strconv.ParseInt(parts[i], 10, 64); e == nil && d >= 0 {
		ml.CasId = uint64(d)
	} else if e != nil {
		return nil, e
	} else {
		return nil, &MsgLineError{"cas unique", "negative number"}
	}
	i++
	return parts[i:], nil
}

//
func ValidKey(key string) bool {
	if l := len(key); l == 0 || l > KeyMax {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] <= ' ' || key[i] == 0x7f {
			return false
		}
	}
	return true
}
