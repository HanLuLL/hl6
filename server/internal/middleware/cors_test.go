package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAllowsNativeMutationHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS(nil))
	router.OPTIONS("/api/v1/auth/native/start", func(c *gin.Context) {})

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/native/start", nil)
	request.Header.Set("Origin", "https://localhost")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type, x-hl6-client-key, x-idempotency-key")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://localhost" {
		t.Fatalf("expected Capacitor origin to be allowed, got %q", got)
	}

	allowedHeaders := strings.ToLower(recorder.Header().Get("Access-Control-Allow-Headers"))
	for _, header := range []string{"content-type", "x-hl6-client-key", "x-idempotency-key"} {
		if !strings.Contains(allowedHeaders, header) {
			t.Errorf("expected CORS response to allow %q, got %q", header, allowedHeaders)
		}
	}
}
