package audit

import (
	"bytes"
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
	"sync"
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
	FetchStatusClean      = "clean"
	FetchStatusUnreachable = "unreachable"
	FetchStatusError       = "error"
)

const defaultMaxBodySize = 10 * 1024 * 1024 // 10MB，覆盖绝大多数正常网页

// FetchResult 抓取结果，供规则引擎匹配。
type FetchResult struct {
	StatusCode   int
	FinalURL     string
	Title        string
	Body         string
	ContentHash  string
	Truncated    bool  // body 因超过 maxBodySize 上限被截断
	BodyBytes    int64 // 实际读入 Body 的字节数（截断后等于 maxBodySize）
	Error        error
	ErrorMessage string
	Status       string // FetchStatusClean / FetchStatusUnreachable / FetchStatusError
}

// ChannelResult 单协议抓取结果。
type ChannelResult struct {
	Scheme     string // "https" | "http"
	RequestURL string
	FetchResult
}

// DualFetchResult 双通道并行抓取结果。
type DualFetchResult struct {
	HTTPS ChannelResult
	HTTP  ChannelResult
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
// 传入 <=0 时无效，保留默认值；如需禁用超时请用非常大的正值（如 24h）。
func WithTimeout(d time.Duration) SafeFetcherOption {
	return func(f *SafeFetcher) {
		if d > 0 {
			f.timeout = d
		}
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

// WithMaxBodySize 设置响应体最大读取上限（默认 10MB）。
// 传入 <=0 时无效，保留默认值。超过上限的 body 被截断，Truncated 标记为 true。
func WithMaxBodySize(size int64) SafeFetcherOption {
	return func(f *SafeFetcher) {
		if size > 0 {
			f.maxBodySize = size
		}
	}
}

// NewSafeFetcher 构造安全的 HTTP 客户端，默认 15s 超时、10MB 限制、5 跳重定向、公共 DNS。
func NewSafeFetcher(opts ...SafeFetcherOption) *SafeFetcher {
	f := &SafeFetcher{
		maxBodySize:  defaultMaxBodySize,
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
		Timeout: 10 * time.Second,
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

// safeDialContext 在建立连接前校验目标 IP 是否为私有/保留地址，
// 并使用 validateHost 返回的已验证 IP 直接连接，避免二次 DNS 解析的 TOCTOU 漏洞。
func (f *SafeFetcher) safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = "80"
	}
	if port != "80" && port != "443" {
		return nil, fmt.Errorf("port %s not allowed (only 80/443)", port)
	}

	ips, err := f.validateHost(ctx, host)
	if err != nil {
		return nil, err
	}

	// 直接用已验证的 IP 连接，不经过 dialer 的二次 DNS 解析
	var firstErr error
	for _, ip := range ips {
		addr := net.JoinHostPort(ip.String(), port)
		conn, dialErr := f.dialer.DialContext(ctx, network, addr)
		if dialErr == nil {
			return conn, nil
		}
		if firstErr == nil {
			firstErr = dialErr
		}
	}
	return nil, firstErr
}

// validateHost 解析主机名、检查所有 IP 是否在禁止范围内，返回通过检查的 IP 列表。
// 调用方应使用返回的 IP 列表发起连接，避免二次 DNS 解析造成的 TOCTOU 漏洞。
func (f *SafeFetcher) validateHost(ctx context.Context, host string) ([]net.IP, error) {
	ips, err := f.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("dns resolve %s: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no IPs resolved for %s", host)
	}

	validated := make([]net.IP, 0, len(ips))
	for _, ipAddr := range ips {
		ip := ipAddr.IP
		for _, cidr := range forbiddenCIDRs {
			if cidr.Contains(ip) {
				return nil, fmt.Errorf("%w: %s", ErrPrivateIP, ip.String())
			}
		}
		validated = append(validated, ip)
	}
	return validated, nil
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

	_, err := f.validateHost(req.Context(), host)
	return err
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

// FetchDual 并行抓取 HTTP 与 HTTPS，两通道独立检测。
func (f *SafeFetcher) FetchDual(ctx context.Context, fqdn string) DualFetchResult {
	var result DualFetchResult
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		fr := f.fetchURL(ctx, "https://"+fqdn+"/")
		result.HTTPS = ChannelResult{
			Scheme:      "https",
			RequestURL:  "https://" + fqdn + "/",
			FetchResult: fr,
		}
	}()
	go func() {
		defer wg.Done()
		fr := f.fetchURL(ctx, "http://"+fqdn+"/")
		result.HTTP = ChannelResult{
			Scheme:      "http",
			RequestURL:  "http://" + fqdn + "/",
			FetchResult: fr,
		}
	}()
	wg.Wait()
	return result
}

// CombinedContentHash 双通道组合哈希，用于增量跳过判断。
func CombinedContentHash(dual DualFetchResult) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%s|%s",
		dual.HTTPS.ContentHash,
		dual.HTTP.ContentHash,
		dual.HTTPS.Status,
		dual.HTTP.Status,
	)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// PrimaryChannel 返回优先用于兼容字段的通道（HTTPS 优先，完全失败时用 HTTP）。
func (d DualFetchResult) PrimaryChannel() ChannelResult {
	httpsOK := d.HTTPS.Status == FetchStatusClean && !IsHTTPStatusInaccessible(d.HTTPS.StatusCode)
	if httpsOK || (d.HTTPS.FinalURL != "" && !IsHTTPStatusInaccessible(d.HTTPS.StatusCode)) {
		return d.HTTPS
	}
	return d.HTTP
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

	// 分块流式读取 body：增量计算 ContentHash，按需提取 Title，
	// 达到 maxBodySize 上限后截断并标记 Truncated。
	hash := sha256.New()
	var buf bytes.Buffer
	var titleFound bool
	chunk := make([]byte, 64*1024)
	scanStart := 0 // buf 中已搜索过 title 的末尾偏移，避免对大 body 重复全量 regex

	for {
		n, readErr := resp.Body.Read(chunk)
		if n > 0 {
			hash.Write(chunk[:n])
			buf.Write(chunk[:n])
			// title 提取：仅在前 ~128KB 内未找到时搜索；找到后不再搜
			if !titleFound && buf.Len() <= 128*1024 {
				if m := titleRegex.FindStringSubmatch(buf.String()[scanStart:]); len(m) >= 2 {
					result.Title = strings.TrimSpace(m[1])
					titleFound = true
				}
				scanStart = buf.Len() - 100 // 保留重叠防止 <title> 跨 chunk 边界
				if scanStart < 0 {
					scanStart = 0
				}
			}
		}
		if int64(buf.Len()) > f.maxBodySize {
			result.Truncated = true
			buf.Truncate(int(f.maxBodySize))
			_, _ = io.Copy(io.Discard, resp.Body) // 排空剩余 body，释放连接
			break
		}
		if readErr != nil {
			if readErr != io.EOF {
				result.Error = readErr
				result.ErrorMessage = readErr.Error()
				result.Status = FetchStatusError
			}
			break
		}
	}

	result.BodyBytes = int64(buf.Len())
	result.Body = buf.String()
	result.ContentHash = fmt.Sprintf("%x", hash.Sum(nil))

	if result.Truncated {
		result.Error = ErrResponseTooLarge
		result.ErrorMessage = ErrResponseTooLarge.Error()
		result.Status = FetchStatusError
		return result
	}

	// body 完整读取，再用 titleRegex 兜底搜索一次（超过 128KB 的场景）
	if !titleFound {
		if m := titleRegex.FindStringSubmatch(result.Body); len(m) >= 2 {
			result.Title = strings.TrimSpace(m[1])
		}
	}

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
