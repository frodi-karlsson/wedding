package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
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
	cookieName    = "wedding_admin"
	sessionMaxAge = 7 * 24 * time.Hour
)

// Authenticator handles admin login and session-cookie verification.
type Authenticator struct {
	password      string
	sessionSecret []byte
	secure        bool
	limiter       *loginLimiter
}

func New(password, sessionSecret string, secure bool) *Authenticator {
	return &Authenticator{
		password:      password,
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

// Login returns true if the given password matches the configured admin password.
// Both sides are SHA-256'd to a fixed 32 bytes before the constant-time compare
// so the comparison neither short-circuits nor leaks the password length via
// timing (hmac.Equal on raw bytes of differing length returns early).
func (a *Authenticator) Login(password string) bool {
	got := sha256.Sum256([]byte(password))
	want := sha256.Sum256([]byte(a.password))
	return hmac.Equal(got[:], want[:])
}

// SetSessionCookie writes a signed session cookie to the response.
func (a *Authenticator) SetSessionCookie(w http.ResponseWriter) {
	ts := time.Now().Unix()
	value := a.sign(ts)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
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

// IsAuthenticated returns true if the request carries a valid signed cookie.
func (a *Authenticator) IsAuthenticated(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	return a.verify(c.Value)
}

// Middleware protects a handler with session-cookie auth.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.IsAuthenticated(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Authenticator) sign(ts int64) string {
	m := hmac.New(sha256.New, a.sessionSecret)
	m.Write([]byte(strconv.FormatInt(ts, 10)))
	sig := hex.EncodeToString(m.Sum(nil))
	return strconv.FormatInt(ts, 10) + "." + sig
}

func (a *Authenticator) verify(value string) bool {
	tsStr, sig, ok := strings.Cut(value, ".")
	if !ok {
		return false
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix()-ts > int64(sessionMaxAge.Seconds()) {
		return false
	}
	m := hmac.New(sha256.New, a.sessionSecret)
	m.Write([]byte(tsStr))
	expected := hex.EncodeToString(m.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}
