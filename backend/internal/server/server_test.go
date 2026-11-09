package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wedding/backend/internal/auth"
	"wedding/backend/internal/db"
	"wedding/backend/internal/email"
	"wedding/backend/internal/invite"
)

func newTestServer(t *testing.T) (http.Handler, *email.Fake) {
	t.Helper()
	srv, fakeEmail, _ := newTestServerWithDB(t)
	return srv, fakeEmail
}

func newTestServerWithDB(t *testing.T) (http.Handler, *email.Fake, *sql.DB) {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if err := db.Migrate(d); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	store := db.NewSQLiteStore(d)
	fakeEmail := &email.Fake{}
	svc := invite.NewService(store, fakeEmail)
	a := auth.New(testPasswords, "session-secret", false)
	return New(svc, a, d, []string{"https://carlaochfrodi.wedding"}), fakeEmail, d
}

func TestGetInvite_ReturnsInviteAndGuests(t *testing.T) {
	srv, _ := newTestServer(t)

	// Create an invite via admin first (login then create).
	createAndLogin(t, srv)

	// We need the invite id; list invites.
	rec := jsonRequest(t, srv, http.MethodGet, "/admin/invites", nil, true)
	var listResp ListInvitesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(listResp.Invites) == 0 {
		t.Fatal("no invites")
	}
	id := listResp.Invites[0].ID

	// Public GET /invites/{id}
	rec = jsonRequest(t, srv, http.MethodGet, "/invites/"+fmt.Sprint(id), nil, false)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	var resp InviteWithGuestsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Invite.Name != "Frodi & Carla" {
		t.Errorf("Name = %q", resp.Invite.Name)
	}
	if len(resp.Guests) != 1 {
		t.Errorf("len(Guests) = %d, want 1", len(resp.Guests))
	}
}

func TestGetInvite_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := jsonRequest(t, srv, http.MethodGet, "/invites/ZZZZZ", nil, false)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// A malformed id is answered exactly like an unknown one. Keeping the two
// indistinguishable denies an enumerator a cheap way to tell "wrong shape" from
// "no such invite".
func TestGetInvite_MalformedIDLooksLikeAnUnknownInvite(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"too short", "1"},
		{"too long", "ABCDEFGH"},
		{"outside the alphabet", "AB-DE"},
		{"legacy length but not hex", strings.Repeat("z", 32)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := newTestServer(t)

			rec := jsonRequest(t, srv, http.MethodGet, "/invites/"+tt.id, nil, false)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestGetInvite_ResolvesAnIDTypedInLowercase(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := createAndLogin(t, srv)
	id := firstInviteID(t, srv, cookies)

	rec := jsonRequest(t, srv, http.MethodGet, "/invites/"+strings.ToLower(id), nil, false)

	if rec.Code != http.StatusOK {
		t.Errorf("GET with lowercased id %q: status = %d, want 200; body: %s", strings.ToLower(id), rec.Code, rec.Body.String())
	}
}

func TestSubmitRSVP_Valid_SendsEmailAndReturnsSaved(t *testing.T) {
	srv, fakeEmail := newTestServer(t)
	cookies := createAndLogin(t, srv)
	id := firstInviteID(t, srv, cookies)

	body := RSVPRequest{Guests: []GuestInput{
		{Name: "Frodi & Carla", IsPrimary: true},
		{Name: "Plus1", DietaryPreference: "vegan", AlcoholFree: true},
	}}
	rec := jsonRequest(t, srv, http.MethodPost, "/invites/"+fmt.Sprint(id)+"/rsvp", body, false)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if len(fakeEmail.Calls) != 1 {
		t.Errorf("email calls = %d, want 1", len(fakeEmail.Calls))
	}
}

func TestSubmitRSVP_ValidationFails(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := createAndLogin(t, srv)
	id := firstInviteID(t, srv, cookies)

	body := RSVPRequest{Guests: []GuestInput{{Name: "NoPrimary"}}}
	rec := jsonRequest(t, srv, http.MethodPost, "/invites/"+fmt.Sprint(id)+"/rsvp", body, false)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestPanicRecovery_Returns500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invites/test", nil)
	panickingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	recoveryMiddleware(panickingHandler).ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 after panic", rec.Code)
	}
}

var testPasswords = auth.Passwords{Admin: "admin-pw", Translator: "translator-pw"}

