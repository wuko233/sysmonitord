package version

import (
	"fmt"
	"sysmonitord/internal/version"

	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示 sysmonitord 的版本信息",
	Long:  "sysmonitord version 命令用于显示当前 sysmonitord 的版本、Git 提交信息和构建时间。",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Info())
	},
}
