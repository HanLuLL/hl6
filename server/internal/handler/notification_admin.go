package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strconv"

	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type NotificationAdminHandler struct {
	repo   *repository.Repository
	broker *SSEBroker
}

func NewNotificationAdminHandler(repo *repository.Repository, broker *SSEBroker) *NotificationAdminHandler {
	return &NotificationAdminHandler{repo: repo, broker: broker}
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

	if err := h.repo.CreateNotification(notification); err != nil {
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
	userIDs, _ := h.repo.GetNotificationTargetUserIDs(notification)

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
	if notification.TargetType == "all" {
		h.broker.SendToAll(event)
	} else if userIDs != nil {
		h.broker.SendToUsers(userIDs, event)
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

	// Decode image
	img, _, err := image.Decode(f)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid image format", "error.invalidImageFormat")
		return
	}

	// Encode to WebP
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: 80}); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encode image", "error.failedToEncodeImage")
		return
	}

	data := buf.Bytes()
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
		"url": fmt.Sprintf("/api/v1/notifications/images/%d", notifImage.ID),
	})
}