func TestLogin_ReturnsRoleAndSetsSessionCookie(t *testing.T) {
	cases := map[string]string{"admin-pw": "admin", "translator-pw": "translator"}
	for password, wantRole := range cases {
		srv, _ := newTestServer(t)

		rec := jsonRequest(t, srv, http.MethodPost, "/auth/login", LoginRequest{Password: password}, false)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", wantRole, rec.Code)
		}
		var resp SessionResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Role != wantRole {
			t.Errorf("role = %q, want %q", resp.Role, wantRole)
		}
		cookies := rec.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != "wedding_session" {
			t.Errorf("%s: expected wedding_session cookie, got %+v", wantRole, cookies)
		}
	}
}

func loginAs(t *testing.T, srv http.Handler, password string) []*http.Cookie {
	t.Helper()
	rec := jsonRequestWithCookies(t, srv, http.MethodPost, "/auth/login", LoginRequest{Password: password}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d", rec.Code)
	}
	return rec.Result().Cookies()
}

func TestSession_ReturnsRoleOfCurrentSession(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAs(t, srv, "translator-pw")

	rec := jsonRequestWithCookies(t, srv, http.MethodGet, "/auth/session", nil, cookies)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp SessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Role != "translator" {
		t.Errorf("role = %q, want translator", resp.Role)
	}
}

func TestSession_UnauthorizedWithoutSession(t *testing.T) {
	srv, _ := newTestServer(t)

	rec := jsonRequest(t, srv, http.MethodGet, "/auth/session", nil, false)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestLogout_EndsSession(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAs(t, srv, "admin-pw")

	rec := jsonRequestWithCookies(t, srv, http.MethodPost, "/auth/logout", nil, cookies)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	cleared := rec.Result().Cookies()
	if len(cleared) != 1 || cleared[0].Name != "wedding_session" || cleared[0].MaxAge >= 0 {
		t.Errorf("expected an expired wedding_session cookie, got %+v", cleared)
	}
}

func TestAdminRoutes_ForbiddenForTranslator(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAs(t, srv, "translator-pw")

	rec := jsonRequestWithCookies(t, srv, http.MethodGet, "/admin/invites", nil, cookies)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := jsonRequest(t, srv, http.MethodPost, "/auth/login", LoginRequest{Password: "wrong"}, false)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestLogin_BodyLimit_Returns413(t *testing.T) {
	srv, _ := newTestServer(t)
	body := `{"password":"` + strings.Repeat("a", 64*1024) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
}

func TestLogin_RateLimitedAfter5Failures(t *testing.T) {
	srv, _ := newTestServer(t)
	// Set X-Forwarded-For so clientIP extracts a stable IP (simulating Caddy).
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rec.Code)
		}
	}
	// 6th attempt: blocked.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("6th attempt: status = %d, want 429", rec.Code)
	}
}

func TestAdminRoutes_RequireAuth(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := jsonRequest(t, srv, http.MethodGet, "/admin/invites", nil, false)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAdminCreateInvite(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/invites",
		CreateInviteRequest{Name: "New Couple", MinPlus: 0, MaxPlus: 1, Guests: presets("New Couple")}, true)
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 200 or 201; body: %s", rec.Code, rec.Body.String())
	}
	var resp InviteWithGuestsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Invite.Name != "New Couple" {
		t.Errorf("Name = %q", resp.Invite.Name)
	}
	if len(resp.Guests) != 1 || !resp.Guests[0].IsPrimary {
		t.Errorf("expected 1 primary guest, got %+v", resp.Guests)
	}
}

func TestAdminDeleteInvite(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := createAndLogin(t, srv)
	id := firstInviteID(t, srv, cookies)

	rec := jsonRequestWithCookies(t, srv, http.MethodDelete, "/admin/invites/"+fmt.Sprint(id), nil, cookies)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 200 or 204", rec.Code)
	}

	// Confirm it's gone.
	rec = jsonRequest(t, srv, http.MethodGet, "/invites/"+fmt.Sprint(id), nil, false)
	if rec.Code != http.StatusNotFound {
		t.Errorf("after delete, GET status = %d, want 404", rec.Code)
	}
}

