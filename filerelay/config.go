package filerelay

const (
	PORT = "12721"
	NetType = "tcp"
)

type Config struct {
	host        string
	port        string
	networkType string

	maxRoutines int
}


func (c *Config) Addr() string {
	return c.host + ":" + c.port
}
