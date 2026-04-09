package safe

import (
	"fmt"
	"os"
	"path/filepath"
	"sysmonitord/internal/config"
	"sysmonitord/internal/scanner/file"
	"sysmonitord/internal/scanner/process"
	"sysmonitord/internal/storage"
	"sysmonitord/pkg/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/term"
)

func NewSafeCmd() *cobra.Command {
	cmd := &cobra.Command{
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
	return cmd
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

	dubiousFiles, err := storage.LoadDubiousFiles(dataDir, cfg.Storage.DubiousFileListFile)
	if err != nil {
		fmt.Printf("无法读取可疑文件列表: %v\n", err)
		return
	}
	if len(dubiousFiles) == 0 {
		fmt.Println("没有可疑文件需要处理。")
		return
	}

	dubiousProcesses, err := storage.LoadDubiousProcesses(dataDir, cfg.Storage.DubiousProcessListFile)
	if err != nil {
		fmt.Printf("无法读取可疑进程列表: %v\n", err)
		return
	}
	if len(dubiousProcesses) == 0 {
		fmt.Println("没有可疑进程需要处理。")
		return
	}

	fmt.Println("\n╔══════════════════════════════════════════════╗")
	fmt.Println("║           可疑文件清单 (" + fmt.Sprintf("%d", len(dubiousFiles)) + "个)                 ║")
	fmt.Println("╠══════════════════════════════════════════════╣")

	for _, file := range dubiousFiles {
		fmt.Printf("║ %-45s║\n", file.Path)
	}
	fmt.Println("╚══════════════════════════════════════════════╝")

	fmt.Println("\n╔══════════════════════════════════════════════╗")
	fmt.Println("║           可疑进程清单 (" + fmt.Sprintf("%d", len(dubiousProcesses)) + "个)                 ║")
	fmt.Println("╠══════════════════════════════════════════════╣")
	for _, proc := range dubiousProcesses {
		fmt.Printf("║ %-45s║\n", fmt.Sprintf("%s (%s)", proc.Name, proc.Path))
	}
	fmt.Println("╚══════════════════════════════════════════════╝")

	fmt.Println("\n请选择操作:")
	fmt.Println("[1] 将以上可疑文件全部确认为安全 (移至白名单)")
	fmt.Println("[2] 将以上可疑进程全部确认为安全 (移至白名单)")
	fmt.Println("[3] 全部确认安全 (文件和进程)")
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
	case "2":
		fmt.Println("正在处理...")
		if err := confirmProcessesAsSafe(cfg, dubiousProcesses); err != nil {
			fmt.Printf("处理失败: %v\n", err)
		} else {
			fmt.Println("已将可疑进程移入白名单。")
		}
	case "3":
		fmt.Println("正在处理...")
		if err := confirmFilesAsSafe(cfg, dubiousFiles); err != nil {
			fmt.Printf("处理失败: %v\n", err)
		} else {
			fmt.Println("已将可疑文件移入白名单。")
		}
		if err := confirmProcessesAsSafe(cfg, dubiousProcesses); err != nil {
			fmt.Printf("处理失败: %v\n", err)
		} else {
			fmt.Println("已将可疑进程移入白名单。")
		}
	case "ESC":
		fmt.Println("已取消操作。")
	default:
		fmt.Println("无效选项，已退出。")
	}

}

func confirmProcessesAsSafe(cfg *config.Config, processes []storage.DubiousProcessInfo) error {
	dataDir := cfg.Storage.DataDir
	whiteListPath := filepath.Join(dataDir, cfg.Storage.ProcessSystemFile)
	// dubiousFile := filepath.Join(dataDir, cfg.Storage.DubiousProcessListFile)

	f, err := os.OpenFile(whiteListPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开白名单文件: %v", err)
	}
	defer f.Close()

	var toWhitelist []process.ProcessInfo
	for _, proc := range processes {
		toWhitelist = append(toWhitelist, process.ProcessInfo{
			Name:     proc.Name,
			Path:     proc.Path,
			FileHash: proc.FileHash,
		})
	}

	if err := storage.AppendProcessToWhitelist(toWhitelist, dataDir, cfg.Storage.ProcessSystemFile); err != nil {
		return fmt.Errorf("更新白名单失败: %v", err)
	}

	logger.Log.Debug("已将可疑进程移入白名单", zap.Int("count", len(toWhitelist)))

	// Todo: 逐个删除条目
	if err := storage.RemoveDubiousProcesses(dataDir, cfg.Storage.DubiousProcessListFile, []storage.DubiousProcessInfo{}); err != nil {
		return fmt.Errorf("删除可疑进程列表失败: %v", err)
	}

	return nil

}

func confirmFilesAsSafe(cfg *config.Config, files []storage.DubiousFileInfo) error {
	dataDir := cfg.Storage.DataDir
	whiteListPath := filepath.Join(dataDir, cfg.Storage.FileSystemFile)
	// dubiousFile := filepath.Join(dataDir, cfg.Storage.DubiousFileListFile)

	f, err := os.OpenFile(whiteListPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开白名单文件: %v", err)
	}
	defer f.Close()

	var toWhitelist []file.FileInfo
	for _, f := range files {
		toWhitelist = append(toWhitelist, file.FileInfo{
			Path: f.Path,
			Hash: f.Hash,
		})
	}

	if err := storage.AppendFileToWhitelist(toWhitelist, dataDir, cfg.Storage.FileSystemFile); err != nil {
		return fmt.Errorf("更新白名单失败: %v", err)
	}

	logger.Log.Debug("已将可疑文件移入白名单", zap.Int("count", len(toWhitelist)))

	// Todo: 逐个删除条目
	if err := storage.RemoveDubiousFiles(dataDir, cfg.Storage.DubiousFileListFile, []storage.DubiousFileInfo{}); err != nil {
		return fmt.Errorf("删除可疑文件列表失败: %v", err)
	}

	return nil
}
