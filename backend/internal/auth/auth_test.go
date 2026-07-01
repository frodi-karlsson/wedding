package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogin_CorrectPassword(t *testing.T) {
	a := New("correct-horse", "session-secret", true)
	if !a.Login("correct-horse") {
		t.Error("Login with correct password returned false")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	a := New("correct-horse", "session-secret", true)
	if a.Login("wrong") {
		t.Error("Login with wrong password returned true")
	}
}

func TestSetSessionCookie_ThenIsAuthenticated(t *testing.T) {
	a := New("pw", "secret", true)
	rec := httptest.NewRecorder()
	a.SetSessionCookie(rec)

	req := httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	if !a.IsAuthenticated(req) {
		t.Error("IsAuthenticated returned false after SetSessionCookie")
	}
}

func TestIsAuthenticated_NoCookie(t *testing.T) {
	a := New("pw", "secret", true)
	req := httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
	if a.IsAuthenticated(req) {
		t.Error("IsAuthenticated returned true with no cookie")
	}
}

func TestIsAuthenticated_TamperedCookie(t *testing.T) {
	a := New("pw", "secret", true)
	rec := httptest.NewRecorder()
	a.SetSessionCookie(rec)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	// Tamper: flip the last char of the value.
	c := cookies[0]
	tampered := c.Value[:len(c.Value)-1]
	if tampered[len(tampered)-1] == 'a' {
		tampered += "b"
	} else {
		tampered += "a"
	}
	req := httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
	req.AddCookie(&http.Cookie{Name: c.Name, Value: tampered})
	if a.IsAuthenticated(req) {
		t.Error("IsAuthenticated returned true for tampered cookie")
	}
}

func TestClearSessionCookie(t *testing.T) {
	a := New("pw", "secret", true)
	rec := httptest.NewRecorder()
	a.ClearSessionCookie(rec)
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "wedding_admin" && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Error("ClearSessionCookie did not set an expired wedding_admin cookie")
	}
}

func TestMiddleware_AllowsAuthenticated(t *testing.T) {
	a := New("pw", "secret", true)
	called := false
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	cookieRec := httptest.NewRecorder()
	a.SetSessionCookie(cookieRec)
	req := httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
	for _, c := range cookieRec.Result().Cookies() {
		req.AddCookie(c)
	}
	h.ServeHTTP(rec, req)

	if !called {
		t.Error("handler not called when authenticated")
	}
}

func TestMiddleware_BlocksUnauthenticated(t *testing.T) {
	a := New("pw", "secret", true)
	called := false
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
	h.ServeHTTP(rec, req)

	if called {
		t.Error("handler called when unauthenticated")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestSessionCookie_SecureFlag(t *testing.T) {
	a := New("pw", "secret", true)
	rec := httptest.NewRecorder()
	a.SetSessionCookie(rec)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if !cookies[0].Secure {
		t.Errorf("SetSessionCookie Secure = false, want true")
	}

	a2 := New("pw", "secret", false)
	rec2 := httptest.NewRecorder()
	a2.SetSessionCookie(rec2)
	cookies2 := rec2.Result().Cookies()
	if len(cookies2) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies2))
	}
	if cookies2[0].Secure {
		t.Errorf("SetSessionCookie Secure = true, want false")
	}
}
