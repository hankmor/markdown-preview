package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PublishResult 发布结果
type PublishResult struct {
	OriginalContent string
	PublishContent  string
	UploadedImages  []string
	Errors          []string
}

// PublishArticle 处理文章发布逻辑
// 1. 扫描 markdown 中的本地图片
// 2. 上传到 GitHub
// 3. 替换链接
func PublishArticle(postPath string, projectRoot string) (*PublishResult, error) {
	contentBytes, err := os.ReadFile(postPath)
	if err != nil {
		return nil, err
	}
	content := string(contentBytes)

	// 正则匹配图片: !\[.*\]\((.*)\)
	re := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
	matches := re.FindAllStringSubmatch(content, -1)

	log.Printf("Debug: Scanning article %s, found %d matches\n", postPath, len(matches))

	uploader := &GitHubUploader{}
	result := &PublishResult{
		OriginalContent: content,
	}

	// 替换映射表 (Local -> Remote)
	urlMap := make(map[string]string)

	// 去重处理
	uniquePaths := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		imgURL := match[2]

		log.Printf("Debug: Found image link: %s\n", imgURL)

		// 忽略网络图片
		if strings.HasPrefix(imgURL, "http") {
			log.Printf("Debug: Skipping remote image: %s\n", imgURL)
			continue
		}

		// 处理带 title 的情况
		parts := strings.Split(imgURL, " ")
		cleanPath := parts[0]

		// 解析本地绝对路径
		mdDir := filepath.Dir(postPath)
		absPath := filepath.Join(mdDir, cleanPath)

		log.Printf("Debug: Resolving local path. MD Dir: %s, Rel: %s -> Abs: %s\n", mdDir, cleanPath, absPath)

		// 确保文件存在
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			log.Printf("Debug: File not found at %s\n", absPath)
			result.Errors = append(result.Errors, fmt.Sprintf("Image not found: %s", cleanPath))
			continue
		}

		if uniquePaths[cleanPath] {
			continue
		}
		uniquePaths[cleanPath] = true

		// 构造远程路径
		// 策略：为了避免重名，可以使用 hash，或者保留目录结构
		// 这里简单起见，保留相对于 projectRoot 的路径
		// 例如: posts/02-openclaw/images/foo.png
		remotePath, _ := filepath.Rel(projectRoot, absPath)

		// 同步上传
		cdnURL, err := uploader.Upload(absPath, remotePath)

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Upload failed for %s: %v", cleanPath, err))
		} else {
			urlMap[cleanPath] = cdnURL
			result.UploadedImages = append(result.UploadedImages, cdnURL)
		}
	}

	// 替换内容
	newContent := content
	for local, remote := range urlMap {
		// 简单替换可能误伤，但既然我们是用正则提取出来的，应该问题不大
		// 更严谨的做法是按 index 替换

		// 注意：如果要替换 `(./images/foo.png)` 为 `(https://...)`
		// 简单的 ReplaceAll 足够应对绝大多数情况
		newContent = strings.ReplaceAll(newContent, "("+local, "("+remote)
	}

	result.PublishContent = newContent
	return result, nil
}
