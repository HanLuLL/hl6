package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/auth"
	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
)

const (
	restoreChallengeTTL       = 5 * time.Minute
	maxRestoreUploadBytes     = int64(2 << 30)
	restoreConfirmationPhrase = "RESTORE DATABASE"
)

type MaintenanceHandler struct {
	repo *repository.Repository
	cfg  *config.Config
	svc  *service.DatabaseMaintenanceService
}

func NewMaintenanceHandler(repo *repository.Repository, cfg *config.Config, svc *service.DatabaseMaintenanceService) *MaintenanceHandler {
	return &MaintenanceHandler{repo: repo, cfg: cfg, svc: svc}
}

func (h *MaintenanceHandler) Export(c *gin.Context) {
	admin := mustGetUser(c)
	if admin == nil || h.svc == nil {
		return
	}
	backup, err := h.svc.CreateBackup(c.Request.Context(), admin.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database export failed", "error.databaseError")
		return
	}
	h.recordAudit(admin.ID, "admin_database_export", "database_backup", backup.ID, map[string]string{"checksum": backup.ChecksumSHA256})
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.FileAttachment(backup.StoragePath, backup.Filename)
}

func (h *MaintenanceHandler) CreateRestoreChallenge(c *gin.Context) {
	admin := mustGetUser(c)
	if admin == nil {
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid restore challenge request", "error.invalidRequestBody")
		return
	}
	if !h.verifyCurrentPassword(admin.ID, body.Password) {
		response.ErrorWithKey(c, http.StatusUnauthorized, "current password is invalid", "error.invalidToken")
		return
	}
	email, err := auth.NormalizeEmail(admin.Email)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "admin email is invalid", "error.invalidRequestBody")
		return
	}
	challenge, err := auth.NewRawToken()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create restore challenge", "error.databaseError")
		return
	}
	expiresAt := time.Now().UTC().Add(restoreChallengeTTL)
	if err := h.repo.CreateAuthToken(&model.AuthToken{
		Purpose:         model.AuthTokenPurposeRestoreChallenge,
		UserID:          &admin.ID,
		EmailNormalized: email,
		TokenHash:       auth.HashToken(challenge),
		ExpiresAt:       expiresAt,
	}); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save restore challenge", "error.databaseError")
		return
	}
	h.recordAudit(admin.ID, "admin_database_restore_challenge", "database_restore", 0, nil)
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	response.OK(c, gin.H{"challenge": challenge, "expires_at": expiresAt})
}

func (h *MaintenanceHandler) Restore(c *gin.Context) {
	admin := mustGetUser(c)
	if admin == nil || h.svc == nil {
		return
	}
	// The request must pass the confirmation and fresh-password gate before
	// the multipart file is opened or copied into durable maintenance storage.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRestoreUploadBytes)
	confirmation := c.PostForm("confirmation")
	password := c.PostForm("password")
	challenge := strings.TrimSpace(c.PostForm("challenge"))
	if !validateRestoreConfirmation(confirmation) {
		response.ErrorWithKey(c, http.StatusBadRequest, "restore confirmation does not match", "error.invalidRequestBody")
		return
	}
	if !h.verifyCurrentPassword(admin.ID, password) {
		response.ErrorWithKey(c, http.StatusUnauthorized, "current password is invalid", "error.invalidToken")
		return
	}
	if challenge == "" || len(challenge) > 256 {
		response.ErrorWithKey(c, http.StatusBadRequest, "restore challenge is invalid", "error.invalidRequestBody")
		return
	}
	challengeHash := auth.HashToken(challenge)
	token, err := h.repo.ConsumeAuthToken(c.Request.Context(), challengeHash, model.AuthTokenPurposeRestoreChallenge)
	if err != nil || token.UserID == nil || *token.UserID != admin.ID {
		response.ErrorWithKey(c, http.StatusBadRequest, "restore challenge is invalid or expired", "error.invalidToken")
		return
	}
	fileHeader, err := c.FormFile("archive")
	if err != nil || fileHeader.Size <= 0 || fileHeader.Size > maxRestoreUploadBytes {
		response.ErrorWithKey(c, http.StatusBadRequest, "database archive is invalid", "error.invalidRequestBody")
		return
	}
	uploaded, err := fileHeader.Open()
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "database archive is unreadable", "error.invalidRequestBody")
		return
	}
	defer uploaded.Close()
	archivePath, err := h.svc.StoreUploadedArchive(c.Request.Context(), uploaded)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "database archive is invalid", "error.invalidRequestBody")
		return
	}
	defer func() { _ = os.Remove(archivePath) }()
	job, err := h.svc.Restore(c.Request.Context(), admin.ID, challengeHash, archivePath)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrUnsafeArchive) || errors.Is(err, service.ErrInvalidBackupArchive) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, service.ErrRestoreInProgress) {
			status = http.StatusConflict
		}
		response.ErrorWithKey(c, status, "database restore failed", "error.databaseError")
		return
	}
	response.OK(c, gin.H{
		"restore":          job,
		"restart_required": true,
		"maintenance_mode": h.svc.Gate().IsRestoring(),
	})
}

func (h *MaintenanceHandler) ListRestores(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	jobs, total, err := h.repo.ListDatabaseRestoreJobs(page, perPage)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list database restore jobs", "error.databaseError")
		return
	}
	response.Paginated(c, gin.H{"items": jobs, "maintenance_mode": h.svc != nil && h.svc.Gate().IsRestoring()}, total, page, perPage)
}

func validateRestoreConfirmation(value string) bool {
	return value == restoreConfirmationPhrase
}

func (h *MaintenanceHandler) verifyCurrentPassword(userID uint, password string) bool {
	if h == nil || h.repo == nil || h.cfg == nil {
		return false
	}
	credential, err := h.repo.FindCredentialByUserID(userID)
	if err != nil || credential.PasswordSetAt == nil || credential.ActivationRequiredAt != nil {
		return false
	}
	valid, _, err := auth.VerifyPassword(password, credential.PasswordHash, auth.PepperSet{
		CurrentID:  strings.TrimSpace(h.cfg.AuthPasswordPepperID),
		Current:    []byte(h.cfg.AuthPasswordPepper),
		PreviousID: strings.TrimSpace(h.cfg.AuthPreviousPepperID),
		Previous:   []byte(h.cfg.AuthPreviousPepper),
	})
	return err == nil && valid
}

func (h *MaintenanceHandler) recordAudit(userID uint, action, resource string, resourceID uint, details interface{}) {
	if h == nil || h.repo == nil || userID == 0 {
		return
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		return
	}
	_ = h.repo.CreateAuditLog(&model.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    encoded,
	})
}
