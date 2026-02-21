package helpers

import (
	"net/http"
	"strconv"

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
