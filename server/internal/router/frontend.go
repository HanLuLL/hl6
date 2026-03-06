package router

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func setupFrontendRoutes(r *gin.Engine) {
	distDir := filepath.Join("web", "dist")
	indexPath := filepath.Join(distDir, "index.html")
	if !isFrontendFile(indexPath) {
		return
	}

	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		requestPath := path.Clean("/" + c.Request.URL.Path)
		if requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		if assetPath := resolveFrontendAssetPath(distDir, requestPath); assetPath != "" {
			c.File(assetPath)
			return
		}

		c.File(indexPath)
	})
}

func resolveFrontendAssetPath(distDir string, requestPath string) string {
	if requestPath == "/" {
		return ""
	}

	assetPath := filepath.Join(distDir, filepath.FromSlash(strings.TrimPrefix(requestPath, "/")))
	if !isFrontendFile(assetPath) {
		return ""
	}

	return assetPath
}

func isFrontendFile(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
