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
			fmt.Printf("[monitor] 路径不存在: %s\n", path)
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
			fmt.Printf("[monitor] 监听错误: %v\n", err)
		case event, ok := <-w.fsnWatcher.Events:
			if !ok {
				return
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
				fmt.Printf("[monitor] 添加监听失败: %s, 错误: %v\n", subPath, err)
			}
		}

		return nil
	})
}

func (w *Watcher) Errors() <-chan error {
	return w.fsnWatcher.Errors
}
