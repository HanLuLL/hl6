package handler

import (
        "encoding/json"
        "fmt"
        "net/http"
        "strconv"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
        "hl6-server/internal/model"
        "hl6-server/internal/repository"
        "hl6-server/pkg/crypto"
        "hl6-server/pkg/response"
)

type AIAuditHandler struct {
        repo          *repository.Repository
        encryptionKey []byte
}

func NewAIAuditHandler(repo *repository.Repository, encryptionKey []byte) *AIAuditHandler {
        return &AIAuditHandler{repo: repo, encryptionKey: encryptionKey}
}

func (h *AIAuditHandler) ListModelConfigs(c *gin.Context) {
        configs, err := h.repo.ListAIModelConfigs()
        if err != nil {
                response.InternalError(c, "failed to list AI model configs")
                return
        }
        type modelConfigView struct {
                model.AIModelConfig
                APIKey string `json:"api_key"`
        }
        result := make([]modelConfigView, len(configs))
        for i, cfg := range configs {
                result[i] = modelConfigView{
                        AIModelConfig: cfg,
                        APIKey:        maskAPIKey(cfg.APIKey, h.encryptionKey),
                }
        }
        response.OK(c, result)
}

func (h *AIAuditHandler) CreateModelConfig(c *gin.Context) {
        _, ok := requireUser(c)
        if !ok {
                return
        }
        var body struct {
                Name         string  `json:"name" binding:"required"`
                Provider     string  `json:"provider"`
                APIBaseURL   string  `json:"api_base_url" binding:"required"`
                APIKey       string  `json:"api_key" binding:"required"`
                ModelName    string  `json:"model_name" binding:"required"`
                IsDefault    bool    `json:"is_default"`
                IsEnabled    bool    `json:"is_enabled"`
                MaxTokens    int     `json:"max_tokens"`
                Temperature  float64 `json:"temperature"`
                RateLimitRPM int     `json:"rate_limit_rpm"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
                response.BadRequest(c, "invalid request body")
                return
        }

        encryptedKey, err := crypto.EncryptIfKey(body.APIKey, h.encryptionKey)
        if err != nil {
                response.InternalError(c, "failed to encrypt API key")
                return
        }

        if body.IsDefault {
                _ = h.repo.ClearDefaultAIModelConfigs()
        }

        if body.Provider == "" {
                body.Provider = "openai"
        }
        if body.MaxTokens == 0 {
                body.MaxTokens = 4096
        }
        if body.Temperature == 0 {
                body.Temperature = 0.1
        }
        if body.RateLimitRPM == 0 {
                body.RateLimitRPM = 60
        }

        config := &model.AIModelConfig{
                Name:         strings.TrimSpace(body.Name),
                Provider:     body.Provider,
                APIBaseURL:   strings.TrimSpace(body.APIBaseURL),
                APIKey:       encryptedKey,
                ModelName:    strings.TrimSpace(body.ModelName),
                IsDefault:    body.IsDefault,
                IsEnabled:    body.IsEnabled,
                MaxTokens:    body.MaxTokens,
                Temperature:  body.Temperature,
                RateLimitRPM: body.RateLimitRPM,
        }

        if err := h.repo.CreateAIModelConfig(config); err != nil {
                response.InternalError(c, "failed to create AI model config")
                return
        }

        auditLogIfAdmin(nil, c, "admin_create_ai_model_config", "ai_model_config", config.ID, map[string]any{
                "name": config.Name, "model": config.ModelName,
        })
        response.Created(c, gin.H{
                "id":         config.ID,
                "name":       config.Name,
                "model_name": config.ModelName,
        })
}

func (h *AIAuditHandler) UpdateModelConfig(c *gin.Context) {
        _, ok := requireUser(c)
        if !ok {
                return
        }
        id, err := strconv.ParseUint(c.Param("id"), 10, 64)
        if err != nil {
                response.BadRequest(c, "invalid id")
                return
        }

        config, err := h.repo.FindAIModelConfig(uint(id))
        if err != nil {
                response.NotFound(c, "AI model config not found")
                return
        }

        var body struct {
                Name         *string  `json:"name"`
                Provider     *string  `json:"provider"`
                APIBaseURL   *string  `json:"api_base_url"`
                APIKey       *string  `json:"api_key"`
                ModelName    *string  `json:"model_name"`
                IsDefault    *bool    `json:"is_default"`
                IsEnabled    *bool    `json:"is_enabled"`
                MaxTokens    *int     `json:"max_tokens"`
                Temperature  *float64 `json:"temperature"`
                RateLimitRPM *int     `json:"rate_limit_rpm"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
                response.BadRequest(c, "invalid request body")
                return
        }

        if body.Name != nil {
                config.Name = strings.TrimSpace(*body.Name)
        }
        if body.Provider != nil {
                config.Provider = *body.Provider
        }
        if body.APIBaseURL != nil {
                config.APIBaseURL = strings.TrimSpace(*body.APIBaseURL)
        }
        if body.APIKey != nil && *body.APIKey != "" && *body.APIKey != "********" {
                encryptedKey, encErr := crypto.EncryptIfKey(*body.APIKey, h.encryptionKey)
                if encErr != nil {
                        response.InternalError(c, "failed to encrypt API key")
                        return
                }
                config.APIKey = encryptedKey
        }
        if body.ModelName != nil {
                config.ModelName = strings.TrimSpace(*body.ModelName)
        }
        if body.IsDefault != nil && *body.IsDefault {
                _ = h.repo.ClearDefaultAIModelConfigs()
                config.IsDefault = true
        }
        if body.IsEnabled != nil {
                config.IsEnabled = *body.IsEnabled
        }
        if body.MaxTokens != nil {
                config.MaxTokens = *body.MaxTokens
        }
        if body.Temperature != nil {
                config.Temperature = *body.Temperature
        }
        if body.RateLimitRPM != nil {
                config.RateLimitRPM = *body.RateLimitRPM
        }

        if err := h.repo.UpdateAIModelConfig(config); err != nil {
                response.InternalError(c, "failed to update AI model config")
                return
        }

        auditLogIfAdmin(nil, c, "admin_update_ai_model_config", "ai_model_config", config.ID, map[string]any{
                "name": config.Name, "model": config.ModelName,
        })
        response.OK(c, gin.H{"updated": true})
}

