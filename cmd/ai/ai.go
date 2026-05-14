package ai

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	aiengine "sysmonitord/internal/ai"
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

			if err := validateAIConfig(cfg.AI); err != nil {
				fmt.Printf("AI 配置错误: %v\n", err)
				os.Exit(1)
			}

			fmt.Println()
			fmt.Printf("AI 模型: %s\n", cfg.AI.Model)
			fmt.Printf("AI 接口: %s\n", cfg.AI.APIURL)
			fmt.Printf("报告目录: %s\n", cfg.AI.ReportDir)
			fmt.Println()
			fmt.Println("计划读取的配置路径:")

			for _, path := range cfg.AI.IncludePaths {
				fmt.Printf("- %s\n", path)
			}

			fmt.Println()
			fmt.Println("正在读取配置文件并发送到 AI 进行分析...")

			result, err := aiengine.BuildPrompt(aiengine.CollectOptions{
				IncludePaths: cfg.AI.IncludePaths,
				MaxFileSize:  cfg.AI.MaxFileSize,
				MaxTotalSize: cfg.AI.MaxTotalSize,
			})
			if err != nil {
				fmt.Printf("生成 AI Prompt 失败: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("已生成 AI Prompt。")
			fmt.Printf("读取文件数: %d\n", result.FileCount)
			fmt.Printf("Prompt 字节数: %d\n", result.PromptBytes)
			if len(result.SkippedFiles) > 0 {
				fmt.Println()
				fmt.Println("以下文件未读取成功或被跳过：")
				for _, skipped := range result.SkippedFiles {
					fmt.Printf("- %s\n", skipped)
				}
			}
			fmt.Println()
			fmt.Println("正在调用 AI 接口进行分析...")

			client, err := aiengine.NewClient(aiengine.ClientConfig{
				APIURL:  cfg.AI.APIURL,
				APIKey:  cfg.AI.APIKey,
				Model:   cfg.AI.Model,
				Timeout: cfg.AI.Timeout,
			})
			if err != nil {
				fmt.Printf("创建 AI 客户端失败: %v\n", err)
				os.Exit(1)
			}

			analysis, err := client.Analyze(result.Prompt)
			if err != nil {
				fmt.Printf("AI 分析失败: %v\n", err)
				os.Exit(1)
			}

			reportPath, err := aiengine.SaveReport(cfg.AI.ReportDir, analysis)
			if err != nil {
				fmt.Printf("保存 AI 报告失败: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("AI 分析完成，报告已保存到: %s\n", reportPath)
		},
	}

	return cmd
}

func validateAIConfig(cfg config.AIConfig) error {
	if !cfg.Enabled {
		return fmt.Errorf("AI 功能未启用，请在 config.yaml 中设置 ai.enabled: true")
	}
	if strings.TrimSpace(cfg.APIURL) == "" {
		return fmt.Errorf("ai.api_url 不能为空")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return fmt.Errorf("ai.api_key 不能为空")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return fmt.Errorf("ai.model 不能为空")
	}
	if strings.TrimSpace(cfg.ReportDir) == "" {
		return fmt.Errorf("ai.report_dir 不能为空")
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("ai.timeout 必须大于 0")
	}
	if cfg.MaxFileSize <= 0 {
		return fmt.Errorf("ai.max_file_size 必须大于 0")
	}
	if cfg.MaxTotalSize <= 0 {
		return fmt.Errorf("ai.max_total_size 必须大于 0")
	}
	if len(cfg.IncludePaths) == 0 {
		return fmt.Errorf("ai.include_paths 至少需要配置一个路径")
	}

	return nil
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
