package safe

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sysmonitord/internal/config"
	"sysmonitord/internal/storage"
	"sysmonitord/pkg/logger"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/term"
)

var SafeCmd = &cobra.Command{
	Use:   "safe",
	Short: "交互式安全确认，将可疑对象加入白名单",
	Long:  "查看当前的可疑文件和进程列表，并选择将其移入白名单。",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig("./config.yaml")
		if err != nil {
			fmt.Printf("加载配置失败: %v\n", err)
			os.Exit(1)
		}

		interactiveSafe(cfg)
	},
}

func readKeyWithESC() (string, error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	b := make([]byte, 1)
	_, err = os.Stdin.Read(b)
	if err != nil {
		return "", err
	}

	if b[0] == 27 { // ESC
		return "ESC", nil
	}
	return string(b), nil
}

func interactiveSafe(cfg *config.Config) {
	dataDir := cfg.Storage.DataDir

	dubiousFiles, err := readDubiousList(filepath.Join(dataDir, cfg.Storage.DubiousFileListFile))
	if err != nil {
		fmt.Printf("无法读取可疑文件列表: %v\n", err)
		return
	}
	if len(dubiousFiles) == 0 {
		fmt.Println("没有可疑文件需要处理。")
		return
	}

	fmt.Println("\n╔══════════════════════════════════════════════╗")
	fmt.Println("║           可疑文件清单 (" + fmt.Sprintf("%d", len(dubiousFiles)) + "个)                 ║")
	fmt.Println("╠══════════════════════════════════════════════╣")

	for _, file := range dubiousFiles {
		fmt.Printf("║ %-45s║\n", file.Path)
	}
	fmt.Println("╚══════════════════════════════════════════════╝")

	fmt.Println("\n请选择操作:")
	fmt.Println("[1] 将以上可疑文件全部确认为安全 (移至白名单)")
	// Todo: 支持逐个确认
	fmt.Println("[ESC] 退出不处理")
	fmt.Print("请输入选项: ")

	input, err := readKeyWithESC()
	if err != nil {
		fmt.Printf("读取输入失败: %v\n", err)
		return
	}

	switch input {
	case "1":
		fmt.Println("正在处理...")
		if err := confirmFilesAsSafe(cfg, dubiousFiles); err != nil {
			fmt.Printf("处理失败: %v\n", err)
		} else {
			fmt.Println("已将可疑文件移入白名单。")
		}
	case "ESC":
		fmt.Println("已取消操作。")
	default:
		fmt.Println("无效选项，已退出。")
	}
}

func readDubiousList(filePath string) ([]storage.DubiousFileInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var list []storage.DubiousFileInfo
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			list = append(list, storage.DubiousFileInfo{
				Path: parts[0],
				Hash: parts[1],
			})
		}
	}
	return list, scanner.Err()
}

func confirmFilesAsSafe(cfg *config.Config, files []storage.DubiousFileInfo) error {
	dataDir := cfg.Storage.DataDir
	whiteListPath := filepath.Join(dataDir, cfg.Storage.FileSystemFile)
	dubiousFile := filepath.Join(dataDir, cfg.Storage.DubiousFileListFile)

	f, err := os.OpenFile(whiteListPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开白名单文件: %v", err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	for _, file := range files {
		line := fmt.Sprintf("%s:%s:%s\n", file.Path, file.Hash, currentTime)
		if _, err := writer.WriteString(line); err != nil {
			return fmt.Errorf("写入白名单失败: %v", err)
		}
		logger.Log.Info("已将可疑文件移入白名单", zap.String("path", file.Path), zap.String("hash", file.Hash))
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("刷新写入缓冲区失败: %v", err)
	}

	// Todo: 逐个删除条目

	if err := os.Remove(dubiousFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除可疑文件列表失败: %v", err)
	}

	return nil
}
