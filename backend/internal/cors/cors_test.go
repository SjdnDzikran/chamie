package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareAllowsAllowedOrigin(t *testing.T) {
	handler := Middleware([]string{"https://chamie.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chat/history", nil)
	req.Header.Set("Origin", "https://chamie.example.com")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://chamie.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Result().StatusCode; got != http.StatusOK {
		t.Errorf("status = %d", got)
	}
}

func TestMiddlewareOmitsHeaderForDisallowedOrigin(t *testing.T) {
	handler := Middleware([]string{"https://chamie.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chat/history", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS header for disallowed origin, got %q", got)
	}
	if got := rec.Result().StatusCode; got != http.StatusOK {
		t.Errorf("request should still be served, status = %d", got)
	}
}

func TestMiddlewarePreflightReturnsNoContent(t *testing.T) {
	handler := Middleware([]string{"https://chamie.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("downstream handler should not be called for preflight")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/chat/stream", nil)
	req.Header.Set("Origin", "https://chamie.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	handler.ServeHTTP(rec, req)

	if got := rec.Result().StatusCode; got != http.StatusNoContent {
		t.Errorf("preflight status = %d, want %d", got, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Access-Control-Allow-Methods missing on preflight")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("Access-Control-Allow-Headers missing on preflight")
	}
}

func TestMiddlewareEmptyAllowListIsNoOp(t *testing.T) {
	called := false
	handler := Middleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chat/history", nil)
	req.Header.Set("Origin", "https://anything.example.com")
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("downstream handler should still be called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS header, got %q", got)
	}
}

func TestMiddlewareIgnoresRequestsWithoutOrigin(t *testing.T) {
	handler := Middleware([]string{"https://chamie.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS header without Origin, got %q", got)
	}
}