func TestAdminDeleteInvite_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := jsonRequest(t, srv, http.MethodDelete, "/admin/invites/ZZZZZ", nil, true)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminCreateInvite_WithPresetGuests(t *testing.T) {
	srv, _ := newTestServer(t)
	body := CreateInviteRequest{Name: "Frodi & Carla", MinPlus: 0, MaxPlus: 2, Guests: presets("Frodi & Carla", "Friend")}
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/invites", body, true)
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	var resp InviteWithGuestsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Guests) != 2 {
		t.Errorf("len(Guests) = %d, want 2", len(resp.Guests))
	}
	if !resp.Guests[0].IsPrimary || resp.Guests[0].Name != "Frodi & Carla" {
		t.Errorf("Guests[0] = %+v", resp.Guests[0])
	}
	if resp.Guests[1].Name != "Friend" {
		t.Errorf("Guests[1].Name = %q", resp.Guests[1].Name)
	}
}

func TestAdminCreateInvite_GuestNamesMissing(t *testing.T) {
	srv, _ := newTestServer(t)
	body := CreateInviteRequest{Name: "X", MinPlus: 0, MaxPlus: 1, Guests: nil}
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/invites", body, true)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestAdminCreateInvite_FirstNameMismatch(t *testing.T) {
	srv, _ := newTestServer(t)
	body := CreateInviteRequest{Name: "X", MinPlus: 0, MaxPlus: 1, Guests: presets("Y")}
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/invites", body, true)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestAdminCreateInvite_InvalidMinMax(t *testing.T) {
	srv, _ := newTestServer(t)
	body := CreateInviteRequest{Name: "X", MinPlus: 3, MaxPlus: 1, Guests: presets("X")}
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/invites", body, true)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestAdminUpdateInvite_GuestFetchNotFound(t *testing.T) {
	svc := invite.NewService(&updateFetchNotFoundStore{}, &email.Fake{})
	a := auth.New(testPasswords, "session-secret", false)
	srv := New(svc, a, nil, []string{"https://example.com"})
	rec := jsonRequest(t, srv, http.MethodPut, "/admin/invites/ABCDE",
		UpdateInviteRequest{Name: "X", MinPlus: 0, MaxPlus: 1, Guests: presets("X")}, true)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestClientIP_StripsPortFromRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = net.JoinHostPort("203.0.113.7", "54321")
	if got := clientIP(req); got != "203.0.113.7" {
		t.Errorf("clientIP() = %q, want %q (port must be stripped)", got, "203.0.113.7")
	}
}

func TestClientIP_PrefersXForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	req.Header.Set("X-Forwarded-For", "198.51.100.5, 10.0.0.1")
	if got := clientIP(req); got != "198.51.100.5" {
		t.Errorf("clientIP() = %q, want %q", got, "198.51.100.5")
	}
}

func TestClientIP_FallsBackToRawWhenNoPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "unixsocket" // no host:port form
	if got := clientIP(req); got != "unixsocket" {
		t.Errorf("clientIP() = %q, want raw fallback %q", got, "unixsocket")
	}
}

func TestHealthz_OKWhenDBReachable(t *testing.T) {
	srv, _, _ := newTestServerWithDB(t)
	rec := jsonRequestWithCookies(t, srv, http.MethodGet, "/healthz", nil, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var resp StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status field = %q, want %q", resp.Status, "ok")
	}
}

func TestHealthz_503WhenDBClosed(t *testing.T) {
	srv, _, d := newTestServerWithDB(t)
	if err := d.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	rec := jsonRequestWithCookies(t, srv, http.MethodGet, "/healthz", nil, nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 when DB is unreachable", rec.Code)
	}
}

func TestHealthz_NotBehindAuth(t *testing.T) {
	// No admin cookie is sent; /healthz must still be reachable (200), unlike
	// admin routes which return 401.
	srv, _, _ := newTestServerWithDB(t)
	rec := jsonRequestWithCookies(t, srv, http.MethodGet, "/healthz", nil, nil)
	if rec.Code == http.StatusUnauthorized {
		t.Error("/healthz must not be behind auth")
	}
}

// --- helpers ---
//
// Admin cookies are threaded explicitly through the helpers below (loginAndGetCookies
// returns them; callers pass them into jsonRequestWithCookies). There is no shared
// mutable global for auth state.

func loginAndGetCookies(t *testing.T, srv http.Handler) []*http.Cookie {
	t.Helper()
	return loginAs(t, srv, "admin-pw")
}

