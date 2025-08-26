package sources

import (
    "net/url"
)

type RateConfig struct {
    RPS   float64 `yaml:"rps"`
    Burst int     `yaml:"burst"`
}

type SearchConfig struct {
    Path          string `yaml:"path"`          // 如 /search
    Param         string `yaml:"param"`         // 如 q
    ItemSelector  string `yaml:"item_selector"` // 列表项选择器
    TitleSelector string `yaml:"title_selector"`
    AuthorSelector string `yaml:"author_selector"`
    LinkSelector  string `yaml:"link_selector"` // a[href]
    LinkAttr      string `yaml:"link_attr"`     // 默认 href
}

type ChaptersConfig struct {
    ListSelector  string `yaml:"list_selector"`
    TitleSelector string `yaml:"title_selector"`
    URLSelector   string `yaml:"url_selector"`
    URLAttr       string `yaml:"url_attr"` // 默认 href
}

type ContentConfig struct {
    ContentSelector string `yaml:"content_selector"`
}

type SourceConfig struct {
    ID             string         `yaml:"id"`
    Name           string         `yaml:"name"`
    BaseURL        string         `yaml:"base_url"`
    Charset        string         `yaml:"charset"`
    Rate           RateConfig     `yaml:"rate_limit"`
    Retries        int            `yaml:"retries"`
    TimeoutSeconds int            `yaml:"timeout_seconds"`
    Proxy          string         `yaml:"proxy"`
    Headers        map[string]string `yaml:"headers"`
    Search         SearchConfig   `yaml:"search"`
    Chapters       ChaptersConfig `yaml:"chapters"`
    Content        ContentConfig  `yaml:"content"`
}

type Book struct {
    Title  string `json:"title"`
    Author string `json:"author"`
    ID     string `json:"id"` // 用详情页 URL 充当 ID
}

type Chapter struct {
    Title string `json:"title"`
    URL   string `json:"url"`
    Index int    `json:"index"`
    ID    string `json:"id"`
}

func absURL(base string, href string) string {
    if href == "" { return base }
    bu, err := url.Parse(base)
    if err != nil { return href }
    hu, err := url.Parse(href)
    if err != nil { return href }
    return bu.ResolveReference(hu).String()
}
