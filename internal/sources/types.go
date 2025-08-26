package sources

import (
	"net/url"
)

type RateConfig struct {
	RPS   float64 `yaml:"rps"`
	Burst int     `yaml:"burst"`
}

type SearchConfig struct {
	// 方式一：新方式（推荐）
	Path  string `yaml:"path"`  // 例如: /search.php
	Param string `yaml:"param"` // 例如: q
	// 方式二：旧/模板方式（可选）
	URLTemplate string `yaml:"url"` // 例如: https://site/search.php?q={{query}}

	// 额外 query 参数（可选）
	ExtraParams map[string]string `yaml:"extra_params"` // 如 { s: "1", ie: "utf-8" }

	ItemSelector     string `yaml:"item_selector"` // 列表项选择器
	TitleSelector    string `yaml:"title_selector"`
	AuthorSelector   string `yaml:"author_selector"`
	LinkSelector     string `yaml:"link_selector"`     // a[href]
	LinkAttr         string `yaml:"link_attr"`         // 默认 href
	UpdateSelector   string `yaml:"update_selector"`   // 列表项更新时间选择器，如 .update
	CategorySelector string `yaml:"category_selector"` // 小说分类选择器
}

type ChaptersConfig struct {
	ListSelector  string `yaml:"list_selector"`
	TitleSelector string `yaml:"title_selector"`
	URLSelector   string `yaml:"url_selector"`
	URLAttr       string `yaml:"url_attr"` // 默认 href

	// 新增：分页
	Pagination struct {
		// 方式一：基于“下一页”链接
		NextSelector string `yaml:"next_selector"` // 例：".pagination a.next"
		NextAttr     string `yaml:"next_attr"`     // 默认 href

		// 方式二：基于页码参数（URL 模式）
		// 如果 NextSelector 为空、但提供了 PageParam/StartPage 等，则使用页码模式：
		PageParam  string `yaml:"page_param"`   // 例：page
		StartPage  int    `yaml:"start_page"`   // 默认 1
		MaxPages   int    `yaml:"max_pages"`    // 安全上限，默认 10
		StopOnSame bool   `yaml:"stop_on_same"` // 如果新页数据与上一页相同则停止
	} `yaml:"pagination"`

	// 目录页推导配置
	TOC struct {
		// 目录页 URL 模板，例如 https://www.dxmwx.org/chapter/{{id}}.html
		URLTemplate string `yaml:"url_template"`

		// 从“详情页 URL”里用正则抓取 ID，例如 /book/(\d+)\.html
		IDFromURLRegex string `yaml:"id_from_url_regex"`

		// 或者：从“详情页 HTML”里用选择器拿到能含有 ID 的属性（如 href 或 data-id）
		IDSelector string `yaml:"id_selector"` // 例：a[href^="/chapter/"]
		IDAttr     string `yaml:"id_attr"`     // 默认为 href
		IDRegex    string `yaml:"id_regex"`    // 可选，对上面的属性再跑一次正则提取 (\d+)
	} `yaml:"toc"`
}

type ContentConfig struct {
	ContentSelector string `yaml:"content_selector"`
}

type SourceConfig struct {
	ID             string            `yaml:"id"`
	Name           string            `yaml:"name"`
	BaseURL        string            `yaml:"base_url"`
	Charset        string            `yaml:"charset"`
	Rate           RateConfig        `yaml:"rate_limit"`
	Retries        int               `yaml:"retries"`
	TimeoutSeconds int               `yaml:"timeout_seconds"`
	Proxy          string            `yaml:"proxy"`
	Headers        map[string]string `yaml:"headers"`
	Search         SearchConfig      `yaml:"search"`
	Chapters       ChaptersConfig    `yaml:"chapters"`
	Content        ContentConfig     `yaml:"content"`
}

type Book struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	ID       string `json:"id"` // 用详情页 URL 充当 ID
	Category string `json:"category"`
	Update   string `json:"update"`
}

type Chapter struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Index int    `json:"index"`
	ID    string `json:"id"`
}

func absURL(base, href string) string {
	if href == "" {
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		return u.String()
	}
	b, err := url.Parse(base)
	if err != nil {
		return ""
	}
	return b.ResolveReference(u).String()
}
