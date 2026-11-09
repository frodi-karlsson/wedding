package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxLimiterEntries = 1000
	maxFailures       = 5 // 5 failures allowed; 6th is blocked
	blockDuration     = 15 * time.Minute
)

// loginLimiter caps per-IP failed login attempts. It trusts the caller to pass
// the real client IP (the server extracts that from X-Forwarded-For after Caddy
// has overwritten it with the true peer address).
type loginLimiter struct {
	mu        sync.Mutex
	failures  map[string]int
	blockTill map[string]time.Time
	lastSeen  map[string]time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{
		failures:  make(map[string]int),
		blockTill: make(map[string]time.Time),
		lastSeen:  make(map[string]time.Time),
	}
}

func (l *loginLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lastSeen[ip] = time.Now()
	l.evictIfNeeded()
	if block, ok := l.blockTill[ip]; ok {
		if time.Now().Before(block) {
			return false
		}
		// Block expired, so give the IP a fresh 5-attempt budget.
		delete(l.failures, ip)
		delete(l.blockTill, ip)
	}
	return true
}

func (l *loginLimiter) RecordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lastSeen[ip] = time.Now()
	l.evictIfNeeded()
	l.failures[ip]++
	if l.failures[ip] > maxFailures {
		l.blockTill[ip] = time.Now().Add(blockDuration)
	}
}

func (l *loginLimiter) Reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, ip)
	delete(l.blockTill, ip)
	delete(l.lastSeen, ip)
}

func (l *loginLimiter) evictIfNeeded() {
	if len(l.lastSeen) <= maxLimiterEntries {
		return
	}
	var oldestIP string
	var oldestTime time.Time
	for ip, t := range l.lastSeen {
		if oldestIP == "" || t.Before(oldestTime) {
			oldestIP = ip
			oldestTime = t
		}
	}
	delete(l.failures, oldestIP)
	delete(l.blockTill, oldestIP)
	delete(l.lastSeen, oldestIP)
}

const (
	cookieName    = "wedding_session"
	sessionMaxAge = 7 * 24 * time.Hour
)

// Role is what a session may do. An admin manages invites, a translator works
// on the speech translations.
type Role string

const (
	RoleAdmin      Role = "admin"
	RoleTranslator Role = "translator"
)

func (r Role) known() bool {
	return r == RoleAdmin || r == RoleTranslator
}

// Passwords holds one shared password per role. An empty password disables
// that role's login.
type Passwords struct {
	Admin      string
	Translator string
}

// Authenticator handles login and role-carrying session cookies.
type Authenticator struct {
	passwords     Passwords
	sessionSecret []byte
	secure        bool
	limiter       *loginLimiter
}

func New(passwords Passwords, sessionSecret string, secure bool) *Authenticator {
	return &Authenticator{
		passwords:     passwords,
		sessionSecret: []byte(sessionSecret),
		secure:        secure,
		limiter:       newLoginLimiter(),
	}
}

// Allow reports whether the given IP is currently allowed to attempt login.
func (a *Authenticator) Allow(ip string) bool {
	return a.limiter.Allow(ip)
}

// RecordFailure records a failed login attempt from the given IP.
func (a *Authenticator) RecordFailure(ip string) {
	a.limiter.RecordFailure(ip)
}

// ResetLogin clears the login failure state for the given IP (e.g. on success).
func (a *Authenticator) ResetLogin(ip string) {
	a.limiter.Reset(ip)
}

// Login returns the role whose password matches. Every role's password is
// compared on every attempt, and both sides are SHA-256'd to a fixed 32 bytes
// before the constant-time compare, so timing reveals neither which role was
// close nor the password length.
func (a *Authenticator) Login(password string) (Role, bool) {
	got := sha256.Sum256([]byte(password))
	candidates := []struct {
		role     Role
		password string
	}{
		{RoleAdmin, a.passwords.Admin},
		{RoleTranslator, a.passwords.Translator},
	}
	var matched Role
	for _, c := range candidates {
		want := sha256.Sum256([]byte(c.password))
		if hmac.Equal(got[:], want[:]) && c.password != "" && matched == "" {
			matched = c.role
		}
	}
	return matched, matched != ""
}

// SetSessionCookie writes a signed session cookie for role to the response.
func (a *Authenticator) SetSessionCookie(w http.ResponseWriter, role Role) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    a.sign(role, time.Now().Unix()),
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionMaxAge.Seconds()),
		Expires:  time.Now().Add(sessionMaxAge),
	})
}

// ClearSessionCookie expires the session cookie immediately.
func (a *Authenticator) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

// SessionRole returns the role of the request's valid session cookie.
func (a *Authenticator) SessionRole(r *http.Request) (Role, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return "", false
	}
	return a.verify(c.Value)
}

// Require lets a request through only with a session for one of roles: 401
// without a session, 403 with a session for some other role.
func (a *Authenticator) Require(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := a.SessionRole(r)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !slices.Contains(roles, role) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// sign renders a cookie value "<role>.<unix ts>.<hex hmac of role.ts>".
func (a *Authenticator) sign(role Role, ts int64) string {
	payload := string(role) + "." + strconv.FormatInt(ts, 10)
	return payload + "." + a.mac(payload)
}

func (a *Authenticator) mac(payload string) string {
	m := hmac.New(sha256.New, a.sessionSecret)
	m.Write([]byte(payload))
	return hex.EncodeToString(m.Sum(nil))
}

func (a *Authenticator) verify(value string) (Role, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return "", false
	}
	roleStr, tsStr, sig := parts[0], parts[1], parts[2]
	if !hmac.Equal([]byte(sig), []byte(a.mac(roleStr+"."+tsStr))) {
		return "", false
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil || time.Now().Unix()-ts > int64(sessionMaxAge.Seconds()) {
		return "", false
	}
	role := Role(roleStr)
	if !role.known() {
		return "", false
	}
	return role, true
}
