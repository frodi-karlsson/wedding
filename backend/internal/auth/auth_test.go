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

var testPasswords = Passwords{Admin: "correct-horse", Translator: "battery-staple"}

func newTestAuth(secure bool) *Authenticator {
	return New(testPasswords, "session-secret", secure)
}

// sessionRequest returns a request carrying the session cookie that a login as
// role would set.
func sessionRequest(t *testing.T, a *Authenticator, role Role) *http.Request {
	t.Helper()
	rec := httptest.NewRecorder()
	a.SetSessionCookie(rec, role)
	req := httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

func TestLogin_ReturnsRoleForEachPassword(t *testing.T) {
	a := newTestAuth(true)
	cases := map[string]Role{"correct-horse": RoleAdmin, "battery-staple": RoleTranslator}
	for pw, want := range cases {
		role, ok := a.Login(pw)

		if !ok || role != want {
			t.Errorf("Login(%q) = (%q, %v), want (%q, true)", pw, role, ok, want)
		}
	}
}

// TestLogin_RejectsWrongPasswords guards the length-leak fix too: both sides are
// SHA-256'd to a fixed 32 bytes before the constant-time compare, so a wrong
// password of any length must be rejected like any other.
func TestLogin_RejectsWrongPasswords(t *testing.T) {
	a := newTestAuth(true)
	for _, pw := range []string{"", "x", "wrong", "correct-hors", "correct-horsee", "battery-staplee", strings.Repeat("z", 200)} {
		if role, ok := a.Login(pw); ok {
			t.Errorf("Login(%q) = (%q, true), want rejection", pw, role)
		}
	}
}

func TestLogin_UnsetPasswordNeverMatches(t *testing.T) {
	a := New(Passwords{Admin: "correct-horse"}, "session-secret", true)

	role, ok := a.Login("")

	if ok {
		t.Errorf("Login(\"\") = (%q, true) with no translator password set, want rejection", role)
	}
}

func TestSessionRole_ReturnsRoleFromCookie(t *testing.T) {
	a := newTestAuth(true)
	for _, want := range []Role{RoleAdmin, RoleTranslator} {
		req := sessionRequest(t, a, want)

		role, ok := a.SessionRole(req)

		if !ok || role != want {
			t.Errorf("SessionRole() = (%q, %v), want (%q, true)", role, ok, want)
		}
	}
}

func TestSessionRole_NoCookie(t *testing.T) {
	a := newTestAuth(true)
	req := httptest.NewRequest(http.MethodGet, "/admin/invites", nil)

	if _, ok := a.SessionRole(req); ok {
		t.Error("SessionRole() ok with no cookie")
	}
}

func TestSessionRole_RejectsTamperedCookies(t *testing.T) {
	a := newTestAuth(true)
	valid := sessionRequest(t, a, RoleTranslator)
	c, err := valid.Cookie(cookieName)
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	// Flip the last signature char to a guaranteed-different one.
	repl := byte('a')
	if c.Value[len(c.Value)-1] == 'a' {
		repl = 'b'
	}
	cases := map[string]string{
		"bad signature": c.Value[:len(c.Value)-1] + string(repl),
		"role swapped":  "admin" + strings.TrimPrefix(c.Value, "translator"),
		"legacy format": strings.TrimPrefix(c.Value, "translator."),
		"unknown role":  "owner" + strings.TrimPrefix(c.Value, "translator"),
	}
	for name, value := range cases {
		req := httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
		req.AddCookie(&http.Cookie{Name: cookieName, Value: value})

		if role, ok := a.SessionRole(req); ok {
			t.Errorf("%s: SessionRole() = (%q, true), want rejection", name, role)
		}
	}
}

func TestClearSessionCookie(t *testing.T) {
	a := newTestAuth(true)
	rec := httptest.NewRecorder()

	a.ClearSessionCookie(rec)

	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Errorf("ClearSessionCookie did not set an expired %s cookie", cookieName)
	}
}

func TestRequire_AllowsListedRole(t *testing.T) {
	a := newTestAuth(true)
	called := false
	h := a.Require(RoleAdmin, RoleTranslator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, sessionRequest(t, a, RoleTranslator))

	if !called {
		t.Errorf("handler not called for an allowed role, status %d", rec.Code)
	}
}

func TestRequire_ForbidsOtherRole(t *testing.T) {
	a := newTestAuth(true)
	called := false
	h := a.Require(RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, sessionRequest(t, a, RoleTranslator))

	if called {
		t.Error("handler called for a role that is not allowed")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestRequire_BlocksUnauthenticated(t *testing.T) {
	a := newTestAuth(true)
	called := false
	h := a.Require(RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/invites", nil))

	if called {
		t.Error("handler called when unauthenticated")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestSessionCookie_SecureFlag(t *testing.T) {
	for _, secure := range []bool{true, false} {
		a := newTestAuth(secure)
		rec := httptest.NewRecorder()

		a.SetSessionCookie(rec, RoleAdmin)

		cookies := rec.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("expected 1 cookie, got %d", len(cookies))
		}
		if cookies[0].Secure != secure {
			t.Errorf("Secure = %v, want %v", cookies[0].Secure, secure)
		}
	}
}
