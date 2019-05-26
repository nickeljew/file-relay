package filerelay

import (
	"fmt"
	"net"
	//"bufio"
)


//
// func init() {
// 	//
// }

func InitConfig() (*Config, error) {
	//TODO: read config from yaml file
	return &Config{
		port:        PORT,
		networkType: NetType,

		maxRoutines: 2,
	}, nil
}

func InitClientConfig(host string) (*Config, error) {
	//TODO: read config from yaml file
	return &Config{
		host: host,
		port: PORT,
		networkType: NetType,
	}, nil
}


//
func Start() int {
	cfg, _ := InitConfig()
	lis, err := net.Listen(cfg.networkType, cfg.Addr())
	if err != nil {
		fmt.Println("Error listening: ", err.Error())
		return 1
	}
	defer lis.Close()

	server := NewServer(cfg.maxRoutines)
	
	go server.Start()
	defer server.Stop()

	for {
        // Listen for an incoming connection.
        conn, err := lis.Accept()
        if err != nil {
            fmt.Println("Error accepting: ", err.Error())
            return 1
        }
		// Handle connections in a new goroutine.
		fmt.Println("# New incoming connection")
        go server.Handle(conn)
    }

	//return 0
}

