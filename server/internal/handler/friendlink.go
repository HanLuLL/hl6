package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type FriendLinkHandler struct {
	repo *repository.Repository
}

func NewFriendLinkHandler(repo *repository.Repository) *FriendLinkHandler {
	return &FriendLinkHandler{repo: repo}
}

// friendLinkResponse 对外返回结构（与 model 基本一致，预留扩展空间）
type friendLinkResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

func toFriendLinkResponse(l model.FriendLink) friendLinkResponse {
	return friendLinkResponse{
		ID:          l.ID,
		Name:        l.Name,
		URL:         l.URL,
		Description: l.Description,
		LogoURL:     l.LogoURL,
		SortOrder:   l.SortOrder,
		IsActive:    l.IsActive,
	}
}

// GetPublicFriendLinks 前台公开接口：返回启用的友链列表。
func (h *FriendLinkHandler) GetPublicFriendLinks(c *gin.Context) {
	links, err := h.repo.ListPublicFriendLinks()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load friend links", "error.databaseError")
		return
	}
	result := make([]friendLinkResponse, 0, len(links))
	for _, l := range links {
		result = append(result, toFriendLinkResponse(l))
	}
	response.OK(c, result)
}

// AdminList 后台：返回全部友链。
func (h *FriendLinkHandler) AdminList(c *gin.Context) {
	links, err := h.repo.ListAllFriendLinks()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load friend links", "error.databaseError")
		return
	}
	result := make([]friendLinkResponse, 0, len(links))
	for _, l := range links {
		result = append(result, toFriendLinkResponse(l))
	}
	response.OK(c, result)
}

// AdminCreate 后台：创建友链。
func (h *FriendLinkHandler) AdminCreate(c *gin.Context) {
	admin := mustGetUser(c)
	if admin == nil {
		return
	}

	var body struct {
		Name        string `json:"name" binding:"required"`
		URL         string `json:"url" binding:"required"`
		Description string `json:"description"`
		LogoURL     string `json:"logo_url"`
		SortOrder   int    `json:"sort_order"`
		IsActive    bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	name := strings.TrimSpace(body.Name)
	url := strings.TrimSpace(body.URL)
	if name == "" || url == "" {
		response.ErrorWithKey(c, http.StatusBadRequest, "name and url required", "error.invalidRequestBody")
		return
	}

	link := &model.FriendLink{
		Name:        name,
		URL:         url,
		Description: strings.TrimSpace(body.Description),
		LogoURL:     strings.TrimSpace(body.LogoURL),
		SortOrder:   body.SortOrder,
		IsActive:    body.IsActive,
	}
	if err := h.repo.CreateFriendLink(link); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create friend link", "error.databaseError")
		return
	}

	details, _ := json.Marshal(map[string]interface{}{
		"name": link.Name,
		"url":  link.URL,
	})
	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:   admin.ID,
		Action:   "admin_create_friend_link",
		Resource: "friend_link",
		ResourceID: link.ID,
		Details:  details,
	})

	response.Created(c, toFriendLinkResponse(*link))
}

// AdminUpdate 后台：更新友链。
func (h *FriendLinkHandler) AdminUpdate(c *gin.Context) {
	admin := mustGetUser(c)
	if admin == nil {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid id", "error.invalidRequestBody")
		return
	}

	link, err := h.repo.FindFriendLink(uint(id))
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "friend link not found", "error.notFound")
		return
	}

	var body struct {
		Name        *string `json:"name"`
		URL         *string `json:"url"`
		Description *string `json:"description"`
		LogoURL     *string `json:"logo_url"`
		SortOrder   *int    `json:"sort_order"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	if body.Name != nil {
		name := strings.TrimSpace(*body.Name)
		if name == "" {
			response.ErrorWithKey(c, http.StatusBadRequest, "name required", "error.invalidRequestBody")
			return
		}
		link.Name = name
	}
	if body.URL != nil {
		url := strings.TrimSpace(*body.URL)
		if url == "" {
			response.ErrorWithKey(c, http.StatusBadRequest, "url required", "error.invalidRequestBody")
			return
		}
		link.URL = url
	}
	if body.Description != nil {
		link.Description = strings.TrimSpace(*body.Description)
	}
	if body.LogoURL != nil {
		link.LogoURL = strings.TrimSpace(*body.LogoURL)
	}
	if body.SortOrder != nil {
		link.SortOrder = *body.SortOrder
	}
	if body.IsActive != nil {
		link.IsActive = *body.IsActive
	}

	if err := h.repo.UpdateFriendLink(link); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update friend link", "error.databaseError")
		return
	}

	details, _ := json.Marshal(map[string]interface{}{
		"name": link.Name,
		"url":  link.URL,
	})
	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:   admin.ID,
		Action:   "admin_update_friend_link",
		Resource: "friend_link",
		ResourceID: link.ID,
		Details:  details,
	})

	response.OK(c, toFriendLinkResponse(*link))
}

// AdminDelete 后台：删除友链。
func (h *FriendLinkHandler) AdminDelete(c *gin.Context) {
	admin := mustGetUser(c)
	if admin == nil {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid id", "error.invalidRequestBody")
		return
	}

	link, err := h.repo.FindFriendLink(uint(id))
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "friend link not found", "error.notFound")
		return
	}

	if err := h.repo.DeleteFriendLink(uint(id)); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete friend link", "error.databaseError")
		return
	}

	details, _ := json.Marshal(map[string]interface{}{
		"name": link.Name,
		"url":  link.URL,
	})
	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:   admin.ID,
		Action:   "admin_delete_friend_link",
		Resource: "friend_link",
		ResourceID: uint(id),
		Details:  details,
	})

	response.OK(c, gin.H{"deleted": true})
}
