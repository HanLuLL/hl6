package audit

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// 禁止访问的 IP 段（SSRF 防护）
var forbiddenCIDRs = mustParseCIDRs([]string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"0.0.0.0/8",
	"100.64.0.0/10",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::1/128",
	"fe80::/10",
	"fc00::/7",
})

var (
	ErrPrivateIP        = errors.New("target resolves to private/reserved IP")
	ErrRedirectLoop     = errors.New("too many redirects")
	ErrResponseTooLarge = errors.New("response body exceeds limit")
)

const (
	FetchStatusClean       = "clean"
	FetchStatusUnreachable = "unreachable"
	FetchStatusError       = "error"
)

// FetchResult 抓取结果，供规则引擎匹配。
type FetchResult struct {
	StatusCode   int
	FinalURL     string
	Title        string
	Body         string
	ContentHash  string
	Error        error
	ErrorMessage string
	Status       string // FetchStatusClean / FetchStatusUnreachable / FetchStatusError
}

// SafeFetcher 通过 SSRF 防护的 HTTP 客户端安全抓取网站内容。
type SafeFetcher struct {
	httpClient   *http.Client
	maxBodySize  int64
	maxRedirects int
	resolver     *net.Resolver
	dialer       *net.Dialer
	timeout      time.Duration
}

// SafeFetcherOption 配置 SafeFetcher。
type SafeFetcherOption func(*SafeFetcher)

// WithTimeout 设置请求超时（默认 15s）。
func WithTimeout(d time.Duration) SafeFetcherOption {
	return func(f *SafeFetcher) {
		f.timeout = d
	}
}

// WithResolver 使用指定的 DNS 解析器（默认 1.1.1.1:53）。
func WithResolver(addr string) SafeFetcherOption {
	return func(f *SafeFetcher) {
		f.resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, "udp", addr)
			},
		}
	}
}

// NewSafeFetcher 构造安全的 HTTP 客户端，默认 15s 超时、2MB 限制、5 跳重定向、公共 DNS。
func NewSafeFetcher(opts ...SafeFetcherOption) *SafeFetcher {
	f := &SafeFetcher{
		maxBodySize:  2 * 1024 * 1024,
		maxRedirects: 5,
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, "udp", "1.1.1.1:53")
			},
		},
	}

	for _, opt := range opts {
		opt(f)
	}

	f.dialer = &net.Dialer{
		Timeout:  10 * time.Second,
		Resolver: f.resolver,
	}

	transport := &http.Transport{
		DialContext:         f.safeDialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   true,
		MaxIdleConns:        1,
	}

	f.httpClient = &http.Client{
		Transport:     transport,
		CheckRedirect: f.checkRedirect,
		Timeout:       f.timeout,
	}

	if f.httpClient.Timeout == 0 {
		f.httpClient.Timeout = 15 * time.Second
	}

	return f
}

// safeDialContext 在建立连接前校验目标 IP 是否为私有/保留地址。
func (f *SafeFetcher) safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = "80"
	}
	if port != "80" && port != "443" {
		return nil, fmt.Errorf("port %s not allowed (only 80/443)", port)
	}

	if err := f.validateHost(ctx, host); err != nil {
		return nil, err
	}

	return f.dialer.DialContext(ctx, network, address)
}

// validateHost 解析主机名并检查所有 IP 是否在禁止范围内。
func (f *SafeFetcher) validateHost(ctx context.Context, host string) error {
	ips, err := f.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("dns resolve %s: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("no IPs resolved for %s", host)
	}

	for _, ipAddr := range ips {
		ip := ipAddr.IP
		for _, cidr := range forbiddenCIDRs {
			if cidr.Contains(ip) {
				return fmt.Errorf("%w: %s", ErrPrivateIP, ip.String())
			}
		}
	}
	return nil
}

// checkRedirect 拦截重定向：限跳数、校验目标 IP。
func (f *SafeFetcher) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= f.maxRedirects {
		return ErrRedirectLoop
	}

	host := req.URL.Hostname()
	port := req.URL.Port()
	if port == "" {
		if req.URL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	if port != "80" && port != "443" {
		return fmt.Errorf("redirect port %s not allowed", port)
	}

	return f.validateHost(req.Context(), host)
}

var titleRegex = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// Fetch 抓取指定 FQDN 的网站内容（优先 https://，不可达时回退 http://）。
func (f *SafeFetcher) Fetch(ctx context.Context, fqdn string) FetchResult {
	httpsResult := f.fetchURL(ctx, "https://"+fqdn+"/")
	if httpsResult.Status == FetchStatusClean {
		return httpsResult
	}
	if httpsResult.Status == FetchStatusUnreachable && !errors.Is(httpsResult.Error, ErrPrivateIP) {
		httpResult := f.fetchURL(ctx, "http://"+fqdn+"/")
		if httpResult.Status == FetchStatusClean {
			return httpResult
		}
	}
	return httpsResult
}

func (f *SafeFetcher) fetchURL(ctx context.Context, rawURL string) FetchResult {
	result := FetchResult{}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		result.Error = err
		result.ErrorMessage = err.Error()
		result.Status = FetchStatusError
		return result
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		result.Error = err
		result.ErrorMessage = err.Error()
		result.Status = FetchStatusError
		return result
	}
	req.Header.Set("User-Agent", "HL6-ContentAudit/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		result.Error = err
		result.ErrorMessage = err.Error()
		if errors.Is(err, ErrPrivateIP) {
			result.Status = FetchStatusError
		} else {
			result.Status = FetchStatusUnreachable
		}
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	if resp.Request != nil && resp.Request.URL != nil {
		result.FinalURL = resp.Request.URL.String()
	}

	limitedReader := io.LimitReader(resp.Body, f.maxBodySize)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		result.Error = err
		result.ErrorMessage = err.Error()
		result.Status = FetchStatusError
		return result
	}
	if len(bodyBytes) >= int(f.maxBodySize) {
		result.Error = ErrResponseTooLarge
		result.ErrorMessage = ErrResponseTooLarge.Error()
	}

	bodyStr := string(bodyBytes)
	result.Body = bodyStr

	if m := titleRegex.FindStringSubmatch(bodyStr); len(m) >= 2 {
		result.Title = strings.TrimSpace(m[1])
	}

	hash := sha256.Sum256(bodyBytes)
	result.ContentHash = fmt.Sprintf("%x", hash)

	result.Status = FetchStatusClean
	return result
}

func mustParseCIDRs(raw []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(raw))
	for _, r := range raw {
		_, cidr, err := net.ParseCIDR(r)
		if err != nil {
			panic(fmt.Sprintf("invalid CIDR %q: %v", r, err))
		}
		out = append(out, cidr)
	}
	return out
}
