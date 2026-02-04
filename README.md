# 微信公众号文章预览工具

一个简单的 Golang Web 服务，用于预览 Markdown 文章并一键复制到微信公众号后台。

## 功能特性

- ✅ 自动扫描 `posts/` 目录下的所有 Markdown 文章
- ✅ 按系列分组展示文章列表
- ✅ 微信公众号风格样式（浅色主题，中文优化）
- ✅ 一键复制文章（HTML 格式，带样式）
- ✅ 响应式布局（PC + 移动端）
- ✅ 支持 GFM（表格、代码高亮、任务列表）

## 快速开始

### 1. 安装依赖

```bash
cd tools/wechat-preview
go mod tidy
```

### 2. 启动服务

```bash
go run main.go
```

### 3. 访问预览

打开浏览器访问：[http://localhost:8080](http://localhost:8080)

## 使用方法

1. **浏览文章列表**：首页按系列分组展示所有文章
2. **查看文章详情**：点击文章卡片进入详情页
3. **复制文章**：点击"复制文章"按钮
4. **粘贴到微信**：打开微信公众号后台，直接粘贴

## 注意事项

⚠️ **图片需要手动处理**

微信公众号不支持外部图片链接，复制后需要：
1. 在微信后台手动上传图片
2. 替换文章中的图片占位符

## 技术栈

- **后端**: Golang + Gin Framework
- **Markdown 渲染**: goldmark + GFM 扩展
- **前端**: 原生 HTML/CSS/JavaScript

## 项目结构

```
wechat-preview/
├── main.go              # 主程序（文章扫描、API 服务）
├── go.mod
├── go.sum
└── web/
    ├── templates/
    │   ├── list.html    # 文章列表页
    │   └── article.html # 文章详情页
    └── static/
        ├── css/
        │   └── wechat.css  # 微信风格样式
        └── js/
            └── copy.js     # 复制功能
```

## 样式特点

### 字体
- 中文：PingFang SC / Hiragino Sans GB / Microsoft YaHei
- 英文：-apple-system / BlinkMacSystemFont
- 代码：SF Mono / Monaco / Consolas

### 排版
- 字号：16px
- 行高：1.75（适合中文阅读）
- 段间距：15px
- 标题层级：H1(24px) / H2(20px) / H3(18px)

### 代码块
- 背景：#f6f8fa
- 边框圆角：6px
- 字体：等宽字体
- 语法高亮：通过 class 标记

### 引用块
- 左边框：4px 绿色 (#42b983)
- 背景：#f9f9f9
- 内边距：12px 16px

## 常见问题

### Q: 为什么图片复制后无法显示？
A: 微信公众号不支持外部图片链接，需要手动上传图片到微信服务器。

### Q: 可以修改样式吗？
A: 可以编辑 `web/static/css/wechat.css` 文件自定义样式。

### Q: 支持哪些 Markdown 语法？
A: 支持标准 Markdown + GFM 扩展（表格、删除线、任务列表、自动链接）。

### Q: 如何更改端口？
A: 修改 `main.go` 中的 `r.Run(":8080")` 为其他端口。

## License

MIT