// createAndLogin logs in as admin, seeds one invite, and returns the admin cookies
// so the caller can make further authenticated requests.
func createAndLogin(t *testing.T, srv http.Handler) []*http.Cookie {
	t.Helper()
	cookies := loginAndGetCookies(t, srv)
	rec := jsonRequestWithCookies(t, srv, http.MethodPost, "/admin/invites",
		CreateInviteRequest{Name: "Frodi & Carla", MinPlus: 0, MaxPlus: 2, Guests: presets("Frodi & Carla")}, cookies)
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("create invite status = %d; body: %s", rec.Code, rec.Body.String())
	}
	return cookies
}

func firstInviteID(t *testing.T, srv http.Handler, cookies []*http.Cookie) string {
	t.Helper()
	rec := jsonRequestWithCookies(t, srv, http.MethodGet, "/admin/invites", nil, cookies)
	var resp ListInvitesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Invites) == 0 {
		t.Fatal("no invites")
	}
	return resp.Invites[0].ID
}

// jsonRequest performs a request, logging in for a fresh admin cookie when
// withAdminCookie is set. Cookies are obtained per-call rather than from a global.
func jsonRequest(t *testing.T, srv http.Handler, method, path string, body interface{}, withAdminCookie bool) *httptest.ResponseRecorder {
	t.Helper()
	var cookies []*http.Cookie
	if withAdminCookie {
		cookies = loginAndGetCookies(t, srv)
	}
	return jsonRequestWithCookies(t, srv, method, path, body, cookies)
}

type updateFetchNotFoundStore struct{}

func (s *updateFetchNotFoundStore) CreateInvite(ctx context.Context, name string, minPlus, maxPlus int, guests []db.PresetGuest, group bool) (db.Invite, error) {
	return db.Invite{}, nil
}

func (s *updateFetchNotFoundStore) GetInvite(ctx context.Context, id string) (db.Invite, error) {
	return db.Invite{}, db.ErrNotFound
}

func (s *updateFetchNotFoundStore) GetInviteWithGuests(ctx context.Context, id string) (db.Invite, []db.Guest, error) {
	return db.Invite{}, nil, db.ErrNotFound
}

func (s *updateFetchNotFoundStore) ListInvites(ctx context.Context) ([]db.Invite, error) {
	return nil, nil
}

func (s *updateFetchNotFoundStore) UpdateInvite(ctx context.Context, id string, name string, minPlus, maxPlus int, guests []db.PresetGuest, group bool) (db.Invite, error) {
	return db.Invite{ID: id, Name: name, MinPlus: minPlus, MaxPlus: maxPlus}, nil
}

func (s *updateFetchNotFoundStore) DeleteInvite(ctx context.Context, id string) error {
	return nil
}

func (s *updateFetchNotFoundStore) SubmitRSVP(ctx context.Context, inviteID string, guests []db.Guest, submitted bool, message string) (db.Invite, []db.Guest, error) {
	return db.Invite{ID: inviteID}, guests, nil
}

