package handler

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

const (
	brandNameConfigKey       = "brand_name"
	defaultBrandName         = "SubDomain"
	brandingAPIPrefix        = "/api/v1/branding"
	maxUploadImageSize       = 20 * 1024 * 1024
	maxEncodedWebPSize       = 2 * 1024 * 1024
	maxImagePixels     int64 = 64_000_000
	faviconSize              = 64
)

type BrandingHandler struct {
	repo        *repository.Repository
	urlResolver *URLResolver
}

type brandingResponse struct {
	Name       string  `json:"name"`
	LogoURL    *string `json:"logo_url"`
	FaviconURL *string `json:"favicon_url"`
	Version    string  `json:"version"`
}

func NewBrandingHandler(repo *repository.Repository, cfg *config.Config) *BrandingHandler {
	return &BrandingHandler{
		repo:        repo,
		urlResolver: NewURLResolver(repo, cfg),
	}
}

func (h *BrandingHandler) GetBranding(c *gin.Context) {
	branding, err := h.loadBranding(c)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load branding", "error.databaseError")
		return
	}
	response.OK(c, branding)
}

func (h *BrandingHandler) GetLogo(c *gin.Context) {
	h.serveBrandingAsset(c, model.BrandingAssetTypeLogoWebP, "image/webp")
}

func (h *BrandingHandler) GetFavicon(c *gin.Context) {
	h.serveBrandingAsset(c, model.BrandingAssetTypeFaviconICO, "image/x-icon")
}

func (h *BrandingHandler) AdminUpdateBranding(c *gin.Context) {
	admin := mustGetUser(c)
	if admin == nil {
		return
	}

	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		response.ErrorWithKey(c, http.StatusBadRequest, "brand name required", "error.invalidRequestBody")
		return
	}

	if err := h.repo.SetSystemConfig(brandNameConfigKey, name); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update branding name", "error.failedToUpdateConfig")
		return
	}

	details, _ := json.Marshal(map[string]interface{}{
		"name": name,
	})
	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:   admin.ID,
		Action:   "admin_update_branding_name",
		Resource: "branding",
		Details:  details,
	})

	branding, err := h.loadBranding(c)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load branding", "error.databaseError")
		return
	}
	response.OK(c, branding)
}

func (h *BrandingHandler) AdminUploadLogo(c *gin.Context) {
	admin := mustGetUser(c)
	if admin == nil {
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "file required", "error.fileRequired")
		return
	}
	if file.Size > maxUploadImageSize {
		response.ErrorWithKey(c, http.StatusBadRequest, "file too large (max 20MB)", "error.fileTooLarge")
		return
	}

	f, err := file.Open()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to open file", "error.failedToOpenFile")
		return
	}
	defer f.Close()

	imgCfg, _, err := image.DecodeConfig(f)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid image format", "error.invalidImageFormat")
		return
	}

	width := int64(imgCfg.Width)
	height := int64(imgCfg.Height)
	if width <= 0 || height <= 0 || width > maxImagePixels || height > maxImagePixels || width > maxImagePixels/height {
		response.ErrorWithKey(c, http.StatusBadRequest, "image dimensions too large", "error.imageDimensionsTooLarge")
		return
	}

	if seeker, ok := f.(io.ReadSeeker); ok {
		seeker.Seek(0, 0)
	}

	img, _, err := image.Decode(f)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid image format", "error.invalidImageFormat")
		return
	}

	logoWebP, err := encodeToWebPWithLimit(img, maxEncodedWebPSize)
	if err != nil {
		if errors.Is(err, errImageTooLargeAfterCompression) {
			response.ErrorWithKey(c, http.StatusBadRequest, "image too large after compression", "error.imageTooLargeAfterCompression")
			return
		}
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encode image", "error.failedToEncodeImage")
		return
	}

	faviconData, err := encodeToICO(img, faviconSize)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encode image", "error.failedToEncodeImage")
		return
	}

	if err := h.repo.UpsertBrandingAsset(model.BrandingAssetTypeLogoWebP, logoWebP); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save image", "error.failedToSaveImage")
		return
	}
	if err := h.repo.UpsertBrandingAsset(model.BrandingAssetTypeFaviconICO, faviconData); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save image", "error.failedToSaveImage")
		return
	}

	details, _ := json.Marshal(map[string]interface{}{
		"logo_size":    len(logoWebP),
		"favicon_size": len(faviconData),
	})
	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:   admin.ID,
		Action:   "admin_upload_branding_logo",
		Resource: "branding",
		Details:  details,
	})

	branding, err := h.loadBranding(c)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load branding", "error.databaseError")
		return
	}
	response.OK(c, branding)
}

