package main

import (
	"context"
	"fmt"
	"os"
	"sysmonitord/cmd/ai"
	"sysmonitord/cmd/safe"
	"sysmonitord/cmd/start"
	"sysmonitord/cmd/status"
	"sysmonitord/cmd/version"
	"sysmonitord/internal/config"
	"sysmonitord/pkg/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cfgFile string
	cfg     *config.Config
)

func main() {
	logger.InitLogger()
	defer logger.Sync()

	var rootCmd = &cobra.Command{
		Use:   "sysmonitord",
		Short: "Sysmonitord 是一个 Linux 系统安全监控工具",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cfgFile == "" {
				if _, err := os.Stat("./config.yaml"); err == nil {
					cfgFile = "./config.yaml"
				} else if _, err := os.Stat("/etc/sysmonitord/config.yaml"); err == nil {
					cfgFile = "/etc/sysmonitord/config.yaml"
				}

				fmt.Println("未指定配置文件，使用配置文件:", cfgFile)
			}

			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("加载配置文件失败: %w", err)
			}

			ctx := context.WithValue(cmd.Context(), "config", cfg)
			cmd.SetContext(ctx)
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "配置文件路径 (默认: ./config.yaml 或 /etc/sysmonitord/config.yaml)")

	rootCmd.AddCommand(start.NewStartCmd())
	rootCmd.AddCommand(version.NewVersionCmd())
	rootCmd.AddCommand(status.NewStatusCmd())
	rootCmd.AddCommand(safe.NewSafeCmd())
	rootCmd.AddCommand(ai.NewAICmd())

	if err := rootCmd.Execute(); err != nil {
		logger.Log.Error("命令执行失败", zap.Error(err))
		os.Exit(1)
	}
}
