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
	ServerAddress     string `json:"server_address,omitempty"`
	GRPCServerAddress string `json:"grpc_server_address,omitempty"`
	BaseURL           string `json:"base_url,omitempty"`
	FileStoragePath   string `json:"file_storage_path,omitempty"`
	DatabaseDSN       string `json:"database_dsn,omitempty"`
	TrustedSubnet     string `json:"trusted_subnet"`
	EnableHTTPS       bool   `json:"enable_https"`
}

// Parse configs
func Parse() Config {
	flagConfigs := Config{}
	var configFilePath string
	flag.StringVar(&flagConfigs.ServerAddress, "a", "", "server's address")
	flag.StringVar(&flagConfigs.GRPCServerAddress, "ga", "", "grpc server's address")
	flag.StringVar(&flagConfigs.BaseURL, "b", "", "base address of the resulting shortened URL")
	flag.StringVar(&flagConfigs.FileStoragePath, "f", "", "file storage path")
	flag.StringVar(&flagConfigs.DatabaseDSN, "d", "", "database URL")
	flag.BoolVar(&flagConfigs.EnableHTTPS, "s", false, "enable HTTPS")
	flag.StringVar(&flagConfigs.TrustedSubnet, "t", "", "trusted subnet")
	flag.StringVar(&configFilePath, "c", "", "file path with json application configs")
	flag.Parse()

	if envConfigFilePath := os.Getenv("CONFIG"); envConfigFilePath != "" {
		configFilePath = envConfigFilePath
	}

	defaultConfigs := Config{
		ServerAddress:     "localhost:8080",
		GRPCServerAddress: ":3200",
		BaseURL:           "http://localhost:8080",
	}
	configs := Config{}
	applyConfigs(&configs, defaultConfigs)
	applyConfigs(&configs, jsonConfigs(configFilePath))
	applyConfigs(&configs, flagConfigs)
	applyConfigs(&configs, envConfigs())

	return configs
}

func applyConfigs(dst *Config, src Config) {
	if src.ServerAddress != "" {
		dst.ServerAddress = src.ServerAddress
	}
	if src.GRPCServerAddress != "" {
		dst.GRPCServerAddress = src.GRPCServerAddress
	}
	if src.BaseURL != "" {
		dst.BaseURL = src.BaseURL
	}
	if src.FileStoragePath != "" {
		dst.FileStoragePath = src.FileStoragePath
	}
	if src.DatabaseDSN != "" {
		dst.DatabaseDSN = src.DatabaseDSN
	}
	if src.TrustedSubnet != "" {
		dst.TrustedSubnet = src.TrustedSubnet
	}
	dst.EnableHTTPS = src.EnableHTTPS
}

func jsonConfigs(configFilePath string) Config {
	configs := Config{}
	configData, err := os.ReadFile(configFilePath)
	if err == nil {
		if err = json.Unmarshal(configData, &configs); err != nil {
			log.Printf("failed to parse configs: %s\n", err.Error())
		}
	} else {
		log.Printf("failed to read configs: %s\n", err.Error())
	}

	return configs
}

func envConfigs() Config {
	configs := Config{
		ServerAddress:     os.Getenv("SERVER_ADDRESS"),
		GRPCServerAddress: os.Getenv("GRPC_SERVER_ADDRESS"),
		BaseURL:           os.Getenv("BASE_URL"),
		FileStoragePath:   os.Getenv("FILE_STORAGE_PATH"),
		DatabaseDSN:       os.Getenv("DATABASE_DSN"),
		TrustedSubnet:     os.Getenv("TRUSTED_SUBNET"),
	}

	enableHTTPS, err := strconv.ParseBool(os.Getenv("ENABLE_HTTPS"))
	if err != nil {
		configs.EnableHTTPS = enableHTTPS
	}

	return configs
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
