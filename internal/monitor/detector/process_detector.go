package detector

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sysmonitord/internal/config"
	"sysmonitord/internal/scanner/hash"
	"sysmonitord/internal/scanner/process"
	"sysmonitord/internal/storage"
	"sysmonitord/pkg/logger"
	"time"

	"go.uber.org/zap"
)

type ProcessDetector struct {
	cfg         *config.Config
	whiteList   map[string]string
	storagePath string
	eventChan   chan storage.DubiousProcessInfo
}

func NewProcessDetector(cfg *config.Config) (*ProcessDetector, error) {
	p := &ProcessDetector{
		cfg:       cfg,
		whiteList: make(map[string]string),
		eventChan: make(chan storage.DubiousProcessInfo, 100),
	}

	p.loadWhiteList()
	return p, nil
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
		return nil
	}

	newCount := 0
	for _, proc := range currentProcs {
		_, exists := p.whiteList[proc.Path]
		if !exists {
			logger.Log.Warn("[monitor] 发现新进程", zap.String("name", proc.Name), zap.String("path", proc.Path))
			newCount++

			dubiousProcess := storage.DubiousProcessInfo{
				PID:          proc.PID,
				Name:         proc.Name,
				Path:         proc.Path,
				Cmdline:      proc.Cmdline,
				DiscoveredAt: time.Now().Format("2006-01-02 15:04:05"),
			}

			select {
			case p.eventChan <- dubiousProcess:
			default:
				logger.Log.Warn("[monitor] 进程事件通道已满，丢弃事件", zap.Int32("pid", proc.PID), zap.String("name", proc.Name))
			}
		}
	}

	logger.Log.Info("[monitor] 进程检测完成", zap.Int("total", len(currentProcs)), zap.Int("new", newCount))
	return nil
}

func (p *ProcessDetector) Event() <-chan storage.DubiousProcessInfo {
	return p.eventChan
}

func (p *ProcessDetector) HandleDubiousProcesses(proc storage.DubiousProcessInfo) {

	hashCfg, err := p.cfg.GetHashConfig()
	if err != nil {
		logger.Log.Error("[monitor] 获取哈希配置失败", zap.Error(err))
	}

	logger.Log.Debug("[monitor] 处理可疑进程", zap.Int32("pid", proc.PID), zap.String("name", proc.Name), zap.String("path", proc.Path))

	procHash, err := hash.Calculate(proc.Path, 0, hashCfg)
	if err != nil {
		logger.Log.Error("[monitor] 计算进程哈希失败", zap.String("path", proc.Path), zap.Error(err))
	}

	proc.FileHash = procHash

	if err := storage.SaveDubiousProcesses(proc, p.cfg.Storage.DataDir, p.cfg.Storage.DubiousProcessListFile); err != nil {
		logger.Log.Error("[monitor] 保存可疑进程失败", zap.Int32("pid", proc.PID), zap.String("name", proc.Name), zap.String("path", proc.Path), zap.Error(err))
	}

}

func (p *ProcessDetector) Name() string {
	return "ProcessMonitor"
}
