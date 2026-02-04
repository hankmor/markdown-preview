package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	GitHubToken  string
	GitHubRepo   string // e.g., "username/repo"
	GitHubBranch string // default "main"
}

var AppConfig *Config

func Load() {
	// 尝试加载 .env 文件，如果不存在也不报错（可能通过系统环境变量注入）
	_ = godotenv.Load()

	AppConfig = &Config{
		GitHubToken:  os.Getenv("GITHUB_TOKEN"),
		GitHubRepo:   os.Getenv("GITHUB_REPO"),
		GitHubBranch: os.Getenv("GITHUB_BRANCH"),
	}

	// 自动去除 .git 后缀
	if before, ok :=strings.CutSuffix(AppConfig.GitHubRepo, ".git"); ok  {
		AppConfig.GitHubRepo = before
	}

	if AppConfig.GitHubBranch == "" {
		AppConfig.GitHubBranch = "main"
	}

	if AppConfig.GitHubToken == "" {
		log.Println("⚠️  Warning: GITHUB_TOKEN not found. Upload feature will be disabled.")
	}
}
