package helpers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"hl6-server/pkg/response"
)

func ParseUintParam(c *gin.Context, name string) (uint, bool) {
	val, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid ID", "error.invalidID")
		return 0, false
	}
	return uint(val), true
}

// RequireUintPathUint 将路径参数解析为 uint；失败时响应 400 并返回 false。
func RequireUintPathUint(c *gin.Context, name string) (uint, bool) {
	return ParseUintParam(c, name)
}

// ParseOptionalRFC3339Query 解析可选 RFC3339 查询参数；空串返回 nil。
func ParseOptionalRFC3339Query(c *gin.Context, name string) (*time.Time, error) {
	s := strings.TrimSpace(c.Query(name))
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, s)
	}
	if err != nil {
		return nil, err
	}
	utc := t.UTC()
	return &utc, nil
}
