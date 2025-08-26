package sources

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ConfigSource struct {
	cfg    SourceConfig
	client *HTTPClient
}

func NewFromConfig(cfg SourceConfig) (*ConfigSource, error) {
	cli, err := NewHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ConfigSource{cfg: cfg, client: cli}, nil
}

func (s *ConfigSource) ID() string {
	if s.cfg.ID != "" {
		return s.cfg.ID
	}
	return s.cfg.BaseURL
}
func (s *ConfigSource) Name() string {
	if s.cfg.Name != "" {
		return s.cfg.Name
	}
	return s.cfg.BaseURL
}

func (s *ConfigSource) Search(ctx context.Context, keyword string, page int) ([]Book, error) {
	q := url.Values{}
	p := s.cfg.Search.Param
	if p == "" {
		p = "q"
	}
	q.Set(p, keyword)
	doc, _, err := s.client.DocumentBy(ctx, s.cfg.BaseURL, s.cfg.Search.Path, http.MethodGet, q, s.cfg.Headers, s.cfg.Charset)
	if err != nil {
		return nil, err
	}
	sel := s.cfg.Search.ItemSelector
	if sel == "" {
		return nil, errors.New("search.item_selector empty")
	}
	var items []Book
	doc.Find(sel).Each(func(i int, s2 *goquery.Selection) {
		title := strings.TrimSpace(s2.Find(s.cfg.Search.TitleSelector).First().Text())
		category := strings.TrimSpace(s2.Find(s.cfg.Search.CategorySelector).First().Text())
		update := strings.TrimSpace(s2.Find(s.cfg.Search.UpdateSelector).First().Text())
		author := ""
		if s.cfg.Search.AuthorSelector != "" {
			author = strings.TrimSpace(s2.Find(s.cfg.Search.AuthorSelector).First().Text())
		}
		linkSel := s.cfg.Search.LinkSelector
		if linkSel == "" {
			linkSel = "a"
		}
		attr := s.cfg.Search.LinkAttr
		if attr == "" {
			attr = "href"
		}
		href, _ := s2.Find(linkSel).First().Attr(attr)
		href = absURL(s.cfg.BaseURL, strings.TrimSpace(href))
		if title != "" && href != "" {
			items = append(items, Book{Title: title, Author: author, ID: href, Category: category, Update: update})
		}
	})
	return items, nil
}

func (s *ConfigSource) Chapters(ctx context.Context, bookURL string, id string) ([]Chapter, error) {
	// 公共工具：解析一页的章节列表
	parseChaptersFromDoc := func(doc *goquery.Document, base string) ([]Chapter, error) {
		var list []Chapter
		ls := s.cfg.Chapters.ListSelector
		if ls == "" {
			return nil, errors.New("chapters.list_selector empty")
		}
		tsel := s.cfg.Chapters.TitleSelector
		usel := s.cfg.Chapters.URLSelector
		if usel == "" {
			usel = "a"
		}
		uattr := s.cfg.Chapters.URLAttr
		if uattr == "" {
			uattr = "href"
		}

		doc.Find(ls).Each(func(i int, li *goquery.Selection) {
			title := strings.TrimSpace(li.Find(tsel).First().Text())
			if title == "" {
				title = strings.TrimSpace(li.Text())
			}
			href, _ := li.Find(usel).First().Attr(uattr)
			href = absURL(base, strings.TrimSpace(href))
			if href != "" {
				list = append(list, Chapter{Title: title, URL: href, Index: i, ID: href})
			}
		})
		return list, nil
	}

	// 先抓第一页
	doc, _, err := s.client.DocumentURL(ctx, bookURL, s.cfg.Headers, s.cfg.Charset)
	if err != nil {
		return nil, err
	}

	all := make([]Chapter, 0, 256)
	seen := make(map[string]bool)

	add := func(items []Chapter) {
		for _, it := range items {
			if it.URL == "" || seen[it.URL] {
				continue
			}
			it.Index = len(all)
			all = append(all, it)
			seen[it.URL] = true
		}
	}

	// 解析第一页
	pageItems, err := parseChaptersFromDoc(doc, bookURL)
	if err != nil {
		return nil, err
	}
	add(pageItems)

	// 分页配置
	p := s.cfg.Chapters.Pagination

	// ============ 模式一：next 链接 ============
	if p.NextSelector != "" {
		nextAttr := p.NextAttr
		if nextAttr == "" {
			nextAttr = "href"
		}

		// 安全上限
		maxPages := p.MaxPages
		if maxPages <= 0 {
			maxPages = 20
		}

		baseURL := bookURL
		prevCount := len(all)

		for page := 2; page <= maxPages; page++ {
			// 找下一页链接
			nextSel := doc.Find(p.NextSelector).First()
			nhref, ok := nextSel.Attr(nextAttr)
			if !ok || strings.TrimSpace(nhref) == "" {
				break
			}

			nextURL := absURL(baseURL, strings.TrimSpace(nhref))

			// 抓取下一页
			d, _, err := s.client.DocumentURL(ctx, nextURL, s.cfg.Headers, s.cfg.Charset)
			if err != nil {
				break
			}
			doc = d // 供下一轮查找“下一页”用

			items, _ := parseChaptersFromDoc(doc, nextURL)
			before := len(all)
			add(items)
			after := len(all)

			if after == before { // 无新章节，认为结束
				break
			}
			if p.StopOnSame && after == prevCount { // 显式开启相同终止
				break
			}
			prevCount = after
		}

		return all, nil
	}

	// ============ 模式二：page=? 参数 ============
	if p.PageParam != "" {
		maxPages := p.MaxPages
		if maxPages <= 0 {
			maxPages = 20
		}
		start := p.StartPage
		if start <= 0 {
			start = 1
		}

		prevCount := len(all)
		// 从第二页开始翻，第一页已抓
		for page := start + 1; page <= start+maxPages-1; page++ {
			// 在 bookURL 上替换/设置页码参数
			u, err := url.Parse(bookURL)
			if err != nil {
				break
			}
			q := u.Query()
			q.Set(p.PageParam, strconv.Itoa(page))
			u.RawQuery = q.Encode()
			nextURL := u.String()

			d, _, err := s.client.DocumentURL(ctx, nextURL, s.cfg.Headers, s.cfg.Charset)
			if err != nil {
				break
			}

			items, _ := parseChaptersFromDoc(d, nextURL)
			before := len(all)
			add(items)
			after := len(all)

			if after == before { // 这一页没有新增
				break
			}
			if p.StopOnSame && after == prevCount {
				break
			}
			prevCount = after
		}
		return all, nil
	}

	// 没有配置分页则返回第一页结果
	return all, nil
}

func (s *ConfigSource) Content(ctx context.Context, ch Chapter) (string, error) {
	doc, _, err := s.client.DocumentURL(ctx, ch.URL, s.cfg.Headers, s.cfg.Charset)
	if err != nil {
		return "", err
	}
	sel := s.cfg.Content.ContentSelector
	if sel == "" {
		return "", errors.New("content.content_selector empty")
	}
	var parts []string
	doc.Find(sel).Each(func(_ int, p *goquery.Selection) {
		html, _ := p.Html()
		t := strings.TrimSpace(html)
		if t != "" {
			parts = append(parts, t)
		}
	})
	return strings.Join(parts, ""), nil
}
