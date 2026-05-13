package ai

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sysmonitord/internal/config"

	"github.com/spf13/cobra"
)

func NewAICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "使用 AI 分析系统和应用配置文件并生成安全建议报告",
		Long:  "读取系统和应用配置文件，在用户确认后将配置文件路径和内容发送到 AI 模型，并生成安全改进建议报告。",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, ok := cmd.Context().Value("config").(*config.Config)
			if !ok {
				fmt.Println("无法获取配置")
				os.Exit(1)
			}

			if !confirmAIRisks() {
				fmt.Println("用户取消了 AI 分析")
				return
			}

			reportDir := filepath.Join(cfg.Storage.DataDir, "ai_reports")
			fmt.Println()
			fmt.Printf("正在分析配置文件并生成 AI 安全建议报告，报告将保存在: %s\n", reportDir)
			fmt.Println("请稍候...")
		},
	}

	return cmd
}

func confirmAIRisks() bool {
	fmt.Println("SysMonitord AI 安全分析")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("本功能将读取系统和应用配置文件，并将配置文件路径和内容明文发送到您配置的 AI 接口进行安全分析。")
	fmt.Println()
	fmt.Println("请在继续前确认您已理解以下风险：")
	fmt.Println("1. 配置文件中可能包含密码、Token、密钥、数据库连接串等敏感信息。")
	fmt.Println("2. 配置文件路径和配置文件内容会作为 Prompt 明文发送给 AI 模型。")
	fmt.Println("3. SysMonitord 不会保存原始配置文件内容。")
	fmt.Println("4. SysMonitord 只会保存 AI 返回的安全建议报告。")
	fmt.Println("5. AI 生成的报告仅供参考，修改系统配置前请先备份并由管理员确认。")
	fmt.Println("6. 请确认您有权限读取这些配置文件，并同意将其发送到您配置的 AI 服务。")
	fmt.Println()
	fmt.Print("是否继续？请输入 yes 确认：")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取输入失败: %v\n", err)
		return false
	}
	input = strings.TrimSpace(input)
	return input == "yes"
}
