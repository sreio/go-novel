package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	fepub "github.com/sreio/go-novel/internal/format/epub"
	fpdf "github.com/sreio/go-novel/internal/format/pdf"
	ftxt "github.com/sreio/go-novel/internal/format/txt"
	"github.com/sreio/go-novel/internal/sources"
	"golang.org/x/sync/errgroup"
)

var (
	sourcesDir  string
	outputDir   string
	concurrency int
)

func main() {
	root := &cobra.Command{Use: "novel"}
	root.PersistentFlags().StringVar(&sourcesDir, "sources", "./configs/sources", "书源配置目录")
	root.PersistentFlags().StringVar(&outputDir, "out", "./outputs", "输出目录")
	root.PersistentFlags().IntVar(&concurrency, "concurrency", 8, "章节并发下载数")

	root.AddCommand(cmdSearch())
	root.AddCommand(cmdDownload())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadAllSources(dir string) ([]*sources.ConfigSource, error) {
	cfgs, err := sources.LoadFromDir(dir)
	if err != nil {
		return nil, err
	}
	var out []*sources.ConfigSource
	for _, c := range cfgs {
		s, err := sources.NewFromConfig(c)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func cmdSearch() *cobra.Command {
	var keyword string
	cmd := &cobra.Command{
		Use: "search",
		RunE: func(cmd *cobra.Command, args []string) error {
			ss, err := loadAllSources(sourcesDir)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			type pair struct {
				src   *sources.ConfigSource
				items []sources.Book
			}
			var res []pair
			for _, s := range ss {
				items, _ := s.Search(ctx, keyword, 1)
				res = append(res, pair{src: s, items: items})
			}
			for _, p := range res {
				fmt.Printf("[%s]%s\n", p.src.ID(), p.src.Name())
				for i, b := range p.items {
					fmt.Printf("  %d. %s — %s (%s)\n", i+1, b.Title, b.Author, b.ID)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&keyword, "keyword", "k", "", "关键词")
	_ = cmd.MarkFlagRequired("keyword")
	return cmd
}

func chooseSourceByURL(all []*sources.ConfigSource, u string) *sources.ConfigSource {
	sort.SliceStable(all, func(i, j int) bool { return len(all[i].Name()) > len(all[j].Name()) })
	for _, s := range all {
		if s == nil || u == "" {
			continue
		}
		if strings.Contains(u, s.Name()) || strings.Contains(u, s.ID()) {
			return s
		}
	}
	if len(all) > 0 {
		return all[0]
	}
	return nil
}

func cmdDownload() *cobra.Command {
	var bookURL, format string
	cmd := &cobra.Command{
		Use: "download",
		RunE: func(cmd *cobra.Command, args []string) error {
			ss, err := loadAllSources(sourcesDir)
			if err != nil {
				return err
			}
			src := chooseSourceByURL(ss, bookURL)
			if src == nil {
				return fmt.Errorf("no source available")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()

			chs, err := src.Chapters(ctx, bookURL, bookURL)
			if err != nil {
				return err
			}
			if len(chs) == 0 {
				return fmt.Errorf("no chapters found")
			}

			type text struct{ Title, Content string }
			out := make([]text, len(chs))

			sem := make(chan struct{}, concurrency)
			g, gctx := errgroup.WithContext(ctx)
			for i := range chs {
				i := i
				g.Go(func() error {
					select {
					case sem <- struct{}{}:
					case <-gctx.Done():
						return gctx.Err()
					}
					defer func() { <-sem }()
					content, err := src.Content(gctx, chs[i])
					if err != nil {
						return err
					}
					out[i] = text{Title: chs[i].Title, Content: content}
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return err
			}

			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				return err
			}
			fname := "book." + strings.ToLower(format)
			if u, e := url.Parse(bookURL); e == nil {
				base := filepath.Base(u.Path)
				if base != "" && base != "/" {
					fname = base + "." + strings.ToLower(format)
				}
			}
			dst := filepath.Join(outputDir, fname)

			switch strings.ToLower(format) {
			case "txt":
				conv := make([]ftxt.Chapter, len(out))
				for i, c := range out {
					conv[i] = ftxt.Chapter{Title: c.Title, Content: c.Content}
				}
				return ftxt.Save(dst, conv)
			case "epub":
				chapters := make([]fepub.Chapter, len(out))
				for i, c := range out {
					chapters[i] = fepub.Chapter{Title: c.Title, Content: c.Content}
				}
				meta := fepub.Meta{Title: "Book", Author: ""}
				return fepub.Save(dst, meta, chapters)
			case "pdf":
				chapters := make([]fpdf.Chapter, len(out))
				for i, c := range out {
					chapters[i] = fpdf.Chapter{Title: c.Title, Content: c.Content}
				}
				meta := fpdf.Meta{Title: "Book", Author: ""}
				return fpdf.Save(dst, meta, chapters)
			default:
				return fmt.Errorf("unknown format: %s", format)
			}
		},
	}
	cmd.Flags().StringVar(&bookURL, "url", "", "书籍详情页 URL")
	cmd.Flags().StringVarP(&format, "format", "f", "txt", "输出格式：txt|epub|pdf")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}
