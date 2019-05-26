package filerelay

import (
	"fmt"
	"net"
)

//
func Start() int {
	cfg, _ := InitConfig()
	addr := cfg.host + ":" + cfg.port
	lis, err := net.Listen(cfg.networkType, addr)
	if err != nil {
		fmt.Println("Error listening: ", err.Error())
		return 1
	}
	defer lis.Close()

	server := &Server{cfg.maxRoutines}
	err = server.handleConn()
	if err != nil {
		fmt.Println("Error handling: ", err.Error())
		return 1
	}

	return 0
}
