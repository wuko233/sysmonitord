package main

import (
	"os"
	"sysmonitord/cmd/safe"
	"sysmonitord/cmd/start"
	"sysmonitord/cmd/status"
	"sysmonitord/cmd/version"
	"sysmonitord/internal/config"
	"sysmonitord/pkg/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func getConfigPath() string {
	if _, err := os.Stat("./config.yaml"); err == nil {
		return "./config.yaml"
	}

	if _, err := os.Stat("/etc/sysmonitord/config.yaml"); err == nil {
		return "/etc/sysmonitord/config.yaml"
	}

	return "./config.yaml"
}

func main() {
	logger.InitLogger()
	defer logger.Sync()

	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		logger.Log.Error("加载配置文件失败", zap.Error(err))
		os.Exit(1)
	} else {
		logger.SetLogLevel(cfg.Log.Level)
	}

	var rootCmd = &cobra.Command{
		Use:   "sysmonitord",
		Short: "Sysmonitord 是一个 Linux 系统安全监控工具",
	}

	rootCmd.AddCommand(start.StartCmd)
	rootCmd.AddCommand(version.VersionCmd)
	rootCmd.AddCommand(status.StatusCmd)
	rootCmd.AddCommand(safe.SafeCmd)

	if err := rootCmd.Execute(); err != nil {
		logger.Log.Error("命令执行失败", zap.Error(err))
		os.Exit(1)
	}
}
