package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	cookieName    = "wedding_admin"
	sessionMaxAge = 7 * 24 * time.Hour
)

// Authenticator handles admin login and session-cookie verification.
type Authenticator struct {
	password      string
	sessionSecret []byte
	secure        bool
}

func New(password, sessionSecret string, secure bool) *Authenticator {
	return &Authenticator{
		password:      password,
		sessionSecret: []byte(sessionSecret),
		secure:        secure,
	}
}

// Login returns true if the given password matches the configured admin password.
func (a *Authenticator) Login(password string) bool {
	return hmac.Equal([]byte(password), []byte(a.password))
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
