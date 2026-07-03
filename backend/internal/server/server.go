package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"wedding/backend/internal/auth"
	"wedding/backend/internal/db"
	"wedding/backend/internal/invite"
)

// healthCheckTimeout bounds the DB ping done by the /healthz handler.
const healthCheckTimeout = 2 * time.Second

// New returns an http.Handler with all routes wired, panic recovery (outermost),
// CORS, body-size limit, and admin auth applied. d is used only by the
// unauthenticated /healthz probe to ping the database.
func New(svc *invite.Service, a *auth.Authenticator, d *sql.DB, allowedOrigins []string) http.Handler {
	mux := http.NewServeMux()

	// Health check — unauthenticated, not rate limited. Used by the container
	// HEALTHCHECK. Kept deliberately trivial: pings the DB and returns JSON.
	mux.HandleFunc("GET /healthz", handleHealthz(d))

	// Public routes
	mux.HandleFunc("GET /invites/{id}", handleGetInvite(svc))
	mux.HandleFunc("POST /invites/{id}/rsvp", handleRSVP(svc))

	// Admin auth routes (login/logout are NOT behind auth middleware)
	mux.HandleFunc("POST /admin/login", handleLogin(a))
	mux.HandleFunc("POST /admin/logout", handleLogout(a))

	// Admin protected routes
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("GET /admin/invites", handleListInvites(svc))
	adminMux.HandleFunc("POST /admin/invites", handleCreateInvite(svc))
	adminMux.HandleFunc("GET /admin/invites/{id}", handleAdminGetInvite(svc))
	adminMux.HandleFunc("PUT /admin/invites/{id}", handleUpdateInvite(svc))
	adminMux.HandleFunc("DELETE /admin/invites/{id}", handleDeleteInvite(svc))
	mux.Handle("/admin/invites", a.Middleware(adminMux))
	mux.Handle("/admin/invites/", a.Middleware(adminMux))

	return recoveryMiddleware(CORS(allowedOrigins)(bodyLimitMiddleware(64 * 1024)(mux)))
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v", rec)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func bodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client's real IP from X-Forwarded-For.
// SAFETY: This is only trustworthy because Caddy's Caddyfile uses
// `header_up X-Forwarded-For {http.request.remote.host}` to OVERWRITE
// (not append to) the XFF header with the real peer IP. Without that
// Caddy config, the left-most XFF entry is attacker-controlled.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(xff)
	}
	// r.RemoteAddr is "host:port"; strip the ephemeral port so the rate limiter
	// keys on the host alone (otherwise every new connection looks like a new IP).
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// handleHealthz reports service liveness by pinging the database. It is
// unauthenticated and not rate limited so container health checks can reach it.
func handleHealthz(d *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()
		if err := d.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, StatusResponse{Status: "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, StatusResponse{Status: "ok"})
	}
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

// logInternalError logs the error cause and writes a generic 500 response.
// Use this instead of writeError for internal errors so the cause is diagnosable.
func logInternalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return maxErr
		}
		return err
	}
	return nil
}

func idFromPath(r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if id == "" {
		return "", false
	}
	return id, true
}

// --- public handlers ---