func (h *AIAuditHandler) DeleteModelConfig(c *gin.Context) {
        id, err := strconv.ParseUint(c.Param("id"), 10, 64)
        if err != nil {
                response.BadRequest(c, "invalid id")
                return
        }
        if err := h.repo.DeleteAIModelConfig(uint(id)); err != nil {
                response.InternalError(c, "failed to delete AI model config")
                return
        }
        response.OK(c, gin.H{"deleted": true})
}

func (h *AIAuditHandler) ListPromptTemplates(c *gin.Context) {
        templates, err := h.repo.ListPromptTemplates()
        if err != nil {
                response.InternalError(c, "failed to list prompt templates")
                return
        }
        if templates == nil {
                templates = []model.AuditPromptTemplate{}
        }
        response.OK(c, templates)
}

func (h *AIAuditHandler) CreatePromptTemplate(c *gin.Context) {
        admin, ok := requireUser(c)
        if !ok {
                return
        }
        var body struct {
                Name         string `json:"name" binding:"required"`
                IsDefault    bool   `json:"is_default"`
                IsEnabled    bool   `json:"is_enabled"`
                SortOrder    int    `json:"sort_order"`
                SystemPrompt string `json:"system_prompt" binding:"required"`
                UserPrompt   string `json:"user_prompt" binding:"required"`
                Description  string `json:"description"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
                response.BadRequest(c, "invalid request body")
                return
        }

        template := &model.AuditPromptTemplate{
                Name:         strings.TrimSpace(body.Name),
                IsDefault:    body.IsDefault,
                IsEnabled:    body.IsEnabled,
                SortOrder:    body.SortOrder,
                SystemPrompt: body.SystemPrompt,
                UserPrompt:   body.UserPrompt,
                Description:  strings.TrimSpace(body.Description),
                CreatedBy:    admin.ID,
        }

        if err := h.repo.CreatePromptTemplate(template); err != nil {
                response.InternalError(c, "failed to create prompt template")
                return
        }

        auditLogIfAdmin(nil, c, "admin_create_prompt_template", "prompt_template", template.ID, map[string]any{
                "name": template.Name,
        })
        response.Created(c, template)
}

func (h *AIAuditHandler) UpdatePromptTemplate(c *gin.Context) {
        admin, ok := requireUser(c)
        if !ok {
                return
        }
        id, err := strconv.ParseUint(c.Param("id"), 10, 64)
        if err != nil {
                response.BadRequest(c, "invalid id")
                return
        }

        template, err := h.repo.FindPromptTemplate(uint(id))
        if err != nil {
                response.NotFound(c, "prompt template not found")
                return
        }

        var body struct {
                Name         *string `json:"name"`
                IsDefault    *bool   `json:"is_default"`
                IsEnabled    *bool   `json:"is_enabled"`
                SortOrder    *int    `json:"sort_order"`
                SystemPrompt *string `json:"system_prompt"`
                UserPrompt   *string `json:"user_prompt"`
                Description  *string `json:"description"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
                response.BadRequest(c, "invalid request body")
                return
        }

        if body.Name != nil {
                template.Name = strings.TrimSpace(*body.Name)
        }
        if body.IsDefault != nil {
                template.IsDefault = *body.IsDefault
        }
        if body.IsEnabled != nil {
                template.IsEnabled = *body.IsEnabled
        }
        if body.SortOrder != nil {
                template.SortOrder = *body.SortOrder
        }
        if body.SystemPrompt != nil {
                template.SystemPrompt = *body.SystemPrompt
        }
        if body.UserPrompt != nil {
                template.UserPrompt = *body.UserPrompt
        }
        if body.Description != nil {
                template.Description = strings.TrimSpace(*body.Description)
        }
        template.UpdatedBy = admin.ID

        if err := h.repo.UpdatePromptTemplate(template); err != nil {
                response.InternalError(c, "failed to update prompt template")
                return
        }

        response.OK(c, template)
}

