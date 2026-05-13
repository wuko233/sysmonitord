package ai

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type CollectResult struct {
	Prompt       string
	FileCount    int
	PromptBytes  int
	SkippedFiles []string
}

type CollectOptions struct {
	IncludePaths []string
	MaxFileSize  int64
	MaxTotalSize int64
}

func BuildPrompt(options CollectOptions) (*CollectResult, error) {
	if len(options.IncludePaths) == 0 {
		return nil, fmt.Errorf("路径列表不能为空，请至少配置一个路径")
	}

	if options.MaxFileSize <= 0 {
		return nil, fmt.Errorf("最大文件大小必须大于 0")
	}

	if options.MaxTotalSize <= 0 {
		return nil, fmt.Errorf("最大总大小必须大于 0")
	}

	result := &CollectResult{
		SkippedFiles: make([]string, 0),
	}

	var builder bytes.Buffer

	writePromptHeader(&builder)

	var totalSize int64

	for _, includePath := range options.IncludePaths {
		includePath = strings.TrimSpace(includePath)
		if includePath == "" {
			continue
		}

		if err := collectPath(includePath, options, &builder, &totalSize, result); err != nil {
			result.SkippedFiles = append(result.SkippedFiles, fmt.Sprintf("%s: %v", includePath, err))
		}
	}

	writePromptFooter(&builder)

	result.Prompt = builder.String()
	result.PromptBytes = len([]byte(result.Prompt))

	return result, nil
}

func writePromptHeader(builder *bytes.Buffer) {
	builder.WriteString("你是 Linux 系统安全审计专家。\n")
	builder.WriteString("请分析以下系统和应用配置文件，输出一份 Markdown 格式的安全改进建议报告。\n\n")

	builder.WriteString("请重点关注：\n")
	builder.WriteString("1. 是否存在弱安全配置。\n")
	builder.WriteString("2. 是否存在可能导致未授权访问的配置。\n")
	builder.WriteString("3. 是否存在暴露敏感信息的配置。\n")
	builder.WriteString("4. 是否存在不符合 Linux 服务器安全最佳实践的配置。\n")
	builder.WriteString("5. 请给出风险等级、问题说明、影响范围和修复建议。\n\n")

	builder.WriteString("输出要求：\n")
	builder.WriteString("1. 使用中文输出。\n")
	builder.WriteString("2. 使用 Markdown 格式。\n")
	builder.WriteString("3. 不要调用任何工具。\n")
	builder.WriteString("4. 不要编造不存在的配置项。\n")
	builder.WriteString("5. 如果证据不足，请明确说明“未从提供的配置中发现”。\n")
	builder.WriteString("6. 修改建议必须谨慎，避免影响业务运行。\n\n")

	builder.WriteString("以下是配置文件路径和明文内容：\n\n")
}

func writePromptFooter(builder *bytes.Buffer) {
	builder.WriteString("\n请基于以上内容生成安全分析报告。\n")
}

func collectPath(path string, options CollectOptions, builder *bytes.Buffer, totalSize *int64, result *CollectResult) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return collectDirectory(path, options, builder, totalSize, result)
	}

	return collectFile(path, info, options, builder, totalSize, result)
}

func collectDirectory(root string, options CollectOptions, builder *bytes.Buffer, totalSize *int64, result *CollectResult) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.SkippedFiles = append(result.SkippedFiles, fmt.Sprintf("%s: %v", path, walkErr))
			return nil
		}

		if entry.IsDir() {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			result.SkippedFiles = append(result.SkippedFiles, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		if !info.Mode().IsRegular() {
			result.SkippedFiles = append(result.SkippedFiles, fmt.Sprintf("%s: not a regular file", path))
			return nil
		}

		if err := collectFile(path, info, options, builder, totalSize, result); err != nil {
			result.SkippedFiles = append(result.SkippedFiles, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		return nil
	})
}

func collectFile(path string, info os.FileInfo, options CollectOptions, builder *bytes.Buffer, totalSize *int64, result *CollectResult) error {
	if info.Size() > options.MaxFileSize {
		return fmt.Errorf("文件大小 %d 超出最大文件大小 %d", info.Size(), options.MaxFileSize)
	}

	if *totalSize+info.Size() > options.MaxTotalSize {
		return fmt.Errorf("总大小超出最大总大小 %d", options.MaxTotalSize)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	*totalSize += int64(len(content))

	builder.WriteString("===== FILE: ")
	builder.WriteString(path)
	builder.WriteString(" =====\n")
	builder.Write(content)
	builder.WriteString("\n===== END FILE: ")
	builder.WriteString(path)
	builder.WriteString(" =====\n\n")

	result.FileCount++

	return nil
}
