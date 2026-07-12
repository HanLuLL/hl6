package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type SEOHandler struct {
	repo *repository.Repository
}

func NewSEOHandler(repo *repository.Repository) *SEOHandler {
	return &SEOHandler{repo: repo}
}

// seoMetaResponse is the public SEO metadata returned to frontend
type seoMetaResponse struct {
	SiteName        string `json:"site_name"`
	SiteDescription string `json:"site_description"`
	SiteKeywords    string `json:"site_keywords"`
}

// GetSEOMeta returns public SEO configuration
func (h *SEOHandler) GetSEOMeta(c *gin.Context) {
	keys := []string{
		"brand_name",
		"seo_description",
		"seo_keywords",
	}
	configs, _ := h.repo.GetSystemConfigsByKeys(keys)

	siteName := configs["brand_name"]
	if siteName == "" {
		siteName = "HL6"
	}

	response.OK(c, seoMetaResponse{
		SiteName:        siteName,
		SiteDescription: configs["seo_description"],
		SiteKeywords:    configs["seo_keywords"],
	})
}

// RobotsTXT serves robots.txt with sitemap reference
func (h *SEOHandler) RobotsTXT(c *gin.Context) {
	urlState, err := h.resolveBackendURL(c)
	if err != nil || urlState == "" {
		urlState = ""
	}

	// Check if SEO is disabled
	disabled, _ := h.repo.GetSystemConfig("seo_indexing_disabled")

	var content string
	if disabled == "true" {
		content = "User-agent: *\nDisallow: /\n"
	} else {
		sitemapURL := fmt.Sprintf("%s/sitemap.xml", urlState)
		content = fmt.Sprintf("User-agent: *\nAllow: /\nDisallow: /api/\nDisallow: /dashboard\nDisallow: /profile\nDisallow: /credits\nDisallow: /subdomains\nDisallow: /admin\n\nSitemap: %s\n", sitemapURL)
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, content)
}

// SitemapXML serves sitemap.xml
func (h *SEOHandler) SitemapXML(c *gin.Context) {
	backendURL, err := h.resolveBackendURL(c)
	if err != nil || backendURL == "" {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Check if indexing is disabled
	disabled, _ := h.repo.GetSystemConfig("seo_indexing_disabled")
	if disabled == "true" {
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`)
		return
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z07:00")

	// Get public domains for sitemap
	domains, _ := h.repo.GetActiveDomains()

	var urls strings.Builder
	urls.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	urls.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)

	// Homepage
	urls.WriteString(fmt.Sprintf(`<url><loc>%s/</loc><lastmod>%s</lastmod><changefreq>daily</changefreq><priority>1.0</priority></url>`, backendURL, now))

	// Domain pages
	for _, domain := range domains {
		urls.WriteString(fmt.Sprintf(`<url><loc>%s/domains</loc><lastmod>%s</lastmod><changefreq>weekly</changefreq><priority>0.8</priority></url>`, backendURL, now))
		_ = domain // domain data could be used for more specific URLs in future
	}

	// Static pages
	staticPages := []struct {
		Path     string
		Priority string
		Freq     string
	}{
		{"/domains", "0.8", "weekly"},
		{"/credits", "0.6", "monthly"},
	}
	for _, p := range staticPages {
		urls.WriteString(fmt.Sprintf(`<url><loc>%s%s</loc><lastmod>%s</lastmod><changefreq>%s</changefreq><priority>%s</priority></url>`, backendURL, p.Path, now, p.Freq, p.Priority))
	}

	urls.WriteString(`</urlset>`)

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.String(http.StatusOK, urls.String())
}

func (h *SEOHandler) resolveBackendURL(c *gin.Context) (string, error) {
	// Try to get backend URL from system config
	backendURL, err := h.repo.GetSystemConfig("backend_url")
	if err == nil && backendURL != "" {
		return strings.TrimRight(backendURL, "/"), nil
	}

	// Fallback to request scheme + host
	scheme := "https"
	if c.Request.TLS == nil {
		if fwd := c.GetHeader("X-Forwarded-Proto"); fwd != "" {
			scheme = fwd
		} else {
			scheme = "http"
		}
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host), nil
}
