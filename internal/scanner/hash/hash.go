package hash

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"sysmonitord/pkg/logger"

	"go.uber.org/zap"
)

type Config struct {
	UseFastHash bool
	Threshold   int64
	ChunkSize   int64
}

func SHA256(filePath string, cfg *Config) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		logger.Log.Warn("[hash]获取文件信息失败", zap.String("path", filePath), zap.Error(err))
		return "", err
	}

	fileSize := info.Size()

	if cfg != nil && cfg.UseFastHash && fileSize > cfg.Threshold {
		logger.Log.Debug("[hash] 分层哈希...",
			zap.String("path", filePath),
			zap.Int64("fileSize", fileSize),
		)
		return calculateFastHash(filePath, fileSize, cfg.ChunkSize)
	}

	return calculateFullHash(filePath)
}

func calculateFullHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		logger.Log.Warn("[scanner]打开文件失败", zap.String("path", filePath), zap.Error(err))
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		logger.Log.Error("[scanner]读取文件失败", zap.String("path", filePath), zap.Error(err))
		return "", err
	}

	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	return hashString, nil
}

func calculateFastHash(filePath string, fileSize int64, chunkSize int64) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		logger.Log.Warn("[scanner]打开文件失败", zap.String("path", filePath), zap.Error(err))
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()

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
