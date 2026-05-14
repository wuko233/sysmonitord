package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ClientConfig struct {
	APIURL  string
	APIKey  string
	Model   string
	Timeout int
}

type Client struct {
	config     ClientConfig
	httpClient *http.Client
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func NewClient(config ClientConfig) (*Client, error) {
	if strings.TrimSpace(config.APIURL) == "" {
		return nil, fmt.Errorf("APIURL为空")
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, fmt.Errorf("APIKey为空")
	}
	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("Model为空")
	}
	if config.Timeout <= 0 {
		config.Timeout = 120
	}
	config.APIURL = strings.TrimSpace(config.APIURL)
	config.APIKey = strings.TrimSpace(config.APIKey)
	config.Model = strings.TrimSpace(config.Model)

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}, nil
}

func (c *Client) Analyze(prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("Prompt不能为空")
	}

	reqBody := chatCompletionRequest{
		Model: c.config.Model,
		Messages: []chatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.2,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("请求体序列化失败: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.config.APIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return "", fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
		}
		return "", fmt.Errorf("请求失败，状态码: %d, 错误信息: %s", resp.StatusCode, errResp.Error.Message)
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("响应解析失败: %v", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("响应中没有返回结果")
	}
	return chatResp.Choices[0].Message.Content, nil
}

func parseErrorResponse(statusCode int, body []byte) error {
	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("请求失败，状态码: %d", statusCode)
	}
	return fmt.Errorf("请求失败，状态码: %d, 错误信息: %s", statusCode, errResp.Error.Message)
}
