package start

import (
	"fmt"
	"os"
	"sysmonitord/internal/config"

	"github.com/spf13/cobra"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动系统监控守护服务",
	Long:  "sysmonitord start 命令用于启动系统监控守护服务，首次启动会进行全量扫描建立白名单。",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("正在启动系统监控守护服务...")

		cfg, err := config.LoadConfig("./config.yaml")
		if err != nil {
			fmt.Println("加载配置文件失败:", err)
			os.Exit(1)
		}

		fmt.Println("配置文件加载成功")
		fmt.Printf("审计服务器地址：%s:%d\n", cfg.Audit.Server, cfg.Audit.Port)

		// Todo: 初始化扫描
	},
}