func (h *BrandingHandler) AdminDeleteLogo(c *gin.Context) {
	admin := mustGetUser(c)
	if admin == nil {
		return
	}

	if err := h.repo.DeleteBrandingAssets([]string{
		model.BrandingAssetTypeLogoWebP,
		model.BrandingAssetTypeFaviconICO,
	}); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete logo", "error.databaseError")
		return
	}

	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:   admin.ID,
		Action:   "admin_delete_branding_logo",
		Resource: "branding",
	})

	branding, err := h.loadBranding(c)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load branding", "error.databaseError")
		return
	}
	response.OK(c, branding)
}

func (h *BrandingHandler) serveBrandingAsset(c *gin.Context, assetType, contentType string) {
	asset, err := h.repo.FindBrandingAssetByType(assetType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Content-Length", strconv.Itoa(asset.Size))
	c.Data(http.StatusOK, contentType, asset.Data)
}

func (h *BrandingHandler) loadBranding(c *gin.Context) (*brandingResponse, error) {
	name := defaultBrandName
	var latest time.Time
	urlState, err := h.urlResolver.Resolve(c)
	if err != nil {
		return nil, err
	}
	backendURL := strings.TrimRight(urlState.BackendURL, "/")

	cfg, err := h.repo.FindSystemConfig(brandNameConfigKey)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		if trimmed := strings.TrimSpace(cfg.Value); trimmed != "" {
			name = trimmed
		}
		latest = maxTime(latest, cfg.UpdatedAt)
	}

	assets, err := h.repo.ListBrandingAssets([]string{
		model.BrandingAssetTypeLogoWebP,
		model.BrandingAssetTypeFaviconICO,
	})
	if err != nil {
		return nil, err
	}

	hasLogo := false
	hasFavicon := false
	for _, asset := range assets {
		latest = maxTime(latest, asset.UpdatedAt)
		switch asset.AssetType {
		case model.BrandingAssetTypeLogoWebP:
			hasLogo = true
		case model.BrandingAssetTypeFaviconICO:
			hasFavicon = true
		}
	}

	version := "0"
	if !latest.IsZero() {
		version = strconv.FormatInt(latest.UnixNano(), 10)
	}

	var logoURL *string
	if hasLogo {
		url := fmt.Sprintf("%s%s/logo.webp?v=%s", backendURL, brandingAPIPrefix, version)
		logoURL = &url
	}

	var faviconURL *string
	if hasFavicon {
		url := fmt.Sprintf("%s%s/favicon.ico?v=%s", backendURL, brandingAPIPrefix, version)
		faviconURL = &url
	}

	return &brandingResponse{
		Name:       name,
		LogoURL:    logoURL,
		FaviconURL: faviconURL,
		Version:    version,
	}, nil
}

func maxTime(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.After(a) {
		return b
	}
	return a
}

var errImageTooLargeAfterCompression = errors.New("image too large after compression")

func encodeToWebPWithLimit(img image.Image, maxSize int) ([]byte, error) {
	qualities := []float32{80, 60, 40, 20}
	var data []byte

	for _, quality := range qualities {
		var buf bytes.Buffer
		if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: quality}); err != nil {
			return nil, err
		}
		data = buf.Bytes()
		if len(data) <= maxSize {
			return data, nil
		}
	}

	return nil, errImageTooLargeAfterCompression
}

func encodeToICO(src image.Image, size int) ([]byte, error) {
	if size <= 0 {
		return nil, errors.New("invalid icon size")
	}

	resized := resizeNearest(src, size, size)
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, resized); err != nil {
		return nil, err
	}
	pngData := pngBuf.Bytes()

	var ico bytes.Buffer

	if err := binary.Write(&ico, binary.LittleEndian, uint16(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(&ico, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(&ico, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}

	width := byte(size)
	height := byte(size)
	if size >= 256 {
		width = 0
		height = 0
	}

	if err := ico.WriteByte(width); err != nil {
		return nil, err
	}
	if err := ico.WriteByte(height); err != nil {
		return nil, err
	}
	if err := ico.WriteByte(0); err != nil {
		return nil, err
	}
	if err := ico.WriteByte(0); err != nil {
		return nil, err
	}

	if err := binary.Write(&ico, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(&ico, binary.LittleEndian, uint16(32)); err != nil {
		return nil, err
	}
	if err := binary.Write(&ico, binary.LittleEndian, uint32(len(pngData))); err != nil {
		return nil, err
	}
	if err := binary.Write(&ico, binary.LittleEndian, uint32(6+16)); err != nil {
		return nil, err
	}

	if _, err := ico.Write(pngData); err != nil {
		return nil, err
	}

	return ico.Bytes(), nil
}

func resizeNearest(src image.Image, targetWidth, targetHeight int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	if srcWidth <= 0 || srcHeight <= 0 {
		return dst
	}

	for y := 0; y < targetHeight; y++ {
		srcY := bounds.Min.Y + (y * srcHeight / targetHeight)
		for x := 0; x < targetWidth; x++ {
			srcX := bounds.Min.X + (x * srcWidth / targetWidth)
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}

	return dst
}
