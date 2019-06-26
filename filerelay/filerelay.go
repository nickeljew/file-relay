package filerelay

import (
	"math"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	. "github.com/nickeljew/file-relay/debug"
)


var (
	rootLogger = logrus.New()
	logger = rootLogger.WithFields(logrus.Fields{
		"name": "file-relay",
		"pkg": "filerelay",
	})

	dtrace = NewDTrace("filerelay")
	metaTrace = NewDTrace("filerelay:meta")
	memTrace = NewDTrace("filerelay:mem")
)


//
func init() {
	// Log as JSON instead of the default ASCII formatter.
	rootLogger.SetFormatter(&logrus.JSONFormatter{})
  
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	rootLogger.SetOutput(os.Stdout)
  
	// Only log the warning severity or above.
	rootLogger.SetLevel(logrus.InfoLevel)
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
		logger.Errorf("Error parsing config: %v", err.Error())
		return 1
	}
	dtrace.Logf("## Init with config: %+v", cfg)

	lis, err2 := net.Listen(cfg.NetworkType, cfg.Addr())
	if err2 != nil {
		logger.Errorf("Error listening: %v", err2.Error())
		return 1
	}
	logger.Infof("Server is listening at: %v", cfg.Addr())
	defer lis.Close()

	server := NewServer(cfg)
	
	server.Start()
	defer server.Stop()

	var cIndex uint64 = 0

	for {
        // Listen for an incoming connection.
        conn, err := lis.Accept()
        if err != nil {
			logger.Errorf("Error accepting: %v", err.Error())
            return 1
        }
		if cIndex == math.MaxUint64 {
			cIndex = 0
		}
        cIndex++
		// Handle connections in a new goroutine.
		logger.Infof("# New incoming connection [%d]", cIndex)
        go server.Handle( MakeServConn(conn, cIndex) )
    }

	//return 0
}

