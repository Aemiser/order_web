package httputils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type StreamCallback func(content string) error

func AiRequest(apiKey, APIURL string, jsonData []byte, timeout int64, headmap map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), "POST", APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败：%w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 添加额外的请求头
	if headmap != nil {
		for k, v := range headmap {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败：%w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}
	return body, err
}

func AiStreamRequest(apiKey, APIURL string, jsonData []byte, timeout int64, headmap map[string]string, callback StreamCallback) error {
	req, err := http.NewRequestWithContext(context.Background(), "POST", APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败：%w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	if headmap != nil {
		for k, v := range headmap {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败：%w", err)
	}
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	accumulated := ""

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			accumulated += string(buf[:n])

			for strings.Contains(accumulated, "\n") {
				lines := strings.SplitN(accumulated, "\n", 2)
				if len(lines) < 2 {
					break
				}
				line := strings.TrimSpace(lines[0])
				accumulated = lines[1]

				if strings.HasPrefix(line, "data:") {
					data := strings.TrimPrefix(line, "data:")
					data = strings.TrimSpace(data)

					if strings.HasPrefix(data, "[") || strings.HasPrefix(data, "{") {
						if callback != nil {
							if err := callback(data); err != nil {
								return err
							}
						}
					} else if data == "[DONE]" {
						if callback != nil {
							callback("DONE")
						}
					}
				}
			}
		}

		if readErr != nil {
			break
		}
	}

	return nil
}
