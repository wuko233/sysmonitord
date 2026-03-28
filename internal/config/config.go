package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Audit   AuditConfig   `yaml:"audit"`
	Scanner ScannerConfig `yaml:"scanner"`
}

type AuditConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Server     string `yaml:"server"`
	Port       int    `yaml:"port"`
	BufferSize int    `yaml:"buffer_size"`
}

type ScannerConfig struct {
	File FileScannerConfig `yaml:"file"`
}

type FileScannerConfig struct {
	ExcludePaths []string `yaml:"exclude_paths"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件： %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("无法解析配置文件： %w", err)
	}
	return &cfg, nil
}
