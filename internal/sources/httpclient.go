package sources

import (
    "bytes"
    "context"
    "crypto/tls"
    "errors"
    "fmt"
    "io"
    "math"
    "math/rand"
    "net"
    "net/http"
    "net/url"
    "strings"
    "time"

    "github.com/PuerkitoBio/goquery"
    "golang.org/x/text/encoding/simplifiedchinese"
    "golang.org/x/text/transform"
    "golang.org/x/time/rate"
)

// HTTPClient：封装限速、超时、代理、重试与 charset 解码。
type HTTPClient struct {
    hc       *http.Client
    limiter  *rate.Limiter
    retries  int
    baseURL  string
    defHeads map[string]string
    charset  string // 默认字符集（可被页面 meta 覆盖）
}

func NewHTTPClient(cfg SourceConfig) (*HTTPClient, error) {
    tr := &http.Transport{
        Proxy:               http.ProxyFromEnvironment,
        DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
        TLSHandshakeTimeout: 10 * time.Second,
        MaxIdleConns:        100,
        IdleConnTimeout:     90 * time.Second,
        TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
    }
    if cfg.Proxy != "" {
        if purl, err := url.Parse(cfg.Proxy); err == nil {
            tr.Proxy = http.ProxyURL(purl)
        }
    }
    timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
    if timeout <= 0 {
        timeout = 15 * time.Second
    }
    hc := &http.Client{Transport: tr, Timeout: timeout}

    var lim *rate.Limiter
    if cfg.Rate.RPS > 0 {
        burst := cfg.Rate.Burst
        if burst <= 0 {
            burst = int(math.Ceil(cfg.Rate.RPS))
        }
        lim = rate.NewLimiter(rate.Limit(cfg.Rate.RPS), burst)
    }

    retries := cfg.Retries
    if retries <= 0 { retries = 3 }

    return &HTTPClient{
        hc:       hc,
        limiter:  lim,
        retries:  retries,
        baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
        defHeads: map[string]string{"User-Agent": "go-novel/1.0"},
        charset:  strings.ToLower(cfg.Charset),
    }, nil
}

func (c *HTTPClient) wait(ctx context.Context) error {
    if c.limiter == nil { return nil }
    return c.limiter.Wait(ctx)
}

func (c *HTTPClient) do(ctx context.Context, req *http.Request) (*http.Response, error) {
    for k, v := range c.defHeads { if req.Header.Get(k) == "" { req.Header.Set(k, v) } }
    var lastErr error
    for attempt := 0; attempt <= c.retries; attempt++ {
        if err := c.wait(ctx); err != nil { return nil, err }
        resp, err := c.hc.Do(req.WithContext(ctx))
        if err == nil && (resp.StatusCode < 500 && resp.StatusCode != 429) {
            return resp, nil
        }
        if err != nil {
            if ne, ok := err.(net.Error); ok && (ne.Timeout() || ne.Temporary()) {
                lastErr = err
            } else { return nil, err }
        } else {
            lastErr = fmt.Errorf("http %d", resp.StatusCode)
            io.Copy(io.Discard, resp.Body)
            resp.Body.Close()
        }
        d := backoff(attempt)
        select { case <-time.After(d): case <-ctx.Done(): return nil, ctx.Err() }
    }
    if lastErr == nil { lastErr = errors.New("request failed") }
    return nil, lastErr
}

func backoff(attempt int) time.Duration {
    base := 300 * time.Millisecond
    max := 3 * time.Second
    d := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
    if d > max { d = max }
    jitter := 0.5 + rand.Float64() // 0.5x ~ 1.5x
    return time.Duration(float64(d) * jitter)
}

func (c *HTTPClient) buildURL(base, p string, q url.Values) string {
    if base == "" { base = c.baseURL }
    base = strings.TrimRight(base, "/")
    p = "/" + strings.TrimLeft(p, "/")
    u := base + p
    if len(q) > 0 { if strings.Contains(u, "?") { u += "&" + q.Encode() } else { u += "?" + q.Encode() } }
    return u
}

func (c *HTTPClient) request(ctx context.Context, method, fullURL string, headers map[string]string, body io.Reader) ([]byte, *http.Response, error) {
    req, err := http.NewRequest(method, fullURL, body)
    if err != nil { return nil, nil, err }
    for k, v := range headers { req.Header.Set(k, v) }
    resp, err := c.do(ctx, req)
    if err != nil { return nil, nil, err }
    defer resp.Body.Close()
    b, err := io.ReadAll(resp.Body)
    if err != nil { return nil, nil, err }
    return b, resp, nil
}

func sniffCharset(b []byte) string {
    s := strings.ToLower(string(b))
    if strings.Contains(s, "charset=gb18030") { return "gb18030" }
    if strings.Contains(s, "charset=gbk") { return "gbk" }
    if strings.Contains(s, "charset=gb2312") { return "gb2312" }
    if strings.Contains(s, "charset=gb-2312") { return "gb2312" }
    return ""
}

func decodeHTML(raw []byte, hint string) ([]byte, string, error) {
    if hint == "" && len(raw) > 0 {
        h := raw
        if len(h) > 2048 { h = h[:2048] }
        hint = sniffCharset(h)
    }
    switch strings.ToLower(hint) {
    case "gbk", "cp936":
        r := transform.NewReader(bytes.NewReader(raw), simplifiedchinese.GBK.NewDecoder())
        b, e := io.ReadAll(r); return b, "gbk", e
    case "gb2312":
        r := transform.NewReader(bytes.NewReader(raw), simplifiedchinese.HZGB2312.NewDecoder())
        b, e := io.ReadAll(r); return b, "gb2312", e
    case "gb18030":
        r := transform.NewReader(bytes.NewReader(raw), simplifiedchinese.GB18030.NewDecoder())
        b, e := io.ReadAll(r); return b, "gb18030", e
    default:
        return raw, "utf-8", nil
    }
}

// DocumentBy: 基于 base+path+query 构造 URL 并返回 goquery 文档
func (c *HTTPClient) DocumentBy(ctx context.Context, base, path, method string, q url.Values, headers map[string]string, charset string) (*goquery.Document, []byte, error) {
    if method == "" { method = http.MethodGet }
    u := c.buildURL(base, path, q)
    return c.DocumentURL(ctx, u, headers, charset)
}

// DocumentURL: 直接 URL 抓取 goquery 文档
func (c *HTTPClient) DocumentURL(ctx context.Context, u string, headers map[string]string, charset string) (*goquery.Document, []byte, error) {
    raw, _, err := c.request(ctx, http.MethodGet, u, headers, nil)
    if err != nil { return nil, nil, err }
    dec, _, err := decodeHTML(raw, charset)
    if err != nil { return nil, nil, err }
    doc, err := goquery.NewDocumentFromReader(bytes.NewReader(dec))
    if err != nil { return nil, nil, err }
    return doc, dec, nil
}
