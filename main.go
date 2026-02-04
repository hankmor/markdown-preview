package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/hankmor/mymedia/tools/wechat-preview/config"
	"github.com/hankmor/mymedia/tools/wechat-preview/services"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// Article 文章元数据
type Article struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Series    string    `json:"series"`
	Path      string    `json:"path"`
    RelPath   string    `json:"relPath"` // 相对 posts 的路径，用于定位图片
	UpdatedAt time.Time `json:"updatedAt"`
}

// ArticleDetail 文章详情
type ArticleDetail struct {
	Article
	HTML        string `json:"html"`
	RawMarkdown string `json:"rawMarkdown"`
}

var (
	postsDir = "../../posts" // 相对于 tools/wechat-preview 的路径
    projectRoot = "../../"
	articles []Article
	md       goldmark.Markdown
)

func init() {
	// 初始化 Markdown 解析器
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,   // GitHub Flavored Markdown
			extension.Table, // 表格
			extension.Strikethrough,
			extension.TaskList,
            highlighting.NewHighlighting(
                highlighting.WithStyle("monokai"), // 使用高对比度主题
                highlighting.WithFormatOptions(
                    chromahtml.WithLineNumbers(false), // 微信里行号可能样式混乱，先关闭
                ),
            ),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
            html.WithUnsafe(), // 允许 HTML 标签
		),
	)
}

func main() {
    config.Load() // 加载配置

	// 扫描文章
	if err := scanArticles(); err != nil {
		fmt.Printf("扫描文章失败: %v\n", err)
		os.Exit(1)
	}
    
    fmt.Println("\n========================================")
    fmt.Println("   Wechat Preview Tool - v2.1 Debug Mode")
    fmt.Printf("   Articles: %d\n", len(articles))
    fmt.Println("========================================\n")

	// 初始化 Gin
	r := gin.Default()

	// 静态文件服务
	r.Static("/static", "./web/static")
    
    // 核心修改：映射 posts 目录为静态资源，用于预览本地图片
    // 访问 /posts-static/02-openclaw/images/foo.png -> ../../posts/02-openclaw/images/foo.png
    r.Static("/posts-static", postsDir)
    
	r.LoadHTMLGlob("web/templates/*")

	// 路由
	r.GET("/", handleList)
	r.GET("/article/:id", handleArticle)
	r.GET("/api/articles", apiArticles)
	r.GET("/api/articles/:id", apiArticleDetail)
    r.POST("/api/publish/:id", handlePublish) // 新增发布接口

	// 启动服务
	fmt.Println("服务启动成功！访问 http://localhost:8080")
	r.Run(":8080")
}

// scanArticles 扫描文章目录
func scanArticles() error {
	articles = []Article{}

	return filepath.WalkDir(postsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理 .md 文件
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// 读取文件获取标题
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		title := extractTitle(string(content))
		if title == "" {
			title = filepath.Base(path)
		}

		// 提取系列名（从目录结构）
		relPath, _ := filepath.Rel(postsDir, path)
		parts := strings.Split(relPath, string(os.PathSeparator))
		series := "其他"
		if len(parts) > 1 {
			series = parts[0]
		}

		// 获取修改时间
		info, _ := d.Info()
		updatedAt := time.Now()
		if info != nil {
			updatedAt = info.ModTime()
		}

		// 生成 ID（去掉 posts/ 前缀和 .md 后缀，替换路径分隔符为下划线）
		id := strings.TrimSuffix(relPath, ".md")
		id = strings.ReplaceAll(id, string(os.PathSeparator), "_")

		articles = append(articles, Article{
			ID:        id,
			Title:     title,
			Series:    series,
			Path:      path,
            RelPath:   relPath,
			UpdatedAt: updatedAt,
		})

		return nil
	})
}

// extractTitle 从 Markdown 内容提取标题
func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

// removeTitle 移除内容中的第一个 H1 标题
func removeTitle(content string) string {
    lines := strings.Split(content, "\n")
    var newLines []string
    removed := false
    for _, line := range lines {
        if !removed && strings.HasPrefix(strings.TrimSpace(line), "# ") {
            removed = true
            continue
        }
        newLines = append(newLines, line)
    }
    return strings.Join(newLines, "\n")
}

// handleList 文章列表页面
func handleList(c *gin.Context) {
	// 按系列分组
	grouped := make(map[string][]Article)
	for _, article := range articles {
		grouped[article.Series] = append(grouped[article.Series], article)
	}

	c.HTML(200, "list.html", gin.H{
		"groupedArticles": grouped,
	})
}

