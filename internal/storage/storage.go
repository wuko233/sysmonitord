package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sysmonitord/internal/scanner/file"
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

func SaveFileSystem(files []file.FileInfo, dataDir string, fileSystemFile string) error {
	filePath := filepath.Join(dataDir, fileSystemFile)
	file, err := os.Create(filePath) // 覆盖
	if err != nil {
		return fmt.Errorf("[storage]无法创建储存文件系统文件%s: %w", filePath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	currentTime := time.Now().Format("2006-01-02 15:04:05")
	header := fmt.Sprintf("# 文件系统白名单 - 生成时间: %s\n", currentTime)
	if _, err := writer.WriteString(header); err != nil {
		return err
	}

	for _, f := range files {
		line := fmt.Sprintf("%v\n", f)
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	logger.Log.Info("[storage]文件系统白名单保存成功",
		zap.String("file", filePath),
		zap.Int("file_count", len(files)),
	)

	return nil
}

type DubiousFileInfo struct {
	Path         string
	Hash         string
	DiscoveredAt string
}

func SaveDubiousFiles(files DubiousFileInfo, dataDir string, dubiousFileName string) error {
	filePath := filepath.Join(dataDir, dubiousFileName)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[storage]无法创建或打开可疑文件记录文件%s: %w", filePath, err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	line := fmt.Sprintf("%s:%s:%s\n", files.Path, files.Hash, files.DiscoveredAt)
	if _, err := writer.WriteString(line); err != nil {
		return err
	}

	return writer.Flush()
}

func LoadFileSystemWhitelist(dataDir string, fileSystemFile string) (map[string]string, error) {
	filePath := filepath.Join(dataDir, fileSystemFile)
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("[storage]无法打开文件系统白名单文件%s: %w", filePath, err)
	}
	defer f.Close()

	whitelist := make(map[string]string)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) >= 2 {
			whitelist[parts[0]] = parts[1]
		}
	}
	return whitelist, nil
}

func GetFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