func (h *AIAuditHandler) DeletePromptTemplate(c *gin.Context) {
        id, err := strconv.ParseUint(c.Param("id"), 10, 64)
        if err != nil {
                response.BadRequest(c, "invalid id")
                return
        }
        if err := h.repo.DeletePromptTemplate(uint(id)); err != nil {
                response.InternalError(c, "failed to delete prompt template")
                return
        }
        response.OK(c, gin.H{"deleted": true})
}

func (h *AIAuditHandler) ListAIReviews(c *gin.Context) {
        page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
        perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
        if page < 1 {
                page = 1
        }
        if perPage < 1 || perPage > 100 {
                perPage = 20
        }

        filter := repository.AuditAIReviewFilter{
                Judgments:     queryStringSlice(c, "judgment"),
                AdminStatuses: queryStringSlice(c, "admin_status"),
                FQDN:          strings.TrimSpace(c.Query("fqdn")),
        }
        if from := c.Query("from"); from != "" {
                if t, err := time.Parse(time.RFC3339, from); err == nil {
                        filter.From = t
                }
        }
        if to := c.Query("to"); to != "" {
                if t, err := time.Parse(time.RFC3339, to); err == nil {
                        filter.To = t
                }
        }

        reviews, total, err := h.repo.ListAuditAIReviews(page, perPage, filter)
        if err != nil {
                response.InternalError(c, "failed to list AI reviews")
                return
        }
        response.Paginated(c, reviews, total, page, perPage)
}

func (h *AIAuditHandler) GetAIReview(c *gin.Context) {
        id, err := strconv.ParseUint(c.Param("id"), 10, 64)
        if err != nil {
                response.BadRequest(c, "invalid id")
                return
        }
        review, err := h.repo.FindAuditAIReview(uint(id))
        if err != nil {
                response.NotFound(c, "AI review not found")
                return
        }
        response.OK(c, review)
}