// handleArticle 文章详情页面
func handleArticle(c *gin.Context) {
    fmt.Println(">>> Entering handleArticle")
	id := c.Param("id")

	// 查找文章
	var article *Article
	for i := range articles {
		if articles[i].ID == id {
			article = &articles[i]
			break
		}
	}

	if article == nil {
		c.String(404, "文章不存在")
		return
	}

	// 读取并渲染 Markdown
	content, err := os.ReadFile(article.Path)
	if err != nil {
		c.String(500, "读取文章失败")
		return
	}

    // 移除标题 (H1)
    markdownContent := removeTitle(string(content))

	var buf strings.Builder
	if err := md.Convert([]byte(markdownContent), &buf); err != nil {
		c.String(500, "渲染文章失败")
		return
	}

	// 1. 列表项优化 (strong -> span, li wrap)
	htmlContent := buf.String()
	
	// 先替换 strong 为 span class="li-bold"
	reStrong := regexp.MustCompile(`<li><strong>([^<]+)</strong>`)
	htmlContent = reStrong.ReplaceAllString(htmlContent, `<li><span class="li-bold">$1</span>`)
	
	// 将 li 内部的所有内容包裹在 <span class="li-text"> 中
	reLiContent := regexp.MustCompile(`<li>(.*?)</li>`)
	htmlContent = reLiContent.ReplaceAllString(htmlContent, `<li><span class="li-text">$1</span></li>`)

    // 2. 本地图片路径修正 (仅用于预览)
    // 假设图片引用是 relative path: ![](./images/foo.png) or ![](images/foo.png)
    // 需要替换为 /posts-static/<article-dir>/images/foo.png
    articleDir := filepath.Dir(article.RelPath)
    
    // 调试日志：打印文章目录
    fmt.Printf("Debug: Article Dir for %s is %s\n", article.ID, articleDir)

    // 正则匹配 img src，排除 http 开头的
    // 匹配 src="xxx" 或 src='xxx'
    reImg := regexp.MustCompile(`src=["']([^"']+)["']`)
    
    // 调试：打印替换前的部分内容
    if len(htmlContent) > 200 {
         fmt.Printf("Debug: Before replace (snippet): %s\n", htmlContent[:200])
    }
    
    htmlContent = reImg.ReplaceAllStringFunc(htmlContent, func(match string) string {
        // match: src="./images/foo.png"
        fmt.Printf("Debug: Matched img tag: %s\n", match)
        
        // 提取引号内的内容
        parts := strings.SplitN(match, "=", 2)
        if len(parts) != 2 {
            return match
        }
        quote := parts[1][0:1]
        src := parts[1][1 : len(parts[1])-1]

        // 忽略绝对路径和网络路径
        if strings.HasPrefix(src, "/") || strings.HasPrefix(src, "http") {
             fmt.Printf("Debug: Skipping abs/remote path: %s\n", src)
            return match
        }
        
        // 拼接静态资源前缀
        cleanDir := strings.ReplaceAll(articleDir, string(os.PathSeparator), "/")
        
        // 处理 ./ 前缀
        cleanSrc := src
        if strings.HasPrefix(src, "./") {
            cleanSrc = src[2:]
        }
        
        newSrc := fmt.Sprintf("/posts-static/%s/%s", cleanDir, cleanSrc)
        newTag := fmt.Sprintf(`src=%s%s%s`, quote, newSrc, quote)
        
        fmt.Printf("Debug: Replaced to: %s\n", newTag)
        return newTag
    })
    
    // 调试：打印替换后的部分内容
    // 查找是否包含 posts-static
    if strings.Contains(htmlContent, "/posts-static/") {
        fmt.Printf("Debug: Success! Found /posts-static/ in HTML\n")
    } else {
        fmt.Printf("Debug: Warning! /posts-static/ NOT found in HTML\n")
    }

	c.HTML(200, "article.html", gin.H{
		"title":  article.Title,
		"html":   template.HTML(htmlContent),
		"id":     article.ID,
		"series": article.Series,
	})
}

// handlePublish 处理发布请求
func handlePublish(c *gin.Context) {
    id := c.Param("id")
    var article *Article
	for i := range articles {
		if articles[i].ID == id {
			article = &articles[i]
			break
		}
	}
    if article == nil {
		c.JSON(404, gin.H{"error": "文章不存在"})
		return
	}
    
    // 调用发布服务
    // projectRoot 需要绝对路径? or relative is fine
    // 我们用 .. 
    result, err := services.PublishArticle(article.Path, projectRoot)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 渲染 Markdown 为 HTML 供复制
    var buf strings.Builder
    // 移除标题
    publishContent := removeTitle(result.PublishContent)
    md.Convert([]byte(publishContent), &buf)
    
    // 同样应用列表项优化
    htmlContent := buf.String()
	reStrong := regexp.MustCompile(`<li><strong>([^<]+)</strong>`)
	htmlContent = reStrong.ReplaceAllString(htmlContent, `<li><span class="li-bold">$1</span>`)
	reLiContent := regexp.MustCompile(`<li>(.*?)</li>`)
	htmlContent = reLiContent.ReplaceAllString(htmlContent, `<li><span class="li-text">$1</span></li>`)
    
    c.JSON(200, gin.H{
        "success": true,
        "content": map[string]string{
            "markdown": result.PublishContent,
            "html": htmlContent, // 返回已处理的 HTML
        },
        "uploaded": result.UploadedImages,
        "logs": result.Errors,
    })
}

// apiArticles API: 文章列表
func apiArticles(c *gin.Context) {
	c.JSON(200, articles)
}

// apiArticleDetail API: 文章详情
func apiArticleDetail(c *gin.Context) {
	id := c.Param("id")

	// 查找文章
	var article *Article
	for i := range articles {
		if articles[i].ID == id {
			article = &articles[i]
			break
		}
	}

	if article == nil {
		c.JSON(404, gin.H{"error": "文章不存在"})
		return
	}

	// 读取内容
	content, err := os.ReadFile(article.Path)
	if err != nil {
		c.JSON(500, gin.H{"error": "读取文章失败"})
		return
	}

	// 渲染 HTML
	var buf strings.Builder
	if err := md.Convert(content, &buf); err != nil {
		c.JSON(500, gin.H{"error": "渲染文章失败"})
		return
	}

	c.JSON(200, ArticleDetail{
		Article:     *article,
		HTML:        buf.String(),
		RawMarkdown: string(content),
	})
}

