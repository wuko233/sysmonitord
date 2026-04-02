package main

import (
	"os"
	"sysmonitord/cmd/start"
	"sysmonitord/cmd/version"
	"sysmonitord/pkg/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	logger.InitLogger()
	defer logger.Sync()

	var rootCmd = &cobra.Command{
		Use:   "sysmonitord",
		Short: "Sysmonitord 是一个 Linux 系统安全监控工具",
	}

	rootCmd.AddCommand(start.StartCmd)
	rootCmd.AddCommand(version.VersionCmd)

	if err := rootCmd.Execute(); err != nil {
		logger.Log.Error("命令执行失败", zap.Error(err))
		os.Exit(1)
	}
}