func handleGetInvite(svc *invite.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idFromPath(r)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		inv, guests, err := svc.GetInvite(r.Context(), id)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeError(w, http.StatusNotFound, "invite not found")
				return
			}
			logInternalError(w, err)
			return
		}
		resp := InviteWithGuestsResponse{
			Invite: toInviteResponse(&inv),
			Guests: toGuestResponses(guests),
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleRSVP(svc *invite.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idFromPath(r)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var req RSVPRequest
		if err := decodeJSON(r, &req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		guests := make([]db.Guest, len(req.Guests))
		for i, g := range req.Guests {
			guests[i] = db.Guest{
				Name:              g.Name,
				DietaryPreference: g.DietaryPreference,
				AlcoholFree:       g.AlcoholFree,
				IsPrimary:         g.IsPrimary,
			}
		}
		if utf8.RuneCountInString(req.Message) > 1000 {
			writeError(w, http.StatusBadRequest, "message must be at most 1000 characters")
			return
		}
		inv, saved, err := svc.SubmitRSVP(r.Context(), id, guests, req.Message)
		if err != nil {
			var ve *invite.ValidationError
			if errors.As(err, &ve) {
				writeError(w, http.StatusBadRequest, ve.Error())
				return
			}
			if errors.Is(err, db.ErrNotFound) {
				writeError(w, http.StatusNotFound, "invite not found")
				return
			}
			logInternalError(w, err)
			return
		}
		resp := InviteWithGuestsResponse{
			Invite: toInviteResponse(&inv),
			Guests: toGuestResponses(saved),
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// --- admin auth handlers ---

func handleLogin(a *auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := decodeJSON(r, &req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		ip := clientIP(r)
		if !a.Allow(ip) {
			writeError(w, http.StatusTooManyRequests, "too many failed attempts, try again later")
			return
		}
		if !a.Login(req.Password) {
			a.RecordFailure(ip)
			if !a.Allow(ip) {
				writeError(w, http.StatusTooManyRequests, "too many failed attempts, try again later")
				return
			}
			writeError(w, http.StatusUnauthorized, "invalid password")
			return
		}
		a.ResetLogin(ip)
		a.SetSessionCookie(w)
		writeJSON(w, http.StatusOK, StatusResponse{Status: "ok"})
	}
}

func handleLogout(a *auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.ClearSessionCookie(w)
		writeJSON(w, http.StatusOK, StatusResponse{Status: "ok"})
	}
}

// --- admin invite handlers ---

// inviteRequest holds the common fields validated for Create/Update invite requests.
type inviteRequest struct {
	Name       string
	GuestNames []string
	MinPlus    int
	MaxPlus    int
}

// validateInviteRequest returns a non-empty error message string on failure
// and "" on success.
func validateInviteRequest(req inviteRequest) string {
	if strings.TrimSpace(req.Name) == "" {
		return "name is required"
	}
	if len(req.GuestNames) == 0 {
		return "at least one guest name is required"
	}
	if req.GuestNames[0] != req.Name {
		return "first guest name must match the invite name"
	}
	if req.MinPlus < 0 || req.MaxPlus < 0 || req.MinPlus > req.MaxPlus {
		return "min_plus and max_plus must be non-negative with min <= max"
	}
	return ""
}

func handleListInvites(svc *invite.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invites, err := svc.ListInvites(r.Context())
		if err != nil {
			logInternalError(w, err)
			return
		}
		resp := ListInvitesResponse{Invites: make([]InviteResponse, len(invites))}
		for i, inv := range invites {
			resp.Invites[i] = toInviteResponse(&inv)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleCreateInvite(svc *invite.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateInviteRequest
		if err := decodeJSON(r, &req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if msg := validateInviteRequest(inviteRequest{Name: req.Name, GuestNames: req.GuestNames, MinPlus: req.MinPlus, MaxPlus: req.MaxPlus}); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		inv, err := svc.CreateInvite(r.Context(), req.Name, req.MinPlus, req.MaxPlus, req.GuestNames)
		if err != nil {
			logInternalError(w, err)
			return
		}
		_, guests, err := svc.GetInvite(r.Context(), inv.ID)
		if err != nil {
			logInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, InviteWithGuestsResponse{
			Invite: toInviteResponse(&inv),
			Guests: toGuestResponses(guests),
		})
	}
}

func handleAdminGetInvite(svc *invite.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idFromPath(r)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		inv, guests, err := svc.GetInvite(r.Context(), id)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeError(w, http.StatusNotFound, "invite not found")
				return
			}
			logInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, InviteWithGuestsResponse{
			Invite: toInviteResponse(&inv),
			Guests: toGuestResponses(guests),
		})
	}
}

func handleUpdateInvite(svc *invite.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idFromPath(r)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var req UpdateInviteRequest
		if err := decodeJSON(r, &req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if msg := validateInviteRequest(inviteRequest{Name: req.Name, GuestNames: req.GuestNames, MinPlus: req.MinPlus, MaxPlus: req.MaxPlus}); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		inv, err := svc.UpdateInvite(r.Context(), id, req.Name, req.MinPlus, req.MaxPlus, req.GuestNames)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeError(w, http.StatusNotFound, "invite not found")
				return
			}
			logInternalError(w, err)
			return
		}
		_, guests, err := svc.GetInvite(r.Context(), inv.ID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeError(w, http.StatusNotFound, "invite not found")
				return
			}
			logInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, InviteWithGuestsResponse{
			Invite: toInviteResponse(&inv),
			Guests: toGuestResponses(guests),
		})
	}
}

func handleDeleteInvite(svc *invite.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idFromPath(r)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if err := svc.DeleteInvite(r.Context(), id); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeError(w, http.StatusNotFound, "invite not found")
				return
			}
			logInternalError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func toGuestResponses(guests []db.Guest) []GuestResponse {
	out := make([]GuestResponse, len(guests))
	for i := range guests {
		out[i] = toGuestResponse(&guests[i])
	}
	return out
}
