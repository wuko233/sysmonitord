package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"sysmonitord/internal/scanner/hash"
)

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
