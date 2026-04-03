package detector

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sysmonitord/internal/config"
	"sysmonitord/internal/scanner/process"
	"sysmonitord/pkg/logger"

	"go.uber.org/zap"
)

type ProcessDetector struct {
	cfg         *config.Config
	whiteList   map[string]string
	storagePath string
}

func NewProcessDetector(cfg *config.Config) *ProcessDetector {
	p := &ProcessDetector{
		cfg:       cfg,
		whiteList: make(map[string]string),
	}

	p.loadWhiteList()
	return p
}

func (p *ProcessDetector) loadWhiteList() {
	filepath := filepath.Join(p.cfg.Storage.DataDir, p.cfg.Storage.ProcessSystemFile)
	file, err := os.Open(filepath)
	if err != nil {
		logger.Log.Error("[monitor] 加载进程白名单失败", zap.String("file", filepath), zap.Error(err))
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			p.whiteList[parts[1]] = parts[2] // name:path:hash
		}
	}

	logger.Log.Info("[monitor] 进程白名单加载完成", zap.Int("count", len(p.whiteList)))
}

func (p *ProcessDetector) Run() error {
	logger.Log.Info("[monitor] 进程检测已启动")

	currentProcs, err := process.ScanAllProcesses(p.cfg)
	if err != nil {
		logger.Log.Error("[monitor] 扫描进程失败", zap.Error(err))
		return err
	}

	newCount := 0
	for _, proc := range currentProcs {
		_, exists := p.whiteList[proc.Path]
		if !exists {
			logger.Log.Warn("[monitor] 发现新进程", zap.String("name", proc.Name), zap.String("path", proc.Path))
			newCount++

			// Todo: 处理新进程
		}
	}

	logger.Log.Info("[monitor] 进程检测完成", zap.Int("total", len(currentProcs)), zap.Int("new", newCount))
	return nil
}

func (p *ProcessDetector) Name() string {
	return "ProcessMonitor"
}
