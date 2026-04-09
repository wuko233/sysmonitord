package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sysmonitord/internal/config"
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
	paths := w.cfg.Scanner.File.IncludePaths

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

			// 忽略不需要监控的路径
			if w.shouldIgnore(event.Name) {
				continue
			}

			// 添加新创建的目录到监听列表
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					w.addPath(event.Name)
				}
			}

			info, err := os.Stat(event.Name)
			if err != nil {
				logger.Log.Debug("[monitor] 检测文件删除或获取文件信息失败", zap.String("path", event.Name))
				w.eventChan <- EventMsg{
					Path:     event.Name,
					Op:       event.Op,
					FileInfo: nil,
				}
				continue
			}

			w.eventChan <- EventMsg{
				Path:     event.Name,
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
