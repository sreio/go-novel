# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目架构

这是一个Go语言开发的小说下载工具，包含：
- **后端**：Go 1.25 + chi框架，提供CLI和Web API两种模式
- **前端**：Vue 3 + Vite + Element Plus
- **书源配置**：YAML格式配置文件在 `configs/sources/`

核心模块：
- `internal/sources/` - 书源加载和解析逻辑
- `internal/format/` - 导出格式支持 (TXT/EPUB/PDF)
- `cmd/sonovel-cli/` - 命令行接口
- `cmd/sonovel-web/` - Web服务接口
- `web/` - 前端Vue应用

## 开发命令

### 后端构建和运行
```bash
# 构建CLI工具
go build -o sonovel-cli ./cmd/sonovel-cli

# 构建Web服务
go build -o sonovel-web ./cmd/sonovel-web

# 运行CLI
go run ./cmd/sonovel-cli --help

# 运行Web服务 (监听8080端口)
go run ./cmd/sonovel-web
```

### 前端开发
```bash
cd web
npm install
npm run dev        # 开发服务器 (http://localhost:5173)
npm run build     # 生产构建
npm run preview   # 预览生产构建
```

### 跨平台构建
```bash
# Linux
go build -ldflags "-s -w" -o sonovel-cli-linux ./cmd/sonovel-cli

# Windows
go build -ldflags "-s -w" -o sonovel-cli.exe ./cmd/sonovel-cli

# macOS
go build -ldflags "-s -w" -o sonovel-cli-darwin ./cmd/sonovel-cli
```

## API接口
Web服务提供以下REST API：
- `GET /api/search?q=关键词` - 搜索小说
- `GET /api/books/chapters?url=目录页URL` - 获取章节目录
- `GET /api/chapter?url=章节URL` - 获取单章内容
- `GET /api/download?url=目录页URL&format=txt|epub|pdf` - 下载整本书

## 书源配置
书源文件位于 `configs/sources/*.yaml`，使用YAML格式定义网站解析规则，支持搜索、目录解析、内容提取等功能。