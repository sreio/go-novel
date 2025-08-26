
# go-novel

## 1. 项目简介

`go-novel` 是一个基于 **Golang 1.25** 重写的开源小说下载与阅读工具，支持 **CLI 命令行模式**与 **Web 界面模式**。
项目目标：通过配置化的书源解析规则，实现多站点搜索、章节抓取、批量下载，并支持多种导出格式（TXT、EPUB、PDF）。

特点：

* 动态书源配置（YAML 格式，易于扩展新站点）
* 并发抓取（支持限速、重试、转码、代理）
* CLI / Web / API 三种运行模式
* 导出 EPUB / PDF / TXT 格式
* GitHub Actions 自动构建 Release 与 Docker 镜像

---

## 2. 技术栈

### 后端

* **语言**：Golang 1.25
* **Web 框架**：[chi](https://github.com/go-chi/chi)
* **并发控制**：`errgroup` + `rate limiter`
* **HTML 解析**：goquery
* **导出**：

    * TXT：原生文件写入
    * EPUB：[go-epub](https://github.com/bmaupin/go-epub)
    * PDF：[gofpdf](https://github.com/jung-kurt/gofpdf)

### 前端

* **框架**：Vue 3
* **构建工具**：Vite
* **UI 库**：Element Plus
* **网络请求**：Axios

### 开发运行命令

```bash
# 后端 CLI
go run ./cmd/sonovel-cli --help

# 后端 Web
go run ./cmd/sonovel-web

# 前端
cd web
npm install
npm run dev   # http://localhost:5173
```

---

## 3. 运行模式

### CLI 模式

直接运行命令行下载：

```bash
# 搜索小说
sonovel-cli search --keyword "遮天"

# 下载小说（支持 txt/epub/pdf）
sonovel-cli download --url "https://example.com/book/123.html" --format epub --out book.epub
```

### Web 模式

```bash
# 启动后端
go run ./cmd/sonovel-web
# 默认监听 http://localhost:8080

# 启动前端
cd web && npm run dev
# 前端代理到后端，访问 http://localhost:5173
```

在浏览器中即可：

* 搜索小说
* 预览章节（抽屉显示）
* 下载 TXT/EPUB/PDF

### API 模式

所有 Web 页面请求均基于 API：

* `GET /api/search?q=关键词` 搜索
* `GET /api/books/chapters?url=目录页URL` 获取章节目录
* `GET /api/chapter?url=章节URL` 获取单章内容
* `GET /api/download?url=目录页URL&format=txt|epub|pdf` 下载整本书


---

## 4. 书源配置

书源位于 `configs/sources/`，使用 YAML 格式描述。
示例（dxmwx，目录页需由 bookURL → chapterURL 映射）：

```yaml
id: dxmwx
name: 大熊猫文学网
base_url: "https://www.dxmwx.org"
charset: "utf-8"

search:
  path: "/search.php"
  param: "q"
  item_selector: ".bookbox"
  title_selector: ".bookinfo h4 a"
  author_selector: ".bookinfo .author"
  link_selector: ".bookinfo h4 a"
  link_attr: "href"
  update_selector: ".bookinfo .update"
  category_selector: ".bookinfo .cat"

chapters:
  toc:
    url_template: "https://www.dxmwx.org/chapter/{{id}}.html"
    id_from_url_regex: "/book/(\\d+)\\.html"
  list_selector: ".listmain dd:not(:first-child)"
  title_selector: "a"
  url_selector: "a"
  url_attr: "href"

content:
  selector: "#content"
```

常用参数：

* `item_selector`：搜索结果列表
* `title_selector`：标题选择器
* `author_selector`：作者选择器
* `link_selector`：详情页链接
* `list_selector`：章节列表选择器
* `toc.url_template`：目录页 URL 模板（可用 `{{id}}` 占位符）
* `toc.id_from_url_regex`：正则从详情页 URL 提取 ID
* `content.selector`：正文内容选择器

---

## 5. 免责声明

1. 本项目仅供 **学习与研究爬虫/解析技术** 之用。
2. 请勿将本项目用于任何 **商业用途**，亦请遵守相关网站的服务条款与版权声明。
3. 小说及相关内容版权归原作者及发布网站所有。
4. 使用本项目下载的任何内容，责任由使用者自行承担。

