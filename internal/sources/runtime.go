package sources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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
	sc := s.cfg.Search
	if sc.ItemSelector == "" {
		return nil, errors.New("search.item_selector empty")
	}

	// 1) 构建 URL / Query
	// 优先：URL 模板
	var doc *goquery.Document
	//var finalURL []byte
	var err error

	if strings.TrimSpace(sc.URLTemplate) != "" {
		// 模板替换 {{query}}
		tpl := strings.ReplaceAll(sc.URLTemplate, "{{query}}", url.QueryEscape(keyword))

		// 如需分页/额外参数，这里补一下
		u, perr := url.Parse(tpl)
		if perr != nil {
			return nil, fmt.Errorf("invalid search.url: %w", perr)
		}
		q := u.Query()
		// 额外参数
		for k, v := range sc.ExtraParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()

		doc, _, err = s.client.DocumentURL(ctx, u.String(), s.cfg.Headers, s.cfg.Charset)
		if err != nil {
			return nil, err
		}
	} else {
		// path + param 方式（你原来的）
		q := url.Values{}
		p := sc.Param
		if p == "" {
			p = "q"
		}
		q.Set(p, keyword)
		// 额外参数
		for k, v := range sc.ExtraParams {
			q.Set(k, v)
		}

		// 用你已有的 DocumentBy：注意把 base + path 交给它
		doc, _, err = s.client.DocumentBy(ctx, s.cfg.BaseURL, sc.Path, http.MethodGet, q, s.cfg.Headers, s.cfg.Charset)
		if err != nil {
			return nil, err
		}
	}

	// 2) 解析（与原逻辑相同，但使用 finalURL 做相对链接基准）
	linkSel := sc.LinkSelector
	if linkSel == "" {
		linkSel = "a"
	}
	attr := sc.LinkAttr
	if attr == "" {
		attr = "href"
	}

	var items []Book
	seen := make(map[string]bool)

	doc.Find(sc.ItemSelector).Each(func(_ int, s2 *goquery.Selection) {
		title := strings.TrimSpace(s2.Find(sc.TitleSelector).First().Text())

		category := ""
		if sc.CategorySelector != "" {
			category = strings.TrimSpace(s2.Find(sc.CategorySelector).First().Text())
		}

		update := ""
		if sc.UpdateSelector != "" {
			update = strings.TrimSpace(s2.Find(sc.UpdateSelector).First().Text())
		}

		author := ""
		if sc.AuthorSelector != "" {
			author = strings.TrimSpace(s2.Find(sc.AuthorSelector).First().Text())
		}

		href, _ := s2.Find(linkSel).First().Attr(attr)
		href = absURL(s.cfg.BaseURL, strings.TrimSpace(href)) // ★ 用最终 URL 做基准

		if title == "" || href == "" {
			return
		}
		if seen[href] {
			return
		}
		seen[href] = true

		items = append(items, Book{
			Title:    title,
			Author:   author,
			ID:       href, // 用 URL 作为唯一 ID
			Category: category,
			Update:   update,
		})
	})
	return items, nil
}

// 在 Chapters 一开始加一个工具：根据配置把 bookURL → tocURL
func (s *ConfigSource) resolveTOCURL(ctx context.Context, bookURL string) (string, error) {
	toc := s.cfg.Chapters.TOC
	if strings.TrimSpace(toc.URLTemplate) == "" {
		return bookURL, nil // 没配置就直接用传入的 URL 当目录页
	}

	// 1) 先用 IDFromURLRegex 从 URL 本身提取
	if strings.TrimSpace(toc.IDFromURLRegex) != "" {
		re, err := regexp.Compile(toc.IDFromURLRegex)
		if err == nil {
			if m := re.FindStringSubmatch(bookURL); len(m) >= 2 {
				id := m[1]
				tocURL := strings.ReplaceAll(toc.URLTemplate, "{{id}}", id)
				return tocURL, nil
			}
		}
	}

	// 2) 否则请求详情页 HTML，再用选择器/正则提取 ID
	doc, _, err := s.client.DocumentURL(ctx, bookURL, s.cfg.Headers, s.cfg.Charset)
	if err != nil {
		return "", err
	}
	sel := strings.TrimSpace(toc.IDSelector)
	if sel == "" {
		return "", fmt.Errorf("toc.url_template set but no id_from_url_regex nor id_selector")
	}
	attr := toc.IDAttr
	if attr == "" {
		attr = "href"
	}
	raw, _ := doc.Find(sel).First().Attr(attr)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("toc id not found by selector")
	}
	id := raw
	if strings.TrimSpace(toc.IDRegex) != "" {
		re, err := regexp.Compile(toc.IDRegex)
		if err != nil {
			return "", err
		}
		m := re.FindStringSubmatch(raw)
		if len(m) < 2 {
			return "", fmt.Errorf("toc id not match by id_regex")
		}
		id = m[1]
	}
	tocURL := strings.ReplaceAll(toc.URLTemplate, "{{id}}", id)
	return tocURL, nil
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

	// 先把 bookURL 解析成实际目录页 URL（若配置了 TOC 模板）
	bookURL, err := s.resolveTOCURL(ctx, bookURL)
	if err != nil {
		return nil, err
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
		//html, _ := p.Html()
		t := strings.TrimSpace(p.Text())
		if t != "" {
			parts = append(parts, t)
		}
	})
	return strings.Join(parts, ""), nil
}
