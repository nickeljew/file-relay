package filerelay

type Config struct {
	host        string
	port        string
	networkType string

	maxRoutines int
}

func InitConfig() (*Config, error) {
	//TODO: read config from yaml file
	return &Config{
		port:        "12721",
		networkType: "tcp",

		maxRoutines: 2,
	}, nil
}
