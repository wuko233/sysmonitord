package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"sysmonitord/internal/scanner/hash"
	"sysmonitord/pkg/logger"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

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

func ParseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, nil
	}

	// 正则匹配：数字 + 单位
	re := regexp.MustCompile(`(?i)^(\d+)\s*([KMGT]?B?)$`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("无效的大小格式: %s", sizeStr)
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, err
	}

	unit := strings.ToUpper(matches[2])
	var multiplier int64 = 1
	switch unit {
	case "B", "":
		multiplier = 1
	case "KB", "K":
		multiplier = 1024
	case "MB", "M":
		multiplier = 1024 * 1024
	case "GB", "G":
		multiplier = 1024 * 1024 * 1024
	case "TB", "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("未知的单位: %s", unit)
	}

	return value * multiplier, nil
}

func (c *Config) GetHashAlgorithm() (hash.HashAlgorithm, error) {
	algoName := c.Scanner.Hash.Algorithm

	switch strings.ToLower(algoName) {
	case "sha256":
		return &hash.SHA256Algorithm{}, nil
	case "md5":
		return &hash.MD5Algorithm{}, nil
	default:
		return nil, fmt.Errorf("不支持的哈希算法: %s", algoName)
	}
}

func (c *Config) GetFileScannerConfig() (*FileScannerConfig, error) {
	return &c.Scanner.File, nil
}

func (c *Config) GetHashConfig() (*hash.Config, error) {
	algo, err := c.GetHashAlgorithm()
	if err != nil {
		return nil, err
	}

	return &hash.Config{
		UseFastHash: c.Scanner.File.FastHash,
		Threshold:   c.Scanner.File.FastHashSize,
		ChunkSize:   c.Scanner.File.FastHashChunk,
		Algorithm:   algo,
	}, nil
}
