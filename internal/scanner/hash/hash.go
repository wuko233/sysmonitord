package hash

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"sysmonitord/pkg/logger"

	"go.uber.org/zap"
)

type HashAlgorithm interface {
	Hash() hash.Hash
	Name() string
}

// ==== SHA256 ====

type SHA256Algorithm struct{}

func (a *SHA256Algorithm) Hash() hash.Hash {
	return sha256.New()
}

func (a *SHA256Algorithm) Name() string {
	return "sha256"
}

// ==== MD5 ====

type MD5Algorithm struct{}

func (a *MD5Algorithm) Hash() hash.Hash {
	return md5.New()
}

func (a *MD5Algorithm) Name() string {
	return "md5"
}

// ==== xxHash64 ====

// Todo: 添加 xxHash64 实现

// ==== 配置结构体 ====

type Config struct {
	UseFastHash bool
	Threshold   int64
	ChunkSize   int64
	Algorithm   HashAlgorithm
}

// ==== 计算文件哈希 ====

func CalculateHash(filePath string, cfg *Config) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		logger.Log.Warn("[hash]获取文件信息失败", zap.String("path", filePath), zap.Error(err))
		return "", err
	}

	fileSize := info.Size()

	if cfg.Algorithm == nil {
		cfg.Algorithm = &SHA256Algorithm{}
	}

	logger.Log.Debug("[hash]计算文件哈希",
		zap.String("path", filePath),
		zap.Int64("fileSize", fileSize),
		zap.String("Algorithm", cfg.Algorithm.Name()))

	if cfg.UseFastHash && fileSize > cfg.Threshold {
		logger.Log.Debug("[hash] 分层哈希...",
			zap.String("path", filePath),
			zap.Int64("fileSize", fileSize),
		)
		return calculateFast(filePath, fileSize, cfg)
	}

	return calculateFull(filePath, cfg)
}

func calculateFull(filePath string, cfg *Config) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		logger.Log.Warn("[scanner]打开文件失败", zap.String("path", filePath), zap.Error(err))
		return "", err
	}
	defer file.Close()

	hasher := cfg.Algorithm.Hash()
	if _, err := io.Copy(hasher, file); err != nil {
		logger.Log.Error("[scanner]读取文件失败", zap.String("path", filePath), zap.Error(err))
		return "", err
	}

	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	return hashString, nil
}

func calculateFast(filePath string, fileSize int64, cfg *Config) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		logger.Log.Warn("[scanner]打开文件失败", zap.String("path", filePath), zap.Error(err))
		return "", err
	}
	defer file.Close()

	hasher := cfg.Algorithm.Hash()
	chunkSize := cfg.ChunkSize

	if _, err := io.CopyN(hasher, file, chunkSize); err != nil {
		if err != io.EOF {
			return "", err
		}
	}

	tailOffset := fileSize - chunkSize
	if tailOffset < 0 {
		tailOffset = 0
	}

	if _, err := file.Seek(tailOffset, io.SeekStart); err != nil {
		return "", err
	}

	if _, err := io.CopyN(hasher, file, chunkSize); err != nil {
		return "", err
	}

	if err := binary.Write(hasher, binary.BigEndian, fileSize); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
