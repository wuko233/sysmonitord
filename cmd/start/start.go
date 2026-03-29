package start

import (
	"fmt"
	"os"
	"sysmonitord/internal/config"
	"sysmonitord/internal/scanner/process"
	"sysmonitord/pkg/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动系统监控守护服务",
	Long:  "sysmonitord start 命令用于启动系统监控守护服务，首次启动会进行全量扫描建立白名单。",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Log.Info("正在启动系统监控守护服务...")

		cfg, err := config.LoadConfig("./config.yaml")
		if err != nil {
			logger.Log.Error("加载配置文件失败", zap.Error(err))
			os.Exit(1)
		}

		logger.Log.Info("配置文件加载成功",
			zap.String("审计服务器地址", fmt.Sprintf("%s:%d", cfg.Audit.Server, cfg.Audit.Port)),
		)

		procs, err := process.ScanAllProcesses()
		if err != nil {
			logger.Log.Error("扫描进程失败", zap.Error(err))
			os.Exit(1)
		}

		logger.Log.Info("进程列表：")
		for i, p := range procs {
			if i >= 10 {
				logger.Log.Info("... (仅显示前10个进程)")
				break
			}
			logger.Log.Info(
				"进程信息",
				zap.Int32("pid", p.PID),
				zap.String("name", p.Name),
				zap.String("path", p.Path),
				zap.String("cmdline", p.Cmdline),
			)
		}
	},
}
