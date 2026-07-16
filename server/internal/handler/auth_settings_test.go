package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/config"
)

func TestUpdateAccessSettingsRejectsWildcardDomain(t *testing.T) {
	repo, _ := newEmailAuthHandlerForTest(t)
	handler := NewAdminHandler(repo, &config.Config{}, nil, nil)
	router := gin.New()
	router.PUT("/api/v1/admin/settings/access", handler.UpdateAccessSettings)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/settings/access",
		bytes.NewBufferString(`{"registration_enabled":true,"domain_policy_mode":"allowlist","domain_policy_domains":["*.example.com"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}
