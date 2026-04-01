package file

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sysmonitord/internal/config"
	"sysmonitord/internal/scanner/hash"
	"sysmonitord/pkg/logger"

	"go.uber.org/zap"
)

type FileInfo struct {
	Path    string
	Hash    string
	ModTime int64
	Size    int64
}

type Scanner struct {
	cfg *config.FileScannerConfig
}

func NewScanner(cfg *config.FileScannerConfig) *Scanner {
	return &Scanner{
		cfg: cfg,
	}
}

func (s *Scanner) Scan() ([]FileInfo, error) {
	targetPaths := s.cfg.IncludePaths
	if len(targetPaths) == 0 {
		targetPaths = []string{"/"}
	}

	var allFiles []FileInfo

	for _, root := range targetPaths {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}

		logger.Log.Info("[scan]正在扫描文件系统", zap.String("root", root))

		err := filepath.WalkDir(root, s.WalkFunc(&allFiles))
		if err != nil {
			logger.Log.Error("[scan]扫描文件系统时发生错误", zap.String("root", root), zap.Error(err))
		}
	}

	return allFiles, nil
}

func (s *Scanner) WalkFunc(result *[]FileInfo) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Log.Debug("[scan]跳过路径", zap.String("path", path), zap.Error(err))
			return fs.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		for _, exclude := range s.cfg.ExcludePaths {
			if strings.HasPrefix(path, exclude) {
				logger.Log.Debug("[scan]跳过路径", zap.String("path", path), zap.String("reason", "匹配排除路径"))
				return fs.SkipDir
			}
		}

		info, err := d.Info()
		if err != nil {
			logger.Log.Debug("[scan]无法获取文件信息", zap.String("path", path), zap.Error(err))
			return nil
		}

		if info.Size() != 0 {
			hash, err := hash.SHA256(path, &hash.Config{
				UseFastHash: s.cfg.FastHash,
				Threshold:   s.cfg.FastHashSize,
				ChunkSize:   s.cfg.FastHashChunk,
			})

			if err != nil {
				logger.Log.Debug("[scan]无法计算文件哈希", zap.String("path", path), zap.Error(err))
				return nil
			}

			*result = append(*result, FileInfo{
				Path:    path,
				Hash:    hash,
				ModTime: info.ModTime().Unix(),
				Size:    info.Size(),
			})
		}

		return nil
	}
}

func (f FileInfo) String() string {
	return fmt.Sprintf("%s:%s", f.Path, f.Hash)
}
