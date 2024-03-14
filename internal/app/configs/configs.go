package configs

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
)

// Application configs
type Config struct {
	ServerAddress   string `json:"server_address,omitempty"`
	BaseURL         string `json:"base_url,omitempty"`
	FileStoragePath string `json:"file_storage_path,omitempty"`
	DatabaseDSN     string `json:"database_dsn,omitempty"`
	EnableHTTPS     bool   `json:"enable_https"`
}

// Parse configs
func Parse() Config {
	var (
		flagServerAddress   string
		flagBaseURL         string
		flagFileStoragePath string
		flagDatabaseDSN     string
		flagEnableHTTPS     bool
		configFilePath      string
	)
	flag.StringVar(&flagServerAddress, "a", "", "server's address")
	flag.StringVar(&flagBaseURL, "b", "", "base address of the resulting shortened URL")
	flag.StringVar(&flagFileStoragePath, "f", "", "file storage path")
	flag.StringVar(&flagDatabaseDSN, "d", "", "database URL")
	flag.BoolVar(&flagEnableHTTPS, "s", false, "enable HTTPS")
	flag.StringVar(&configFilePath, "c", "", "file path with json application configs")
	flag.Parse()

	if envConfigFilePath := os.Getenv("CONFIG"); envConfigFilePath != "" {
		configFilePath = envConfigFilePath
	}

	config := Config{ServerAddress: "localhost:8080", BaseURL: "http://localhost:8080"}
	configData, err := os.ReadFile(configFilePath)
	if err == nil {
		if err = json.Unmarshal(configData, &config); err != nil {
			log.Printf("failed to parse configs: %s\n", err.Error())
		}
	} else {
		log.Printf("failed to read configs: %s\n", err.Error())
	}

	if flagServerAddress != "" {
		config.ServerAddress = flagServerAddress
	}
	if flagBaseURL != "" {
		config.BaseURL = flagBaseURL
	}
	if flagFileStoragePath != "" {
		config.FileStoragePath = flagFileStoragePath
	}
	if flagDatabaseDSN != "" {
		config.DatabaseDSN = flagDatabaseDSN
	}

	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		config.ServerAddress = envServerAddress
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		config.BaseURL = envBaseURL
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		config.FileStoragePath = envFileStoragePath
	}
	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		config.DatabaseDSN = envDatabaseDSN
	}
	envEnableHTTPS, err := strconv.ParseBool(os.Getenv("ENABLE_HTTPS"))

	if err == nil {
		config.EnableHTTPS = config.EnableHTTPS || flagEnableHTTPS || envEnableHTTPS
	} else {
		config.EnableHTTPS = config.EnableHTTPS || flagEnableHTTPS
	}

	return config
}

// Use database storage
func (c Config) UseDBStorage() bool {
	return c.DatabaseDSN != ""
}

// Use file storage
func (c Config) UseFileStorage() bool {
	return c.FileStoragePath != ""
}

// Use HTTPS
func (c Config) UseHTTPS() bool {
	return c.EnableHTTPS
}
