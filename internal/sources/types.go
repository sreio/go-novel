package sources

import (
	"net/url"
)

type RateConfig struct {
	RPS   float64 `yaml:"rps"`
	Burst int     `yaml:"burst"`
}

type SearchConfig struct {
	Path             string `yaml:"path"`          // 如 /search
	Param            string `yaml:"param"`         // 如 q
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

func absURL(base string, href string) string {
	if href == "" {
		return base
	}
	bu, err := url.Parse(base)
	if err != nil {
		return href
	}
	hu, err := url.Parse(href)
	if err != nil {
		return href
	}
	return bu.ResolveReference(hu).String()
}
