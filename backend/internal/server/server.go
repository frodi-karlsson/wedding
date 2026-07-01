package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"wedding/backend/internal/auth"
	"wedding/backend/internal/db"
	"wedding/backend/internal/invite"
)

// New returns an http.Handler with all routes wired and CORS + admin auth applied.
func New(svc *invite.Service, a *auth.Authenticator, allowedOrigins []string) http.Handler {
	mux := http.NewServeMux()

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

	return CORS(allowedOrigins)(mux)
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func idFromPath(r *http.Request) (int64, bool) {
	s := r.PathValue("id")
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
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
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		resp := InviteWithGuestsResponse{
			Invite: toInviteResponse(inv),
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
		inv, saved, err := svc.SubmitRSVP(r.Context(), id, guests)
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
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		resp := InviteWithGuestsResponse{
			Invite: toInviteResponse(inv),
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
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if !a.Login(req.Password) {
			writeError(w, http.StatusUnauthorized, "invalid password")
			return
		}
		a.SetSessionCookie(w)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func handleLogout(a *auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.ClearSessionCookie(w)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		resp := ListInvitesResponse{Invites: make([]InviteResponse, len(invites))}
		for i, inv := range invites {
			resp.Invites[i] = toInviteResponse(inv)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleCreateInvite(svc *invite.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateInviteRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if msg := validateInviteRequest(inviteRequest{Name: req.Name, GuestNames: req.GuestNames, MinPlus: req.MinPlus, MaxPlus: req.MaxPlus}); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		inv, err := svc.CreateInvite(r.Context(), req.Name, req.MinPlus, req.MaxPlus, req.GuestNames)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		_, guests, err := svc.GetInvite(r.Context(), inv.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusCreated, InviteWithGuestsResponse{
			Invite: toInviteResponse(inv),
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
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, InviteWithGuestsResponse{
			Invite: toInviteResponse(inv),
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
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		_, guests, err := svc.GetInvite(r.Context(), inv.ID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeError(w, http.StatusNotFound, "invite not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, InviteWithGuestsResponse{
			Invite: toInviteResponse(inv),
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
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func toGuestResponses(guests []db.Guest) []GuestResponse {
	out := make([]GuestResponse, len(guests))
	for i, g := range guests {
		out[i] = toGuestResponse(g)
	}
	return out
}
