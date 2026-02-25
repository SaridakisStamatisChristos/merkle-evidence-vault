package config

import (
	"time"
)

type ServerConfig struct {
	HTTPAddr    string        `yaml:"http_addr"`
	GRPCAddr    string        `yaml:"grpc_addr"`
	ReadTimeout time.Duration `yaml:"read_timeout"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
}

// Load is a placeholder for config loading (file + env overrides).
func Load(path string) (*Config, error) {
	return &Config{Server: ServerConfig{HTTPAddr: ":8443", GRPCAddr: ":9443", ReadTimeout: 30 * time.Second}}, nil
}
