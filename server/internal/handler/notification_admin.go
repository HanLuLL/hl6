package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"hl6-server/internal/config"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type NotificationAdminHandler struct {
	repo   *repository.Repository
	broker *SSEBroker
	cfg    *config.Config
}

func NewNotificationAdminHandler(repo *repository.Repository, broker *SSEBroker, cfg *config.Config) *NotificationAdminHandler {
	return &NotificationAdminHandler{repo: repo, broker: broker, cfg: cfg}
}

func (h *NotificationAdminHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "15"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 15
	}

	notifications, total, err := h.repo.ListNotificationsAdmin(page, perPage)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list notifications", "error.failedToListNotifications")
		return
	}
	response.Paginated(c, notifications, total, page, perPage)
}

func (h *NotificationAdminHandler) Create(c *gin.Context) {
	admin := ctxutil.GetUser(c)
	if admin == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
		return
	}

	var body struct {
		Title        string          `json:"title" binding:"required"`
		Content      string          `json:"content" binding:"required"`
		Type         string          `json:"type" binding:"required"`
		TargetType   string          `json:"target_type" binding:"required"`
		TargetIDs    json.RawMessage `json:"target_ids"`
		VisibleToNew bool            `json:"visible_to_new"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	// Validate type
	if body.Type != "normal" && body.Type != "urgent" && body.Type != "pinned" {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid notification type", "error.invalidNotificationType")
		return
	}

	// Validate target_type
	if body.TargetType != "users" && body.TargetType != "groups" && body.TargetType != "all" {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid target type", "error.invalidTargetType")
		return
	}

	// Validate target_ids for users and groups
	if body.TargetType == "users" || body.TargetType == "groups" {
		if len(body.TargetIDs) == 0 || string(body.TargetIDs) == "null" {
			response.ErrorWithKey(c, http.StatusBadRequest, "target_ids required", "error.targetIDsRequired")
			return
		}
		var ids []uint
		if err := json.Unmarshal(body.TargetIDs, &ids); err != nil || len(ids) == 0 {
			response.ErrorWithKey(c, http.StatusBadRequest, "target_ids required", "error.targetIDsRequired")
			return
		}
	}

	// Validate content length
	if errKey, ok := ValidateNotificationContent(body.Title, body.Content); !ok {
		response.ErrorWithKey(c, http.StatusBadRequest, "validation failed", errKey)
		return
	}

	notification := &model.Notification{
		Title:        body.Title,
		Content:      body.Content,
		Type:         body.Type,
		TargetType:   body.TargetType,
		TargetIDs:    body.TargetIDs,
		VisibleToNew: body.VisibleToNew,
		CreatedBy:    admin.ID,
	}

	if err := h.repo.CreateNotificationWithImages(notification); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create notification", "error.failedToCreateNotification")
		return
	}

	// Audit log
	details, _ := json.Marshal(map[string]interface{}{
		"title":       body.Title,
		"type":        body.Type,
		"target_type": body.TargetType,
	})
	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:     admin.ID,
		Action:     "admin_create_notification",
		Resource:   "notification",
		ResourceID: notification.ID,
		Details:    details,
	})

	// Send SSE event
	event := SSEEvent{Event: "new_notification", Data: fmt.Sprintf(`{"id":%d}`, notification.ID)}
	userIDs, err := h.repo.GetNotificationTargetUserIDs(notification)
	if err == nil {
		if notification.TargetType == "all" {
			h.broker.SendToAll(event)
		} else {
			h.broker.SendToUsers(userIDs, event)
		}
	}

	response.Created(c, notification)
}

func (h *NotificationAdminHandler) Update(c *gin.Context) {
	admin := ctxutil.GetUser(c)
	if admin == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
		return
	}

	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	notification, err := h.repo.FindNotification(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "notification not found", "error.notificationNotFound")
		return
	}

	var body struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Type    string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	// Validate type
	if body.Type != "normal" && body.Type != "urgent" && body.Type != "pinned" {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid notification type", "error.invalidNotificationType")
		return
	}

	// Validate content
	if errKey, ok := ValidateNotificationContent(body.Title, body.Content); !ok {
		response.ErrorWithKey(c, http.StatusBadRequest, "validation failed", errKey)
		return
	}

	notification.Title = body.Title
	notification.Content = body.Content
	notification.Type = body.Type

	if err := h.repo.UpdateNotificationWithImages(notification); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update notification", "error.failedToUpdateNotification")
		return
	}

	// Audit log
	details, _ := json.Marshal(map[string]interface{}{
		"title": body.Title,
		"type":  body.Type,
	})
	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:     admin.ID,
		Action:     "admin_update_notification",
		Resource:   "notification",
		ResourceID: id,
		Details:    details,
	})

	// Send SSE event to target users
	event := SSEEvent{Event: "update_notification", Data: fmt.Sprintf(`{"id":%d}`, id)}
	userIDs, err := h.repo.GetNotificationTargetUserIDs(notification)
	if err == nil {
		if notification.TargetType == "all" {
			h.broker.SendToAll(event)
		} else {
			h.broker.SendToUsers(userIDs, event)
		}
	}

	response.OK(c, notification)
}

func (h *NotificationAdminHandler) Delete(c *gin.Context) {
	admin := ctxutil.GetUser(c)
	if admin == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
		return
	}

	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	notification, err := h.repo.FindNotification(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "notification not found", "error.notificationNotFound")
		return
	}

	// Resolve target users before deletion for SSE
	event := SSEEvent{Event: "delete_notification", Data: fmt.Sprintf(`{"id":%d}`, id)}
	userIDs, targErr := h.repo.GetNotificationTargetUserIDs(notification)

	if err := h.repo.DeleteNotification(id); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete notification", "error.failedToDeleteNotification")
		return
	}

	// Audit log
	details, _ := json.Marshal(map[string]interface{}{
		"title": notification.Title,
		"type":  notification.Type,
	})
	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:     admin.ID,
		Action:     "admin_delete_notification",
		Resource:   "notification",
		ResourceID: id,
		Details:    details,
	})

	// Send SSE event
	if targErr == nil {
		if notification.TargetType == "all" {
			h.broker.SendToAll(event)
		} else if len(userIDs) > 0 {
			h.broker.SendToUsers(userIDs, event)
		}
	}

	response.OK(c, gin.H{"message": "notification deleted"})
}

func (h *NotificationAdminHandler) UploadImage(c *gin.Context) {
	admin := ctxutil.GetUser(c)
	if admin == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "file required", "error.fileRequired")
		return
	}

	// 20MB limit
	if file.Size > 20*1024*1024 {
		response.ErrorWithKey(c, http.StatusBadRequest, "file too large (max 20MB)", "error.fileTooLarge")
		return
	}

	f, err := file.Open()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to open file", "error.failedToOpenFile")
		return
	}
	defer f.Close()

	// Check image dimensions first
	imgCfg, _, err := image.DecodeConfig(f)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid image format", "error.invalidImageFormat")
		return
	}
	const maxPixels int64 = 64_000_000
	width := int64(imgCfg.Width)
	height := int64(imgCfg.Height)
	if width <= 0 || height <= 0 || width > maxPixels || height > maxPixels || width > maxPixels/height {
		response.ErrorWithKey(c, http.StatusBadRequest, "image dimensions too large", "error.imageDimensionsTooLarge")
		return
	}

	// Seek back to start for full decode
	if seeker, ok := f.(io.ReadSeeker); ok {
		seeker.Seek(0, 0)
	}

	// Decode image
	img, _, err := image.Decode(f)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid image format", "error.invalidImageFormat")
		return
	}

	// Encode to WebP with progressive quality reduction
	const maxSize = 2 * 1024 * 1024 // 2MB
	qualities := []float32{80, 60, 40, 20}
	var data []byte
	for _, q := range qualities {
		var buf bytes.Buffer
		if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: q}); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encode image", "error.failedToEncodeImage")
			return
		}
		data = buf.Bytes()
		if len(data) <= maxSize {
			break
		}
	}
	if len(data) > maxSize {
		response.ErrorWithKey(c, http.StatusBadRequest, "image too large after compression", "error.imageTooLargeAfterCompression")
		return
	}

	notifImage := &model.NotificationImage{
		Data:      data,
		Size:      len(data),
		CreatedBy: admin.ID,
	}

	if err := h.repo.CreateNotificationImage(notifImage); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save image", "error.failedToSaveImage")
		return
	}

	response.Created(c, gin.H{
		"id":  notifImage.ID,
		"url": fmt.Sprintf("%s/api/v1/notifications/images/%d", strings.TrimRight(h.cfg.BackendURL, "/"), notifImage.ID),
	})
}
