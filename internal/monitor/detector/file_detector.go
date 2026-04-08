package detector

import (
	"sync"
	"sysmonitord/internal/config"
	"sysmonitord/internal/scanner/hash"
	"sysmonitord/internal/storage"
	"sysmonitord/pkg/logger"
	"time"

	"go.uber.org/zap"
)

type FileDetector struct {
	cfg         *config.Config
	whiteList   map[string]string
	storageDir  string
	mu          sync.RWMutex
	timer       map[string]*time.Timer
	debDuration time.Duration
}

func NewFileDetector(cfg *config.Config) (*FileDetector, error) {
	d := &FileDetector{
		cfg:         cfg,
		storageDir:  cfg.Storage.DataDir,
		timer:       make(map[string]*time.Timer),
		debDuration: 1 * time.Second,
	}

	if err := d.loadWhiteList(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *FileDetector) loadWhiteList() error {
	whiteMap, err := storage.LoadFileSystemWhitelist(d.cfg.Storage.DataDir, d.cfg.Storage.FileSystemFile)
	if err != nil {
		logger.Log.Warn("[monitor] 加载文件白名单失败， 使用空白名单...", zap.Error((err)))
		d.whiteList = make(map[string]string)
		return nil
	}

	d.whiteList = whiteMap
	logger.Log.Info("[monitor] 文件白名单加载成功", zap.Int("count", len(d.whiteList)))
	return nil
}

func (d *FileDetector) HandleEvent(eventPath string, opStr string) {
	// Todo: 忽略临时文件等

	d.mu.Lock()
	defer d.mu.Unlock()

	if t, exists := d.timer[eventPath]; exists {
		t.Stop()
	}

	d.timer[eventPath] = time.AfterFunc(d.debDuration, func() {
		d.processEvent(eventPath)
		d.mu.Lock()
		delete(d.timer, eventPath)
		d.mu.Unlock()
	})
}

func (d *FileDetector) processEvent(eventPath string) {
	info, err := storage.GetFileInfo(eventPath)
	if err != nil {
		logger.Log.Warn("[monitor] 获取文件信息失败", zap.String("path", eventPath), zap.Error(err))
		return
	}

	if info.IsDir() {
		return
	}

	hashCfg, err := d.cfg.GetHashConfig()
	if err != nil {
		logger.Log.Warn("[monitor] 获取哈希配置失败", zap.Error(err))
		return
	}
	curHash, err := hash.Calculate(eventPath, info.Size(), hashCfg)
	if err != nil {
		logger.Log.Warn("[monitor] 计算文件哈希失败", zap.String("path", eventPath), zap.Error(err))
		return
	}

	d.mu.RLock()
	whiteHash, exist := d.whiteList[eventPath]
	d.mu.RUnlock()

	isSuspicious := false
	reason := ""

	if !exist {
		// 新文件
		isSuspicious = true
		reason = "新文件"
	} else if whiteHash != curHash {
		// 文件被修改
		isSuspicious = true
		reason = "文件被修改"
	}

	if isSuspicious {
		logger.Log.Warn("[monitor] 可疑文件事件", zap.String("path", eventPath), zap.String("reason", reason), zap.String("hash", curHash))
		dubiousInfo := storage.DubiousFileInfo{
			Path:         eventPath,
			Hash:         curHash,
			DiscoveredAt: time.Now().Format("2006-01-02 15:04:05"),
		}
		if err := storage.SaveDubiousFiles(dubiousInfo, d.storageDir, d.cfg.Storage.DubiousFileListFile); err != nil {
			logger.Log.Error("[monitor] 保存可疑文件信息失败", zap.String("path", eventPath), zap.Error(err))
		}
	}
}
