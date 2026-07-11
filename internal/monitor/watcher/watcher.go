package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sysmonitord/internal/config"
	"sysmonitord/internal/pathmatcher"
	"sysmonitord/pkg/logger"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type Watcher struct {
	fsnWatcher *fsnotify.Watcher
	cfg        *config.Config
	eventChan  chan EventMsg
}

type EventMsg struct {
	Path     string
	Op       fsnotify.Op
	FileInfo os.FileInfo
}

func NewWatcher(cfg *config.Config) (*Watcher, error) {
	fsnW, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("[monitor] 创建文件监听失败: %w", err)
	}

	return &Watcher{
		fsnWatcher: fsnW,
		cfg:        cfg,
		eventChan:  make(chan EventMsg, 100),
	}, nil
}

func (w *Watcher) Start() {
	includePaths := w.cfg.Scanner.File.IncludePaths

	paths := w.getWatchPaths(includePaths)

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			logger.Log.Warn("[monitor] 路径不存在", zap.String("path", path))
			continue
		}

		w.addPath(path)
	}

	logger.Log.Info("[monitor] 已启用文件监听", zap.Strings("paths", paths))

	go w.eventLoop()
}

func (w *Watcher) getWatchPaths(includePaths []string) []string {
	roots := make([]string, 0)

	for _, includePath := range includePaths {
		root := getRootFromPattern(includePath)
		if root == "" {
			continue
		}
		roots = append(roots, root)
	}

	seen := make(map[string]struct{})
	uniqueRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		if _, ok := seen[root]; !ok {
			seen[root] = struct{}{}
			uniqueRoots = append(uniqueRoots, root)
		}
	}

	roots = uniqueRoots

	return roots
}

// 路径分类，返回最近根目录
func getRootFromPattern(pattern string) string {
	isRelative := strings.HasPrefix(pattern, "."+string(os.PathSeparator))

	pattern = filepath.Clean(pattern)

	if !pathmatcher.HasGlobMeta(pattern) {
		info, err := os.Stat(pattern)
		if err == nil {
			if info.IsDir() {
				return pattern
			}
			return filepath.Dir(pattern)
		}
	}

	parts := strings.Split(pattern, string(os.PathSeparator))
	rootParts := make([]string, 0, len(parts))

	for _, part := range parts {
		if pathmatcher.HasGlobMeta(part) {
			break
		}
		rootParts = append(rootParts, part)
	}

	root := strings.Join(rootParts, string(os.PathSeparator))
	if root == "" {
		root = string(os.PathSeparator)
	}

	if isRelative {
		root = "." + string(os.PathSeparator) + root
	}

	return root
}

func (w *Watcher) Stop() {
	if w.fsnWatcher != nil {
		_ = w.fsnWatcher.Close()
	}
	close(w.eventChan)
}

func (w *Watcher) Events() <-chan EventMsg {
	return w.eventChan
}

func (w *Watcher) eventLoop() {
	for {
		select {
		case err, ok := <-w.fsnWatcher.Errors:
			if !ok {
				return
			}
			logger.Log.Error("[monitor] 监听错误", zap.Error(err))
		case event, ok := <-w.fsnWatcher.Events:
			if !ok {
				return
			}

			eventPath := filepath.Clean(event.Name)

			// 添加新创建的目录到监听列表， 26.7.11 fix: glob方式需要先添加新增目录后再判断是否该被忽略
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(eventPath)
				if err == nil && info.IsDir() {
					w.addPath(eventPath)
				}
			}

			// 忽略不需要监控的路径
			if w.shouldIgnore(eventPath) {
				continue
			}

			info, err := os.Stat(eventPath)
			if err != nil {
				logger.Log.Debug("[monitor] 检测文件删除或获取文件信息失败", zap.String("path", eventPath))
				w.eventChan <- EventMsg{
					Path:     eventPath,
					Op:       event.Op,
					FileInfo: nil,
				}
				continue
			}

			w.eventChan <- EventMsg{
				Path:     eventPath,
				Op:       event.Op,
				FileInfo: info,
			}

		}
	}
}

func (w *Watcher) addPath(path string) {
	filepath.WalkDir(path, func(subPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			for _, ignorePath := range w.cfg.Scanner.File.ExcludePaths {
				if strings.HasPrefix(subPath, ignorePath) {
					return filepath.SkipDir
				}
			}

			if err := w.fsnWatcher.Add(subPath); err != nil {
				logger.Log.Error("[monitor] 添加监听失败", zap.String("path", subPath), zap.Error(err))
			}
		}

		return nil
	})
}

func (w *Watcher) Errors() <-chan error {
	return w.fsnWatcher.Errors
}

func (w *Watcher) shouldIgnore(path string) bool {
	dataDir := w.cfg.Storage.DataDir

	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		absDataDir = dataDir
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	if strings.HasPrefix(absPath, absDataDir) {
		// 忽略数据目录下的指定文件
		fileSystemName := w.cfg.Storage.FileSystemFile
		processListName := w.cfg.Storage.ProcessSystemFile
		dubiousFileName := w.cfg.Storage.DubiousFileListFile
		dubiousProcessName := w.cfg.Storage.DubiousProcessListFile

		if strings.HasSuffix(absPath, fileSystemName) ||
			strings.HasSuffix(absPath, processListName) ||
			strings.HasSuffix(absPath, dubiousFileName) ||
			strings.HasSuffix(absPath, dubiousProcessName) {
			logger.Log.Debug("[monitor] 忽略数据目录下的文件", zap.String("path", absPath))
			return true
		}
	}

	return false
}
