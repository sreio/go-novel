package sources

import (
	"context"
	"errors"
	"net/http"
	"net/url"
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
			items = append(items, Book{Title: title, Author: author, ID: href})
		}
	})
	return items, nil
}

func (s *ConfigSource) Chapters(ctx context.Context, bookURL string, id string) ([]Chapter, error) {
	doc, _, err := s.client.DocumentURL(ctx, bookURL, s.cfg.Headers, s.cfg.Charset)
	if err != nil {
		return nil, err
	}
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
		href = absURL(bookURL, strings.TrimSpace(href))
		if href != "" {
			list = append(list, Chapter{Title: title, URL: href, Index: i, ID: href})
		}
	})
	return list, nil
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
		t := strings.TrimSpace(p.Text())
		if t != "" {
			parts = append(parts, t)
		}
	})
	return strings.Join(parts, ""), nil
}
