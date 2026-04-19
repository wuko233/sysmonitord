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

	"github.com/schollz/progressbar/v3"
	"go.uber.org/zap"
)

type FileInfo struct {
	Path    string
	Hash    string
	ModTime int64
	Size    int64
}

type Scanner struct {
	cfg *config.Config
}

func NewScanner(cfg *config.Config) *Scanner {
	return &Scanner{
		cfg: cfg,
	}
}

func (s *Scanner) Scan() ([]FileInfo, error) {
	targetPaths := s.cfg.Scanner.File.IncludePaths
	if len(targetPaths) == 0 {
		targetPaths = []string{"/"}
	}

	var allPaths []string
	for _, root := range targetPaths {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			logger.Log.Debug("扫描路径不存在，已跳过", zap.String("path", root))
			continue
		}
		logger.Log.Info("[scan]正在扫描文件系统", zap.String("root", root))

		err := filepath.WalkDir(root, s.collectPathsFunc(&allPaths))
		if err != nil {
			logger.Log.Error("[scan]扫描文件系统时发生错误", zap.String("root", root), zap.Error(err))
		}
	}

	logger.Log.Info("[scan]开始计算文件哈希", zap.Int("文件数量", len(allPaths)))

	var allFiles []FileInfo
	hashCfg, _ := s.cfg.GetHashConfig()

	var bar *progressbar.ProgressBar
	if isInteractiveTerminal() {
		bar = progressbar.NewOptions(len(allPaths),
			progressbar.OptionSetDescription("[scan]计算文件哈希"),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionSetItsString("files"),
			progressbar.OptionOnCompletion(func() {
				logger.Log.Info("[scan]文件哈希计算完成")
			}),
		)
	} else {
		logger.Log.Info("[scan]开始计算文件哈希", zap.Int("total_files", len(allPaths)))
	}

	for _, path := range allPaths {
		if bar != nil {
			bar.Add(1)
		}

		info, err := os.Stat(path)
		if err != nil {
			logger.Log.Debug("[scan]无法获取文件信息", zap.String("path", path), zap.Error(err))
			continue
		}

		if info.Size() > 0 {
			hash, err := hash.Calculate(path, info.Size(), hashCfg)
			if err != nil {
				logger.Log.Debug("[scan]无法计算文件哈希", zap.String("path", path), zap.Error(err))
				continue
			}

			allFiles = append(allFiles, FileInfo{
				Path:    path,
				Hash:    hash,
				ModTime: info.ModTime().Unix(),
				Size:    info.Size(),
			})
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

		for _, exclude := range s.cfg.Scanner.File.ExcludePaths {
			if strings.HasPrefix(path, exclude) {
				logger.Log.Debug("[scan]跳过路径", zap.String("path", path), zap.String("reason", "匹配排除路径"))
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			logger.Log.Debug("[scan]无法获取文件信息", zap.String("path", path), zap.Error(err))
			return nil
		}

		if info.Size() > 0 {
			hashCfg, err := s.cfg.GetHashConfig()
			if err != nil {
				logger.Log.Debug("[scan]无法获取哈希配置", zap.String("path", path), zap.Error(err))
				return nil
			}

			hash, err := hash.Calculate(path, info.Size(), hashCfg)
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

func (s *Scanner) collectPathsFunc(result *[]string) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Log.Debug("[scan]跳过路径", zap.String("path", path), zap.Error(err))
			return fs.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		for _, exclude := range s.cfg.Scanner.File.ExcludePaths {
			if strings.HasPrefix(path, exclude) {
				logger.Log.Debug("[scan]跳过路径", zap.String("path", path), zap.String("reason", "匹配排除路径"))
				return nil
			}
		}

		*result = append(*result, path)
		return nil
	}
}

func isInteractiveTerminal() bool {
	fileInfo, err := os.Stderr.Stat()
	if err != nil {
		return false
	}

	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func (f FileInfo) String() string {
	return fmt.Sprintf("%s:%s", f.Path, f.Hash)
}
