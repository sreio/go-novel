# go-novel (Go 1.25)

重写自 so-novel，支持：CLI + Web(Vue3+Vite)、TXT/EPUB/PDF 导出、动态书源、HTTP 限速重试转码代理、并发抓取、GitHub Actions 打包 Release + 多平台 Docker 镜像。

## 快速开始

```bash
# 构建 CLI
go build -o bin/sonovel ./cmd/sonovel-cli

# 运行 Web（默认端口 8080）
go build -o bin/sonovel-web ./cmd/sonovel-web
./bin/sonovel-web
# 前端
cd web && npm ci && npm run dev
```

## 配置书源

在 `configs/sources/` 放置 YAML，参考 `example.yaml`。

## API
- `GET /api/search?q=关键词`
- `GET /api/books/chapters?url=书籍详情页URL`
- `GET /api/chapter?url=章节URL&limit=1000&full=0|1`
- `GET /api/download?url=书籍详情页URL&format=txt|epub|pdf`
