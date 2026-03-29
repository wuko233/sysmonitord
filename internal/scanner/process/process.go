package process

import (
	"fmt"
	"sysmonitord/pkg/logger"

	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap"
)

type ProcessInfo struct {
	PID     int32  `json:"pid"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Cmdline string `json:"cmdline"`
}

func ScanAllProcesses() ([]ProcessInfo, error) {
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
		processList = append(processList, info)
	}

	logger.Log.Info("[scan]进程扫描完成", zap.Int("进程数量", len(processList)))
	return processList, nil
}

func (p ProcessInfo) String() string {
	return fmt.Sprintf("%s:%s:%d", p.Name, p.Path, p.PID)

	// Todo: 哈希计算
}
