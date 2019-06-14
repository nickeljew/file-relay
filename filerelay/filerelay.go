package filerelay

import (
	"net"
	"os"

	"github.com/sirupsen/logrus"
)


var (
	logger = logrus.New()
	log = logger.WithFields(logrus.Fields{
		"name": "filerelay",
	})
)


//
func init() {
	// Log as JSON instead of the default ASCII formatter.
	logger.SetFormatter(&logrus.JSONFormatter{})
  
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logger.SetOutput(os.Stdout)
  
	// Only log the warning severity or above.
	logger.SetLevel(logrus.InfoLevel)
}



//
// func init() {
// 	//
// }

func InitConfig() (*MemConfig, error) {
	//TODO: read config from yaml file
	cfg := NewMemConfig()
	cfg.Port = PORT
	cfg.NetworkType = NetType
	cfg.MaxRoutines = 2
	return cfg, nil
}

func InitClientConfig(host string) (*Config, error) {
	//TODO: read config from yaml file
	return &Config{
		Host: host,
		Port: PORT,
		NetworkType: NetType,
	}, nil
}


//
func Start() int {
	cfg, _ := InitConfig()
	lis, err := net.Listen(cfg.NetworkType, cfg.Addr())
	if err != nil {
		log.Error("Error listening: ", err.Error())
		return 1
	}
	log.Info("Server is listening at: ", cfg.Addr())
	defer lis.Close()

	server := NewServer(cfg)
	
	go server.Start()
	defer server.Stop()

	for {
        // Listen for an incoming connection.
        conn, err := lis.Accept()
        if err != nil {
            log.Error("Error accepting: ", err.Error())
            return 1
        }
		// Handle connections in a new goroutine.
		log.Info("# New incoming connection")
        go server.Handle(conn)
    }

	//return 0
}

