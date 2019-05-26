package filerelay


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
func (t *Item) gen(cmd string) {
	//
}

func ValidKey(key string) bool {
	if len(key) > 250 {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] <= ' ' || key[i] == 0x7f {
			return false
		}
	}
	return true
}
