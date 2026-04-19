package status

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sysmonitord/internal/config"
	"sysmonitord/pkg/logger"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
	runtimeInfo := getRuntime()

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

func getRuntime() string {
	cmd := exec.Command("systemctl", "is-active", "sysmonitord")
	output, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) != "active" {
		return "N/A"
	}

	cmd = exec.Command("systemctl", "show", "sysmonitord", "--property=ActiveEnterTimestamp")
	output, err = cmd.Output()
	if err != nil {
		return "N/A"
	}

	parts := strings.SplitN(string(output), "=", 2)
	if len(parts) != 2 {
		return "N/A"
	}

	timestampStr := strings.TrimSpace(parts[1])
	if timestampStr == "" {
		return "N/A"
	}

	layouts := []string{
		"Mon 2006-01-02 15:04:05 MST",
		"Mon 2006-01-02 15:04:05",
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05",
		"Mon 2006-01-02 15:04:05 MST 2006",
	}

	var startTime time.Time
	var parseErr error
	for _, layout := range layouts {
		startTime, parseErr = time.Parse(layout, timestampStr)
		if parseErr == nil {
			logger.Log.Debug("时间解析成功", zap.String("layout", layout))
			break
		}
	}
	if parseErr != nil {
		return "N/A"
	}

	if time.Since(startTime) < 0 {
		return "N/A"
	}

	runtime := time.Since(startTime)

	days := int(runtime.Hours()) / 24
	hours := int(runtime.Hours()) % 24
	minutes := int(runtime.Minutes()) % 60
	seconds := int(runtime.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%d天 %d小时 %d分钟 %d秒", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时 %d分钟 %d秒", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%d分钟 %d秒", minutes, seconds)
	} else {
		return fmt.Sprintf("%d秒", seconds)
	}
}