func (h *AIAuditHandler) ReviewAIReview(c *gin.Context) {
        admin, ok := requireUser(c)
        if !ok {
                return
        }
        id, err := strconv.ParseUint(c.Param("id"), 10, 64)
        if err != nil {
                response.BadRequest(c, "invalid id")
                return
        }

        var body struct {
                Status string `json:"status" binding:"required"`
                Note   string `json:"note"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
                response.BadRequest(c, "invalid request body")
                return
        }

        validStatuses := map[string]bool{
                model.AdminReviewConfirmed:  true,
                model.AdminReviewOverturned: true,
                model.AdminReviewDismissed:  true,
        }
        if !validStatuses[body.Status] {
                response.BadRequest(c, "invalid status, must be confirmed/overturned/dismissed")
                return
        }

        if err := h.repo.UpdateAuditAIReviewAdminReview(uint(id), body.Status, body.Note, admin.ID); err != nil {
                response.InternalError(c, "failed to update AI review")
                return
        }

        auditLogIfAdmin(nil, c, "admin_review_ai_audit", "audit_ai_review", uint(id), map[string]any{
                "status": body.Status,
                "note":   body.Note,
        })
        response.OK(c, gin.H{"updated": true})
}

func (h *AIAuditHandler) GetAIStats(c *gin.Context) {
        stats, err := h.repo.GetAIAuditStats()
        if err != nil {
                response.InternalError(c, "failed to get AI audit stats")
                return
        }
        response.OK(c, stats)
}

func (h *AIAuditHandler) CreateUserAppeal(c *gin.Context) {
        user, ok := requireUser(c)
        if !ok {
                return
        }

        var body struct {
                Content string `json:"content" binding:"required"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
                response.BadRequest(c, "invalid request body")
                return
        }

        content := strings.TrimSpace(body.Content)
        if content == "" {
                response.BadRequest(c, "appeal content cannot be empty")
                return
        }

        hasPending, err := h.repo.HasPendingAppealByUser(user.ID)
        if err != nil {
                response.InternalError(c, "failed to check pending appeals")
                return
        }
        if hasPending {
                response.ErrorWithKey(c, http.StatusConflict, "you already have a pending appeal", "error.appealAlreadyPending")
                return
        }

        appeal := &model.UserAppeal{
                UserID:  user.ID,
                Content: content,
                Status:  model.AppealStatusPending,
        }
        if err := h.repo.CreateUserAppeal(appeal); err != nil {
                response.InternalError(c, "failed to create appeal")
                return
        }

        response.Created(c, gin.H{"id": appeal.ID})
}

func (h *AIAuditHandler) ListMyAppeals(c *gin.Context) {
        user, ok := requireUser(c)
        if !ok {
                return
        }
        appeals, err := h.repo.ListUserAppealsByUser(user.ID)
        if err != nil {
                response.InternalError(c, "failed to list appeals")
                return
        }
        response.OK(c, appeals)
}

func (h *AIAuditHandler) AdminListAppeals(c *gin.Context) {
        page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
        perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
        if page < 1 {
                page = 1
        }
        if perPage < 1 || perPage > 100 {
                perPage = 20
        }

        appeals, total, err := h.repo.ListUserAppeals(page, perPage)
        if err != nil {
                response.InternalError(c, "failed to list appeals")
                return
        }
        response.Paginated(c, appeals, total, page, perPage)
}

func (h *AIAuditHandler) AdminReviewAppeal(c *gin.Context) {
        admin, ok := requireUser(c)
        if !ok {
                return
        }
        id, err := strconv.ParseUint(c.Param("id"), 10, 64)
        if err != nil {
                response.BadRequest(c, "invalid id")
                return
        }

        var body struct {
                Status string `json:"status" binding:"required"`
                Reply  string `json:"reply"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
                response.BadRequest(c, "invalid request body")
                return
        }

        if body.Status != model.AppealStatusApproved && body.Status != model.AppealStatusRejected {
                response.BadRequest(c, "invalid status, must be approved/rejected")
                return
        }

        if err := h.repo.UpdateUserAppealReview(uint(id), body.Status, body.Reply, admin.ID); err != nil {
                response.InternalError(c, "failed to review appeal")
                return
        }

        if body.Status == model.AppealStatusApproved {
                appeal, _ := h.repo.FindUserAppeal(uint(id))
                if appeal != nil {
                        _ = h.repo.UnbanUser(appeal.UserID)
                }
        }

        auditLogIfAdmin(nil, c, "admin_review_appeal", "user_appeal", uint(id), map[string]any{
                "status": body.Status,
        })
        response.OK(c, gin.H{"updated": true})
}

func (h *AIAuditHandler) GetBanInfo(c *gin.Context) {
        user, ok := requireUser(c)
        if !ok {
                return
        }
        if !user.IsBanned {
                response.OK(c, gin.H{"banned": false})
                return
        }
        response.OK(c, gin.H{
                "banned":    true,
                "reason":    user.BannedReason,
                "banned_at": user.BannedAt,
        })
}

func maskAPIKey(encryptedKey string, encryptionKey []byte) string {
        if encryptedKey == "" {
                return ""
        }
        decrypted := crypto.DecryptOrPlaintext(encryptedKey, encryptionKey)
        if len(decrypted) > 8 {
                return "********" + decrypted[len(decrypted)-4:]
        }
        return "********"
}

var _ = json.Marshal
var _ = fmt.Sprintf