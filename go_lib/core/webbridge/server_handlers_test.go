package webbridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerHealthEndpointReturnsSuccessEnvelope(t *testing.T) {
	server := NewServer("")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode health payload: %v", err)
	}
	if payload["success"] != true {
		t.Fatalf("expected success=true, got %#v", payload)
	}
}

func TestServerStaticServesIndexFallbackForSPARoutes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("index page"), 0600); err != nil {
		t.Fatalf("failed to write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "asset.txt"), []byte("asset"), 0600); err != nil {
		t.Fatalf("failed to write asset: %v", err)
	}

	server := NewServer(root)
	handler := server.Handler()

	assetReq := httptest.NewRequest(http.MethodGet, "/asset.txt", nil)
	assetRR := httptest.NewRecorder()
	handler.ServeHTTP(assetRR, assetReq)
	if assetRR.Code != http.StatusOK || strings.TrimSpace(assetRR.Body.String()) != "asset" {
		t.Fatalf("expected asset response, got status=%d body=%q", assetRR.Code, assetRR.Body.String())
	}

	routeReq := httptest.NewRequest(http.MethodGet, "/nested/route", nil)
	routeRR := httptest.NewRecorder()
	handler.ServeHTTP(routeRR, routeReq)
	if routeRR.Code != http.StatusOK || !strings.Contains(routeRR.Body.String(), "index page") {
		t.Fatalf("expected SPA fallback, got status=%d body=%q", routeRR.Code, routeRR.Body.String())
	}
}
