package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress   string `env:"RUN_ADDRESS"`
	DatabaseURI     string `env:"DATABASE_URI"`
	AccrualAddress  string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	LogLevel        string `env:"LOG_LEVEL"`
	SecretKey       string `env:"SECRET_KEY" default:""`
	WorkerNumber    int    `env:"WORKERS"`
	TaskChannelSize int
}

func NewConfig() (Config, error) {
	flags := parseServerFlags()

	config := Config{}
	if err := env.Parse(&config); err != nil {
		return Config{}, err
	}

	if config.ServerAddress == "" {
		config.ServerAddress = flags.ServerAddress
	}

	if config.DatabaseURI == "" {
		config.DatabaseURI = flags.DatabaseURI
	}

	if config.AccrualAddress == "" {
		config.AccrualAddress = flags.AccrualAddress
	}

	if config.LogLevel == "" {
		config.LogLevel = flags.LogLevel
	}

	if config.WorkerNumber == 0 {
		config.WorkerNumber = flags.WorkerNumber
	}

	config.TaskChannelSize = config.WorkerNumber * 2

	return config, nil
}

func parseServerFlags() Config {
	serverAddress := flag.String("a", "localhost:8080", "address and port to run server")
	databaseURI := flag.String("d", "", "database URI")
	accrualAddress := flag.String("r", "localhost:8081", "address and port of the accrual system")
	logLevel := flag.String("l", "INFO", "specify log level")
	workerNumber := flag.Int("w", 3, "specify worker number")

	flag.Parse()

	return Config{
		ServerAddress:  *serverAddress,
		DatabaseURI:    *databaseURI,
		AccrualAddress: *accrualAddress,
		LogLevel:       *logLevel,
		WorkerNumber:   *workerNumber,
	}
}
