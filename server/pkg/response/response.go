package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	MessageKey string      `json:"message_key,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

type PaginatedResponse struct {
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	MessageKey string      `json:"message_key,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
}

type OffsetPaginatedResponse struct {
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	MessageKey string      `json:"message_key,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	Total      int64       `json:"total"`
	Offset     int         `json:"offset"`
	Limit      int         `json:"limit"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "ok", Data: data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{Code: 0, Message: "created", Data: data})
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, Response{Code: -1, Message: message})
}

func ErrorWithKey(c *gin.Context, status int, message, messageKey string) {
	c.JSON(status, Response{Code: -1, Message: message, MessageKey: messageKey})
}

func ErrorWithKeyData(c *gin.Context, status int, message, messageKey string, data interface{}) {
	c.JSON(status, Response{Code: -1, Message: message, MessageKey: messageKey, Data: data})
}

func Paginated(c *gin.Context, data interface{}, total int64, page, perPage int) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Code:    0,
		Message: "ok",
		Data:    data,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

func OffsetPaginated(c *gin.Context, data interface{}, total int64, offset, limit int) {
	c.JSON(http.StatusOK, OffsetPaginatedResponse{
		Code:    0,
		Message: "ok",
		Data:    data,
		Total:   total,
		Offset:  offset,
		Limit:   limit,
	})
}
