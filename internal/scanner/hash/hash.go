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

	"github.com/cespare/xxhash/v2"
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

type XXHash64Algorithm struct{}

func (a *XXHash64Algorithm) Hash() hash.Hash {
	return &xxHash64Wrapper{
		xxhash: xxhash.New(),
	}
}

func (a *XXHash64Algorithm) Name() string {
	return "xxhash64"
}

type xxHash64Wrapper struct {
	xxhash *xxhash.Digest
}

func (w *xxHash64Wrapper) Write(p []byte) (n int, err error) {
	return w.xxhash.Write(p)
}

// Sum 返回当前哈希值，追加到 b 后面
// xxHash64 返回 8 字节的哈希值（小端序）
func (w *xxHash64Wrapper) Sum(b []byte) []byte {
	// 获取当前的 64 位哈希值
	h := w.xxhash.Sum64()

	// 将 uint64 转换为 8 字节的小端序字节数组
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, h)

	// 追加到输入的 b 后面
	return append(b, buf...)
}

// Reset 重置哈希状态
func (w *xxHash64Wrapper) Reset() {
	w.xxhash.Reset()
}

// Size 返回哈希值的字节数
func (w *xxHash64Wrapper) Size() int {
	return 8 // xxHash64 输出 64 位 = 8 字节
}

// BlockSize 返回底层哈希的块大小
func (w *xxHash64Wrapper) BlockSize() int {
	return w.xxhash.BlockSize()
}

// Sum64 提供直接获取 uint64 的便捷方法
func (w *xxHash64Wrapper) Sum64() uint64 {
	return w.xxhash.Sum64()
}

// ==== 配置结构体 ====

type Config struct {
	UseFastHash bool
	Threshold   int64
	ChunkSize   int64
	Algorithm   HashAlgorithm
}

// ==== 计算文件哈希 ====
func Calculate(filePath string, fileSize int64, cfg *Config) (string, error) {
	if cfg == nil {
		cfg = &Config{
			Algorithm: &SHA256Algorithm{},
		}
	}

	if fileSize == 0 {
		info, err := os.Stat(filePath)
		if err != nil {
			logger.Log.Warn("[scanner]获取文件信息失败", zap.String("path", filePath), zap.Error(err))
			return "", err
		}
		fileSize = info.Size()
	}

	if cfg.Algorithm == nil {
		cfg.Algorithm = &SHA256Algorithm{}
	}

	logger.Log.Debug("[scanner]计算文件哈希", zap.String("path", filePath), zap.Int64("size", fileSize), zap.String("algorithm", cfg.Algorithm.Name()))

	if cfg.UseFastHash && fileSize > cfg.Threshold {
		return calculateFast(filePath, fileSize, cfg)
	} else {
		return calculateFull(filePath, cfg)
	}
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
