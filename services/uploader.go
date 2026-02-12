package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hankmor/mymedia/tools/wechat-preview/config"
)

type GitHubUploader struct{}

// Upload 上传文件到 GitHub
// filePath: 本地文件绝对路径
// remotePath: 仓库内目标路径，例如 "images/2024/01/foo.png"
func (u *GitHubUploader) Upload(filePath, remotePath string) (string, error) {
	if config.AppConfig.GitHubToken == "" || config.AppConfig.GitHubRepo == "" {
		log.Printf("Error: GitHub config missing. Token len: %d, Repo: %s\n", len(config.AppConfig.GitHubToken), config.AppConfig.GitHubRepo)
		return "", fmt.Errorf("GitHub configuration missing")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// 规范化 remotePath，GitHub API 要求 / 分隔符
	remotePath = filepath.ToSlash(remotePath)

	// 1. 检查文件是否存在
	fileURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s",
		config.AppConfig.GitHubRepo, remotePath, config.AppConfig.GitHubBranch)

	log.Printf("Debug: Checking exist %s\n", fileURL)

	req, _ := http.NewRequest("GET", fileURL, nil)
	req.Header.Set("Authorization", "token "+config.AppConfig.GitHubToken)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err == nil && resp.StatusCode == 200 {
		// 文件已存在
		log.Printf("Debug: File exists, skipping upload: %s\n", remotePath)
		resp.Body.Close()
		return u.getCDNUrl(remotePath), nil
	}
	if resp != nil {
		log.Printf("Debug: File check status: %d\n", resp.StatusCode)
		resp.Body.Close()
	}

	// 2. 上传文件 (PUT)
	encContent := base64.StdEncoding.EncodeToString(content)

	// 构造请求体
	body := map[string]string{
		"message": "Upload image via wechat-preview tool",
		"content": encContent,
		"branch":  config.AppConfig.GitHubBranch,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ = http.NewRequest("PUT", fileURL, bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "token "+config.AppConfig.GitHubToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		// 并发冲突处理：如果是 409 或 422，再次检查文件是否已存在
		if resp.StatusCode == 409 || resp.StatusCode == 422 {
			log.Printf("Debug: Upload conflict (%d), re-checking file existence: %s\n", resp.StatusCode, remotePath)

			checkReq, _ := http.NewRequest("GET", fileURL, nil)
			checkReq.Header.Set("Authorization", "token "+config.AppConfig.GitHubToken)
			checkResp, checkErr := client.Do(checkReq)
			if checkErr == nil {
				defer checkResp.Body.Close()
				if checkResp.StatusCode == 200 {
					log.Printf("Debug: File actually exists after conflict: %s\n", remotePath)
					return u.getCDNUrl(remotePath), nil
				}
			}
		}

		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Error: Upload failed. Status: %d, Response: %s\n", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("upload failed: %s", string(respBody))
	}

	log.Printf("Debug: Upload success for %s\n", remotePath)

	return u.getCDNUrl(remotePath), nil
}

func (u *GitHubUploader) getCDNUrl(remotePath string) string {
	// 使用 jsDelivr 加速
	// 格式: https://cdn.jsdelivr.net/gh/user/repo@branch/file
	return fmt.Sprintf("https://cdn.jsdelivr.net/gh/%s@%s/%s",
		config.AppConfig.GitHubRepo, config.AppConfig.GitHubBranch, remotePath)
}
