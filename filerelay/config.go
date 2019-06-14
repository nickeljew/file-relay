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
	Host        string `yaml:"host,omitempty"`
	Port        string `yaml:"port"`
	NetworkType string `yaml:"network-type"`

	MaxRoutines int `yaml:"max-routines"`
}


func (c *Config) Addr() string {
	return c.Host + ":" + c.Port
}


func RandomNum(min, max int) int {
	return rand.Intn(max-min) + min
}

func RandomStr(min, max int) string {
	return strconv.Itoa(RandomNum(min, max))
}
