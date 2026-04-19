package status

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sysmonitord/internal/config"

	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "显示系统状态",
		Long:  "显示Sysmonitod的当前状态",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, ok := cmd.Context().Value("config").(*config.Config)
			if !ok {
				fmt.Println("无法获取配置")
				os.Exit(1)
			}

			printStatus(cfg)
		},
	}
	return cmd
}

func printStatus(cfg *config.Config) {
	dataDir := cfg.Storage.DataDir

	fmt.Println("Sysmonitord Status")
	fmt.Println("================")

	// Todo: 显示运行时长
	runtimeInfo := "N/A"

	fmt.Printf("Runtime: %s\n", runtimeInfo)
	fmt.Printf("Data Directory: %s\n", dataDir)

	fmt.Println()

	fmt.Println("[白名单统计]")
	fileCount, err := countLines(filepath.Join(dataDir, cfg.Storage.FileSystemFile))
	if err != nil {
		fmt.Printf("无法统计文件系统白名单: %v\n", err)
		fileCount = 0
	}
	processCount, err := countLines(filepath.Join(dataDir, cfg.Storage.ProcessSystemFile))
	if err != nil {
		fmt.Printf("无法统计进程白名单: %v\n", err)
		processCount = 0
	}

	fmt.Printf("文件系统白名单: %d 条\n", fileCount)
	fmt.Printf("进程白名单: %d 条\n", processCount)

	dubFileCount, _ := countLines(filepath.Join(dataDir, cfg.Storage.DubiousFileListFile))
	dubProcCount, _ := countLines(filepath.Join(dataDir, cfg.Storage.DubiousProcessListFile))

	fmt.Printf("可疑文件列表: %d 条\n", dubFileCount)
	fmt.Printf("可疑进程列表: %d 条\n", dubProcCount)
	fmt.Println()
}

func countLines(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}

		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && line[0] != '#' {
			lineCount++
		}
	}

	return lineCount, scanner.Err()
}
