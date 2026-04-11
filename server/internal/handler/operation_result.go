package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
)

func idempotencyKeyFromHeader(c *gin.Context) (string, bool) {
	key := strings.TrimSpace(c.GetHeader("X-Idempotency-Key"))
	if key == "" {
		response.ErrorWithKey(c, http.StatusBadRequest, "missing idempotency key", "error.invalidRequestBody")
		return "", false
	}
	return key, true
}

func writeOperationResult(c *gin.Context, result service.OperationResult) {
	status := result.HTTPStatus
	if status == 0 {
		status = http.StatusInternalServerError
	}
	if status >= 400 {
		if result.MessageKey != "" {
			if result.Data != nil {
				response.ErrorWithKeyData(c, status, result.Message, result.MessageKey, result.Data)
				return
			}
			response.ErrorWithKey(c, status, result.Message, result.MessageKey)
			return
		}
		if result.Data != nil {
			c.JSON(status, response.Response{Code: -1, Message: result.Message, Data: result.Data})
			return
		}
		response.Error(c, status, result.Message)
		return
	}
	c.JSON(status, response.Response{Code: 0, Message: result.Message, Data: result.Data})
}
