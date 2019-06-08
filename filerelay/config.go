package filerelay

import (
	"math/rand"
	"strconv"
	"time"
)

const (
	PORT = "12721"
	NetType = "tcp"

	KeyMax = 250
)


func init() {
	rand.Seed(time.Now().Unix())
}


type Config struct {
	host        string
	port        string
	networkType string

	maxRoutines int
}


func (c *Config) Addr() string {
	return c.host + ":" + c.port
}


func RandomNum(min, max int) int {
	return rand.Intn(max-min) + min
}

func RandomStr(min, max int) string {
	return strconv.Itoa(RandomNum(min, max))
}
