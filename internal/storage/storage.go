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

type DubiousProcessInfo struct {
	PID          int32
	Name         string
	Path         string
	Cmdline      string
	FileHash     string
	DiscoveredAt string
}

func SaveDubiousProcesses(proc DubiousProcessInfo, dataDir string, dubiousProcessFile string) error {
	filePath := filepath.Join(dataDir, dubiousProcessFile)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[storage]无法创建或打开可疑进程记录文件%s: %w", filePath, err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	line := fmt.Sprintf("%s:%s:%s\n",
		proc.Name, proc.Path, proc.FileHash)
	if _, err := writer.WriteString(line); err != nil {
		return err
	}

	return writer.Flush()
}

func LoadDubiousProcesses(dataDir string, dubiousProcessFile string) ([]DubiousProcessInfo, error) {
	filePath := filepath.Join(dataDir, dubiousProcessFile)
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("[storage]无法打开可疑进程记录文件%s: %w", filePath, err)
	}
	defer f.Close()

	var processes []DubiousProcessInfo
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			processes = append(processes, DubiousProcessInfo{
				Name:     parts[0],
				Path:     parts[1],
				FileHash: parts[2],
			})
		}
	}
	return processes, scanner.Err()
}

func AppendProcessToWhitelist(procs []process.ProcessInfo, dataDir string, processSystemFile string) error {
	filePath := filepath.Join(dataDir, processSystemFile)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[storage]无法创建或打开进程白名单文件%s: %w", filePath, err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	for _, p := range procs {
		line := p.String() + "\n"
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func RemoveDubiousProcesses(dataDir string, dubiousProcessFile string, toKeep []DubiousProcessInfo) error {
	filePath := filepath.Join(dataDir, dubiousProcessFile)
	if len(toKeep) == 0 {
		return os.Remove(filePath)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("[storage]无法创建可疑进程记录文件%s: %w", filePath, err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	for _, proc := range toKeep {
		line := fmt.Sprintf("%s:%s:%s\n", proc.Name, proc.Path, proc.FileHash)
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	return writer.Flush()
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

func AppendFileToWhitelist(files []file.FileInfo, dataDir string, fileSystemFile string) error {
	filePath := filepath.Join(dataDir, fileSystemFile)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[storage]无法创建或打开文件白名单文件%s: %w", filePath, err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	for _, f := range files {
		line := f.String() + "\n"
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func RemoveDubiousFiles(dataDir string, dubiousFileName string, toKeep []DubiousFileInfo) error {
	filePath := filepath.Join(dataDir, dubiousFileName)
	if len(toKeep) == 0 {
		return os.Remove(filePath)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("[storage]无法创建可疑文件记录文件%s: %w", filePath, err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	for _, file := range toKeep {
		line := fmt.Sprintf("%s:%s:%s\n", file.Path, file.Hash, file.DiscoveredAt)
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func LoadDubiousFiles(dataDir string, dubiousFileName string) ([]DubiousFileInfo, error) {
	filePath := filepath.Join(dataDir, dubiousFileName)
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("[storage]无法打开可疑文件记录文件%s: %w", filePath, err)
	}
	defer f.Close()

	var files []DubiousFileInfo
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			files = append(files, DubiousFileInfo{
				Path:         parts[0],
				Hash:         parts[1],
				DiscoveredAt: parts[2],
			})
		}
	}
	return files, scanner.Err()
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
