package safe

import (
	"fmt"
	"os"
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
			cfg, ok := cmd.Context().Value("config").(*config.Config)
			if !ok {
				fmt.Println("无法获取配置")
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
		dubiousFiles = nil
	}
	if len(dubiousFiles) == 0 {
		fmt.Println("没有可疑文件需要处理。")
	}

	dubiousProcesses, err := storage.LoadDubiousProcesses(dataDir, cfg.Storage.DubiousProcessListFile)
	if err != nil {
		fmt.Printf("无法读取可疑进程列表: %v\n", err)
		dubiousProcesses = nil
	}
	if len(dubiousProcesses) == 0 {
		fmt.Println("没有可疑进程需要处理。")
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
	fmt.Println("[1] 逐个确认可疑文件")
	fmt.Println("[2] 逐个确认可疑进程")
	fmt.Println("[3] 将可疑文件全部确认为安全")
	fmt.Println("[4] 将可疑进程全部确认为安全")
	fmt.Println("[5] 全部确认安全 (文件和进程)")
	fmt.Println("[ESC] 退出不处理")
	fmt.Print("请输入选项: ")

	input, err := readKeyWithESC()
	if err != nil {
		fmt.Printf("读取输入失败: %v\n", err)
		return
	}

	switch input {
	case "1":
		conformed, unconfirmed := confirmFiles(dubiousFiles)
		if len(conformed) > 0 {
			fmt.Println("正在处理可疑文件...")
			if err := confirmFilesAsSafe(cfg, conformed); err != nil {
				fmt.Printf("确认文件安全失败: %v\n", err)
			} else {
				fmt.Printf("已将 %d 个文件移入白名单，%d 个文件仍为可疑\n", len(conformed), len(unconfirmed))
			}
		} else {
			fmt.Println("没有文件被确认安全，所有文件仍为可疑。")
		}
	case "2":
		conformed, unconfirmed := confirmProcesses(dubiousProcesses)
		if len(conformed) > 0 {
			fmt.Println("正在处理可疑进程...")
			if err := confirmProcessesAsSafe(cfg, conformed); err != nil {
				fmt.Printf("确认进程安全失败: %v\n", err)
			} else {
				fmt.Printf("已将 %d 个进程移入白名单，%d 个进程仍为可疑\n", len(conformed), len(unconfirmed))
			}
		} else {
			fmt.Println("没有进程被确认安全，所有进程仍为可疑。")
		}
	case "3":
		fmt.Println("正在将所有可疑文件确认为安全...")
		if err := confirmFilesAsSafe(cfg, dubiousFiles); err != nil {
			fmt.Printf("确认文件安全失败: %v\n", err)
		} else {
			fmt.Printf("已将 %d 个文件移入白名单\n", len(dubiousFiles))
		}
	case "4":
		fmt.Println("正在将所有可疑进程确认为安全...")
		if err := confirmProcessesAsSafe(cfg, dubiousProcesses); err != nil {
			fmt.Printf("确认进程安全失败: %v\n", err)
		} else {
			fmt.Printf("已将 %d 个进程移入白名单\n", len(dubiousProcesses))
		}
	case "5":
		fmt.Println("正在将所有可疑文件和进程确认为安全...")
		if err := confirmFilesAsSafe(cfg, dubiousFiles); err != nil {
			fmt.Printf("确认文件安全失败: %v\n", err)
		} else {
			fmt.Printf("已将 %d 个文件移入白名单\n", len(dubiousFiles))
		}
		if err := confirmProcessesAsSafe(cfg, dubiousProcesses); err != nil {
			fmt.Printf("确认进程安全失败: %v\n", err)
		} else {
			fmt.Printf("已将 %d 个进程移入白名单\n", len(dubiousProcesses))
		}
	case "ESC":
		fmt.Println("已取消操作。")
	default:
		fmt.Println("无效选项，已退出。")
	}

}

func confirmProcessesAsSafe(cfg *config.Config, processes []storage.DubiousProcessInfo) error {
	if len(processes) == 0 {
		return nil
	}

	dataDir := cfg.Storage.DataDir

	var toWhitelist []process.ProcessInfo
	for _, proc := range processes {
		toWhitelist = append(toWhitelist, process.ProcessInfo{
			Name:     proc.Name,
			Path:     proc.Path,
			FileHash: proc.FileHash,
		})
	}

	if err := storage.AppendProcessToWhitelist(toWhitelist, dataDir, cfg.Storage.ProcessSystemFile); err != nil {
		return fmt.Errorf("更新进程白名单失败: %v", err)
	}

	logger.Log.Debug("已将可疑进程移入白名单", zap.Int("count", len(toWhitelist)))

	if err := storage.RemoveDubiousProcesses(dataDir, cfg.Storage.DubiousProcessListFile, processes); err != nil {
		return fmt.Errorf("删除可疑进程列表失败: %v", err)
	}
	return nil
}

func confirmFilesAsSafe(cfg *config.Config, files []storage.DubiousFileInfo) error {
	if len(files) == 0 {
		return nil
	}

	dataDir := cfg.Storage.DataDir

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

	if err := storage.RemoveDubiousFiles(dataDir, cfg.Storage.DubiousFileListFile, files); err != nil {
		return fmt.Errorf("删除可疑文件列表失败: %v", err)
	}
	return nil

}

func confirmFiles(files []storage.DubiousFileInfo) (confirmed []storage.DubiousFileInfo, unconfirmed []storage.DubiousFileInfo) {
	if len(files) == 0 {
		return nil, nil
	}

	fmt.Println("\n╔══════════════════════════════════════════════╗")
	fmt.Println("║           逐个确认可疑文件                     ║")
	fmt.Println("╠══════════════════════════════════════════════╣")
	fmt.Println("║     [y] 确认安全     [n] 仍为可疑     [q] 退出   ║")
	fmt.Println("╚══════════════════════════════════════════════╝")

	for i, file := range files {
		fmt.Printf("\n[%d/%d] 文件路径: %s\n", i+1, len(files), file.Path)
		fmt.Printf("       哈希值: %s\n", file.Hash)
		fmt.Printf("       发现时间: %s\n", file.DiscoveredAt)
		fmt.Print("       选择 [y/n/q]: ")

		input, err := readKeyWithESC()
		if err != nil {
			fmt.Printf("读取输入失败: %v，跳过此文件\n", err)
			unconfirmed = append(unconfirmed, file)
			continue
		}

		switch input {
		case "y", "Y":
			fmt.Println("已确认安全")
			confirmed = append(confirmed, file)
		case "n", "N":
			fmt.Println("仍为可疑")
			unconfirmed = append(unconfirmed, file)
		case "q", "Q":
			fmt.Println("已退出确认")
			for j := i; j < len(files); j++ {
				unconfirmed = append(unconfirmed, files[j])
			}
			return confirmed, unconfirmed
		default:
			fmt.Println("无效输入，默认仍为可疑")
			unconfirmed = append(unconfirmed, file)
		}
	}

	fmt.Printf("\n╔══════════════════════════════════════════════╗\n")
	fmt.Printf("║  文件确认完成                                 ║\n")
	fmt.Printf("║  ✓ 确认安全: %d 个                            ║\n", len(confirmed))
	fmt.Printf("║  ✗ 保留可疑: %d 个                            ║\n", len(unconfirmed))
	fmt.Printf("╚══════════════════════════════════════════════╝\n")

	return confirmed, unconfirmed
}

func confirmProcesses(processes []storage.DubiousProcessInfo) (confirmed []storage.DubiousProcessInfo, unconfirmed []storage.DubiousProcessInfo) {
	if len(processes) == 0 {
		return nil, nil
	}

	fmt.Println("\n╔══════════════════════════════════════════════╗")
	fmt.Println("║           逐个确认可疑进程                     ║")
	fmt.Println("╠══════════════════════════════════════════════╣")
	fmt.Println("║     [y] 确认安全     [n] 仍为可疑     [q] 退出   ║")
	fmt.Println("╚══════════════════════════════════════════════╝")

	for i, proc := range processes {
		fmt.Printf("\n[%d/%d] 进程名称: %s\n", i+1, len(processes), proc.Name)
		fmt.Printf("       路径: %s\n", proc.Path)
		fmt.Printf("       哈希值: %s\n", proc.FileHash)
		fmt.Print("       选择 [y/n/q]: ")

		input, err := readKeyWithESC()
		if err != nil {
			fmt.Printf("读取输入失败: %v，跳过此进程\n", err)
			unconfirmed = append(unconfirmed, proc)
			continue
		}

		switch input {
		case "y", "Y":
			fmt.Println("已确认安全")
			confirmed = append(confirmed, proc)
		case "n", "N":
			fmt.Println("仍为可疑")
			unconfirmed = append(unconfirmed, proc)
		case "q", "Q":
			fmt.Println("已退出确认")
			for j := i; j < len(processes); j++ {
				unconfirmed = append(unconfirmed, processes[j])
			}
			return confirmed, unconfirmed
		default:
			fmt.Println("无效输入，默认仍为可疑")
			unconfirmed = append(unconfirmed, proc)
		}
	}

	fmt.Printf("\n╔══════════════════════════════════════════════╗\n")
	fmt.Printf("║  进程确认完成                                 ║\n")
	fmt.Printf("║  ✓ 确认安全: %d 个                            ║\n", len(confirmed))
	fmt.Printf("║  ✗ 保留可疑: %d 个                            ║\n", len(unconfirmed))
	fmt.Printf("╚══════════════════════════════════════════════╝\n")

	return confirmed, unconfirmed
}