func jsonRequestWithCookies(t *testing.T, srv http.Handler, method, path string, body interface{}, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

// publicRequest issues a request from a given client IP, spelled the way Caddy
// presents it to the service.
func publicRequest(t *testing.T, srv http.Handler, method, path, ip string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(`{"guests":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", ip)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestPublicInviteRoutes_RateLimitedPerIP(t *testing.T) {
	srv, _ := newTestServer(t)

	for i := range publicInviteLimit {
		rec := publicRequest(t, srv, http.MethodGet, "/invites/ZZZZZ", "1.2.3.4")
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d was rate limited, want the first %d allowed through", i+1, publicInviteLimit)
		}
	}

	rec := publicRequest(t, srv, http.MethodGet, "/invites/ZZZZZ", "1.2.3.4")
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("request %d: status = %d, want 429", publicInviteLimit+1, rec.Code)
	}
}

// Lookups and submissions draw on one budget, because an enumerator can guess
// ids against either of them.
func TestPublicInviteRoutes_ShareOneBudgetAcrossLookupAndRSVP(t *testing.T) {
	srv, _ := newTestServer(t)

	for range publicInviteLimit {
		publicRequest(t, srv, http.MethodGet, "/invites/ZZZZZ", "1.2.3.4")
	}
	rec := publicRequest(t, srv, http.MethodPost, "/invites/ZZZZZ/rsvp", "1.2.3.4")

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("RSVP after exhausting lookups: status = %d, want 429", rec.Code)
	}
}

func TestPublicInviteRoutes_BudgetIsNotSharedBetweenClients(t *testing.T) {
	srv, _ := newTestServer(t)

	for range publicInviteLimit + 1 {
		publicRequest(t, srv, http.MethodGet, "/invites/ZZZZZ", "1.2.3.4")
	}
	rec := publicRequest(t, srv, http.MethodGet, "/invites/ZZZZZ", "5.6.7.8")

	if rec.Code == http.StatusTooManyRequests {
		t.Error("a second client was rate limited by the first client's requests")
	}
}

// Admins legitimately click around far faster than a guest, and the routes are
// password protected, so the enumeration budget must not apply to them.
func TestAdminRoutes_AreNotRateLimited(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := createAndLogin(t, srv)

	for i := range publicInviteLimit * 2 {
		rec := jsonRequestWithCookies(t, srv, http.MethodGet, "/admin/invites", nil, cookies)
		if rec.Code != http.StatusOK {
			t.Fatalf("admin request %d: status = %d, want 200", i+1, rec.Code)
		}
	}
}

func presets(names ...string) []PresetGuestInput {
	out := make([]PresetGuestInput, len(names))
	for i, n := range names {
		out[i] = PresetGuestInput{Name: n}
	}
	return out
}

func TestAdminCreateInvite_PresetAlcoholFreeIsReturned(t *testing.T) {
	srv, _ := newTestServer(t)
	body := CreateInviteRequest{Name: "Parent", MinPlus: 0, MaxPlus: 1, Guests: []PresetGuestInput{
		{Name: "Parent"},
		{Name: "Kid", AlcoholFree: true},
	}}

	rec := jsonRequest(t, srv, http.MethodPost, "/admin/invites", body, true)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	var resp InviteWithGuestsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Guests[0].AlcoholFree || !resp.Guests[1].AlcoholFree {
		t.Errorf("alcohol_free = [%v %v], want [false true]", resp.Guests[0].AlcoholFree, resp.Guests[1].AlcoholFree)
	}
}

func TestAdminUpdateInvite_KeepsGuestMatchedByID(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAndGetCookies(t, srv)
	created := decodeInvite(t, jsonRequestWithCookies(t, srv, http.MethodPost, "/admin/invites",
		CreateInviteRequest{Name: "A", MinPlus: 0, MaxPlus: 2, Guests: presets("A", "B", "C")}, cookies))
	path := "/admin/invites/" + created.Invite.ID
	body := UpdateInviteRequest{Name: "A", MinPlus: 0, MaxPlus: 2, Guests: []PresetGuestInput{
		{ID: created.Guests[0].ID, Name: "A"},
		{ID: created.Guests[2].ID, Name: "C", AlcoholFree: true},
	}}

	rec := jsonRequestWithCookies(t, srv, http.MethodPut, path, body, cookies)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	got := decodeInvite(t, rec).Guests
	if len(got) != 2 || got[1].ID != created.Guests[2].ID || got[1].Name != "C" || !got[1].AlcoholFree {
		t.Errorf("guests = %+v, want A then C (id %d, alcohol-free)", got, created.Guests[2].ID)
	}
}

func TestAdminUpdateInvite_StaleGuestIDConflicts(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAndGetCookies(t, srv)
	created := decodeInvite(t, jsonRequestWithCookies(t, srv, http.MethodPost, "/admin/invites",
		CreateInviteRequest{Name: "A", MinPlus: 0, MaxPlus: 1, Guests: presets("A")}, cookies))
	body := UpdateInviteRequest{Name: "A", MinPlus: 0, MaxPlus: 1, Guests: []PresetGuestInput{
		{ID: created.Guests[0].ID + 1000, Name: "A"},
	}}

	rec := jsonRequestWithCookies(t, srv, http.MethodPut, "/admin/invites/"+created.Invite.ID, body, cookies)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409; body: %s", rec.Code, rec.Body.String())
	}
}

func decodeInvite(t *testing.T, rec *httptest.ResponseRecorder) InviteWithGuestsResponse {
	t.Helper()
	var resp InviteWithGuestsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v; body: %s", err, rec.Body.String())
	}
	return resp
}
