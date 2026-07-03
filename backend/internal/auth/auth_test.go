package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoginLimiter_Allows5Failures_Blocks6th(t *testing.T) {
	l := newLoginLimiter()
	// 5 failures: still allowed
	for i := 0; i < 5; i++ {
		l.RecordFailure("1.2.3.4")
		if !l.Allow("1.2.3.4") {
			t.Fatalf("after %d failures, IP should still be allowed", i+1)
		}
	}
	// 6th failure: now blocked
	l.RecordFailure("1.2.3.4")
	if l.Allow("1.2.3.4") {
		t.Error("after 6 failures, IP should be blocked")
	}
}

func TestLoginLimiter_AllowsOtherIPs(t *testing.T) {
	l := newLoginLimiter()
	for i := 0; i < 6; i++ {
		l.RecordFailure("1.2.3.4")
	}
	if !l.Allow("5.6.7.8") {
		t.Error("different IP should not be blocked")
	}
}

func TestLoginLimiter_ResetsOnSuccess(t *testing.T) {
	l := newLoginLimiter()
	for i := 0; i < 4; i++ {
		l.RecordFailure("1.2.3.4")
	}
	l.Reset("1.2.3.4")
	for i := 0; i < 5; i++ {
		l.RecordFailure("1.2.3.4")
	}
	if !l.Allow("1.2.3.4") {
		t.Error("after reset + 5 failures, IP should still be allowed (6th blocks)")
	}
}

func TestLoginLimiter_FreshBudgetAfterBlockExpires(t *testing.T) {
	l := newLoginLimiter()
	// Exhaust the 5-attempt budget and trigger a block.
	for i := 0; i < 6; i++ {
		l.RecordFailure("1.2.3.4")
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("IP should be blocked after 6 failures")
	}
	// Simulate block expiry by manually advancing the blockTill timestamp.
	l.mu.Lock()
	l.blockTill["1.2.3.4"] = time.Now().Add(-time.Second)
	l.mu.Unlock()
	// Allow should return true and reset the failure count.
	if !l.Allow("1.2.3.4") {
		t.Fatal("after block expires, IP should be allowed again")
	}
	// Should have a fresh 5-attempt budget, not immediately re-block.
	for i := 0; i < 5; i++ {
		l.RecordFailure("1.2.3.4")
		if !l.Allow("1.2.3.4") {
			t.Fatalf("after expiry + %d failures, IP should still be allowed (fresh budget)", i+1)
		}
	}
	// 6th failure after expiry: blocked again.
	l.RecordFailure("1.2.3.4")
	if l.Allow("1.2.3.4") {
		t.Error("after expiry + 6 failures, IP should be blocked")
	}
}

func TestLoginLimiter_CapsEntries(t *testing.T) {
	l := newLoginLimiter()
	for i := 0; i < maxLimiterEntries+100; i++ {
		l.Allow(fmt.Sprintf("10.0.0.%d", i))
	}
	l.mu.Lock()
	count := len(l.lastSeen)
	l.mu.Unlock()
	if count > maxLimiterEntries {
		t.Errorf("map size = %d, want <= %d", count, maxLimiterEntries)
	}
}

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

// TestLogin_WrongPasswordDifferentLengths guards the length-leak fix: because
// both sides are SHA-256'd to a fixed 32 bytes before the constant-time compare,
// a wrong password of a different length must be rejected just like any other.
func TestLogin_WrongPasswordDifferentLengths(t *testing.T) {
	a := New("correct-horse", "session-secret", true)
	for _, pw := range []string{"", "x", "correct-hors", "correct-horsee", strings.Repeat("z", 200)} {
		if a.Login(pw) {
			t.Errorf("Login(%q) returned true, want false", pw)
		}
	}
	if !a.Login("correct-horse") {
		t.Error("Login with correct password returned false")
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
	// Tamper: replace the last char with a guaranteed-different one. Compare the
	// original last char (not the one before it) so the result always differs.
	// Otherwise it can reconstruct the valid value and still verify (flaky).
	c := cookies[0]
	repl := byte('a')
	if c.Value[len(c.Value)-1] == 'a' {
		repl = 'b'
	}
	tampered := c.Value[:len(c.Value)-1] + string(repl)
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
