package config

import (
	"fmt"
	"os"
	"sysmonitord/pkg/logger"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Audit   AuditConfig   `yaml:"audit"`
	Scanner ScannerConfig `yaml:"scanner"`
	Storage StorageConfig `yaml:"storage"`
}

type AuditConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Server     string `yaml:"server"`
	Port       int    `yaml:"port"`
	BufferSize int    `yaml:"buffer_size"`
}

type ScannerConfig struct {
	File FileScannerConfig `yaml:"file"`
	Hash hashConfig        `yaml:"hash"`
}

type hashConfig struct {
	Algorithm string `yaml:"algorithm"`
}

type StorageConfig struct {
	DataDir           string `yaml:"data_dir"`
	ProcessSystemFile string `yaml:"process_system_file"`
	FileSystemFile    string `yaml:"file_system_file"`
}

type FileScannerConfig struct {
	IncludePaths     []string `yaml:"include_paths"`
	ExcludePaths     []string `yaml:"exclude_paths"`
	FastHash         bool     `yaml:"fast_hash"`
	FastHashSizeRaw  string   `yaml:"fast_hash_size"`
	FastHashChunkRaw string   `yaml:"fast_hash_chunk"`
	FastHashSize     int64
	FastHashChunk    int64
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

	// 解析 FastHashSize
	cfg.Scanner.File.FastHashSize, err = ParseSize(cfg.Scanner.File.FastHashSizeRaw)
	if err != nil {
		return nil, fmt.Errorf("解析 fast_hash_size 失败: %w", err)
	}

	// 解析 FastHashChunk
	cfg.Scanner.File.FastHashChunk, err = ParseSize(cfg.Scanner.File.FastHashChunkRaw)
	if err != nil {
		return nil, fmt.Errorf("解析 fast_hash_chunk 失败: %w", err)
	}

	logger.Log.Debug("配置加载完成",
		zap.Int64("fast_hash_size", cfg.Scanner.File.FastHashSize),
		zap.Int64("fast_hash_chunk", cfg.Scanner.File.FastHashChunk),
	)

	return &cfg, nil
}
