package helpers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func ParsePageParams(c *gin.Context, defaultPerPage, maxPerPage int) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", strconv.Itoa(defaultPerPage)))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > maxPerPage {
		perPage = defaultPerPage
	}
	return page, perPage
}

func ParseOffsetLimit(c *gin.Context, defaultLimit, maxLimit int) (int, int) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	if offset < 0 {
		offset = 0
	}
	if limit < 1 || limit > maxLimit {
		limit = defaultLimit
	}
	return offset, limit
}
