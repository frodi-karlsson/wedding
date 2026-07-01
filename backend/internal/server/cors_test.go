package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_AllowsAllowedOrigin(t *testing.T) {
	mw := CORS([]string{"https://carlaochfrodi.wedding"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invites/1", nil)
	req.Header.Set("Origin", "https://carlaochfrodi.wedding")
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://carlaochfrodi.wedding" {
		t.Errorf("ACAO = %q, want %q", got, "https://carlaochfrodi.wedding")
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("ACAC = %q, want true", got)
	}
}

func TestCORS_RejectedOrigin(t *testing.T) {
	mw := CORS([]string{"https://carlaochfrodi.wedding"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invites/1", nil)
	req.Header.Set("Origin", "https://evil.com")
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO = %q, want empty for rejected origin", got)
	}
}

func TestCORS_PreflightReturns204(t *testing.T) {
	mw := CORS([]string{"https://carlaochfrodi.wedding"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called on preflight")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/invites/1", nil)
	req.Header.Set("Origin", "https://carlaochfrodi.wedding")
	req.Header.Set("Access-Control-Request-Method", "POST")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("ACAM header missing")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("ACAH header missing")
	}
}
