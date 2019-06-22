package filerelay

import (
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	. "github.com/nickeljew/file-relay/debug"
)


var (
	logger = logrus.New()
	log = logger.WithFields(logrus.Fields{
		"name": "file-relay",
		"pkg": "filerelay",
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
func ParseConfig(rawCfg string) (*MemConfig, error) {
	cfg := NewMemConfig()

	if e := yaml.Unmarshal([]byte(rawCfg), cfg); e != nil {
		return nil, e
	}
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
func Start(rawCfg string) int {
	cfg, err := ParseConfig(rawCfg)
	if err != nil {
		log.Error("Error parsing config: ", err.Error())
		return 1
	}
	Debugf("## Init with config: %v", cfg)

	lis, err2 := net.Listen(cfg.NetworkType, cfg.Addr())
	if err2 != nil {
		log.Error("Error listening: ", err2.Error())
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

