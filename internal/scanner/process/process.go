package process

import (
	"fmt"
	"os"
	"sysmonitord/internal/scanner/hash"
	"sysmonitord/pkg/logger"

	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap"
)

type ProcessInfo struct {
	PID      int32  `json:"pid"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Cmdline  string `json:"cmdline"`
	FileHash string `json:"file_hash"`
}

func ScanAllProcesses(hashCfg *hash.Config) ([]ProcessInfo, error) {
	logger.Log.Info("[scan]正在扫描系统中的所有进程...")

	pids, err := process.Pids()
	if err != nil {
		logger.Log.Error("[scan]获取进程列表失败", zap.Error(err))
		return nil, err
	}

	var processList []ProcessInfo
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			continue // 跳过临时进程
		}

		name, err := p.Name()
		if err != nil {
			name = "unknown"
		}

		exePath, err := p.Exe()
		if err != nil {
			exePath = ""
		}

		cmdline, err := p.Cmdline()
		if err != nil {
			cmdline = ""
		}

		info := ProcessInfo{
			PID:     pid,
			Name:    name,
			Path:    exePath,
			Cmdline: cmdline,
		}

		if exePath != "" {
			if _, err := os.Stat(exePath); err == nil {
				fileHash, err := hash.CalculateHash(exePath, hashCfg)
				if err == nil {
					info.FileHash = fileHash
				} else {
					logger.Log.Warn("[scan]计算文件哈希失败", zap.String("path", exePath), zap.Error(err))
				}
			}
		}

		processList = append(processList, info)
	}

	logger.Log.Info("[scan]进程扫描完成", zap.Int("进程数量", len(processList)))
	return processList, nil
}

func (p ProcessInfo) String() string {
	return fmt.Sprintf("%s:%s:%s", p.Name, p.Path, p.FileHash)
}
