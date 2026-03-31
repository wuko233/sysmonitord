package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sysmonitord/internal/scanner/process"
	"sysmonitord/pkg/logger"
	"time"

	"go.uber.org/zap"
)

type Storage struct {
	DataDir           string
	ProcessSystemFile string
	FileSystemFile    string
}

func InitDataDir(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("[storage]无法创建数据目录: %w", err)
	}
	return nil
}

func SaveProcessSystem(proc []process.ProcessInfo, dataDir string, processSystemFile string) error {
	filePath := filepath.Join(dataDir, processSystemFile)

	f, err := os.Create(filePath) // 覆盖
	if err != nil {
		return fmt.Errorf("[storage]无法创建储存进程文件%s: %w", filePath, err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	currentTime := time.Now().Format("2006-01-02 15:04:05")
	header := fmt.Sprintf("# 进程白名单 - 生成时间: %s\n", currentTime)
	if _, err := writer.WriteString(header); err != nil {
		return err
	}

	for _, p := range proc {
		line := fmt.Sprintf("%v\n", p)
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	logger.Log.Info("[storage]进程白名单保存成功",
		zap.String("file", filePath),
		zap.Int("process_count", len(proc)),
	)

	return nil
}
