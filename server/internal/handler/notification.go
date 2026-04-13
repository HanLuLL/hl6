package handler

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/helpers"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

var htmlTagRegexp = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	return htmlTagRegexp.ReplaceAllString(s, "")
}

type NotificationHandler struct {
	repo   *repository.Repository
	broker *SSEBroker
}

func NewNotificationHandler(repo *repository.Repository, broker *SSEBroker) *NotificationHandler {
	return &NotificationHandler{repo: repo, broker: broker}
}

func (h *NotificationHandler) List(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}

	offset, limit := helpers.ParseOffsetLimit(c, 20, 50)

	groupID := userGroupID(user)

	notifications, total, err := h.repo.ListNotificationsForUser(
		user.ID, groupID, user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		offset, limit,
	)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list notifications", "error.failedToListNotifications")
		return
	}

	response.OffsetPaginated(c, notifications, total, offset, limit)
}

func (h *NotificationHandler) Get(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}

	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	groupID := userGroupID(user)

	notification, err := h.repo.FindNotificationForUser(
		id, user.ID, groupID,
		user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "notification not found", "error.notificationNotFound")
		return
	}

	response.OK(c, notification)
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}

	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Verify visibility before marking read
	groupID := userGroupID(user)
	if _, err := h.repo.FindNotificationForUser(id, user.ID, groupID, user.CreatedAt.Format("2006-01-02T15:04:05Z07:00")); err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "notification not found", "error.notificationNotFound")
		return
	}

	if err := h.repo.MarkNotificationRead(id, user.ID); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to mark read", "error.failedToMarkRead")
		return
	}

	response.OK(c, gin.H{"message": "marked as read"})
}

func (h *NotificationHandler) UnreadStatus(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}

	groupID := userGroupID(user)

	hasUnread, err := h.repo.HasUnreadNotifications(
		user.ID, groupID,
		user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to check unread", "error.failedToCheckUnread")
		return
	}

	response.OK(c, gin.H{"has_unread": hasUnread})
}

func (h *NotificationHandler) SSE(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		c.Abort()
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := h.broker.Subscribe(user.ID)
	defer h.broker.Unsubscribe(user.ID, ch)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent(event.Event, event.Data)
			return true
		case <-ticker.C:
			c.SSEvent("heartbeat", "")
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

func (h *NotificationHandler) GetImage(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		c.Abort()
		return
	}

	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	img, err := h.repo.FindNotificationImage(id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Permission check
	if img.NotificationID != nil {
		// Image is linked to a notification — verify user can see that notification
		groupID := userGroupID(user)
		if _, err := h.repo.FindNotificationForUser(*img.NotificationID, user.ID, groupID, user.CreatedAt.Format("2006-01-02T15:04:05Z07:00")); err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	} else {
		// Orphan image — only uploader can access
		if img.CreatedBy != user.ID {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	}

	c.Header("Content-Type", "image/webp")
	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.Header("Content-Length", fmt.Sprintf("%d", img.Size))
	c.Data(http.StatusOK, "image/webp", img.Data)
}

// ValidateNotificationContent validates title and content length
func ValidateNotificationContent(title, content string) (string, bool) {
	if len(content) > 100*1024 {
		return "error.notificationContentTooLarge", false
	}
	if utf8.RuneCountInString(title) > 50 {
		return "error.notificationTitleTooLong", false
	}
	plainText := stripHTML(content)
	if utf8.RuneCountInString(plainText) > 1024 {
		return "error.notificationContentTooLong", false
	}
	if title == "" {
		return "error.notificationTitleRequired", false
	}
	if content == "" {
		return "error.notificationContentRequired", false
	}
	return "", true
}
