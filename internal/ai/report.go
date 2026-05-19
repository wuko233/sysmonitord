package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func SaveReport(reportDir string, content string) (string, error) {
	if reportDir == "" {
		return "", fmt.Errorf("报告目录不能为空")
	}
	if content == "" {
		return "", fmt.Errorf("报告内容不能为空")
	}

	if err := os.MkdirAll(reportDir, 0750); err != nil {
		return "", fmt.Errorf("创建目录失败")
	}

	fileName := fmt.Sprintf("ai_report_%s.md", time.Now().Format("20020102_150405"))
	filePath := filepath.Join(reportDir, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return "", fmt.Errorf("写入文件失败")
	}
	return filePath, nil
}
