package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	adminCookies = nil
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
	a := auth.New("admin-pw", "session-secret", false)
	return New(svc, a, []string{"https://carlaochfrodi.wedding"}), fakeEmail
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
	rec := jsonRequest(t, srv, http.MethodGet, "/invites/999", nil, false)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestSubmitRSVP_Valid_SendsEmailAndReturnsSaved(t *testing.T) {
	srv, fakeEmail := newTestServer(t)
	createAndLogin(t, srv)
	id := firstInviteID(t, srv)

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
	createAndLogin(t, srv)
	id := firstInviteID(t, srv)

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

func TestAdminLogin_CorrectPassword(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/login", LoginRequest{Password: "admin-pw"}, false)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "wedding_admin" {
		t.Errorf("expected wedding_admin cookie, got %+v", cookies)
	}
}

func TestAdminLogin_WrongPassword(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/login", LoginRequest{Password: "wrong"}, false)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAdminLogin_BodyLimit_Returns413(t *testing.T) {
	srv, _ := newTestServer(t)
	body := `{"password":"` + strings.Repeat("a", 64*1024) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
}

func TestAdminLogin_RateLimitedAfter5Failures(t *testing.T) {
	srv, _ := newTestServer(t)
	// Set X-Forwarded-For so clientIP extracts a stable IP (simulating Caddy).
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(`{"password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rec.Code)
		}
	}
	// 6th attempt: blocked.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(`{"password":"wrong"}`))
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
		CreateInviteRequest{Name: "New Couple", MinPlus: 0, MaxPlus: 1, GuestNames: []string{"New Couple"}}, true)
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
	createAndLogin(t, srv)
	id := firstInviteID(t, srv)

	rec := jsonRequest(t, srv, http.MethodDelete, "/admin/invites/"+fmt.Sprint(id), nil, true)
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
	rec := jsonRequest(t, srv, http.MethodDelete, "/admin/invites/999", nil, true)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminCreateInvite_WithPresetGuests(t *testing.T) {
	srv, _ := newTestServer(t)
	body := CreateInviteRequest{Name: "Frodi & Carla", MinPlus: 0, MaxPlus: 2, GuestNames: []string{"Frodi & Carla", "Friend"}}
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
	body := CreateInviteRequest{Name: "X", MinPlus: 0, MaxPlus: 1, GuestNames: nil}
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/invites", body, true)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestAdminCreateInvite_FirstNameMismatch(t *testing.T) {
	srv, _ := newTestServer(t)
	body := CreateInviteRequest{Name: "X", MinPlus: 0, MaxPlus: 1, GuestNames: []string{"Y"}}
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/invites", body, true)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestAdminCreateInvite_InvalidMinMax(t *testing.T) {
	srv, _ := newTestServer(t)
	body := CreateInviteRequest{Name: "X", MinPlus: 3, MaxPlus: 1, GuestNames: []string{"X"}}
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/invites", body, true)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestAdminUpdateInvite_GuestFetchNotFound(t *testing.T) {
	svc := invite.NewService(&updateFetchNotFoundStore{}, &email.Fake{})
	a := auth.New("admin-pw", "session-secret", false)
	srv := New(svc, a, []string{"https://example.com"})
	rec := jsonRequest(t, srv, http.MethodPut, "/admin/invites/1",
		UpdateInviteRequest{Name: "X", MinPlus: 0, MaxPlus: 1, GuestNames: []string{"X"}}, true)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

// --- helpers ---

var adminCookies []*http.Cookie

func loginAndGetCookies(t *testing.T, srv http.Handler) []*http.Cookie {
	t.Helper()
	rec := jsonRequest(t, srv, http.MethodPost, "/admin/login", LoginRequest{Password: "admin-pw"}, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d", rec.Code)
	}
	return rec.Result().Cookies()
}

func createAndLogin(t *testing.T, srv http.Handler) {
	t.Helper()
	cookies := loginAndGetCookies(t, srv)
	adminCookies = cookies
	rec := jsonRequestWithCookies(t, srv, http.MethodPost, "/admin/invites",
		CreateInviteRequest{Name: "Frodi & Carla", MinPlus: 0, MaxPlus: 2, GuestNames: []string{"Frodi & Carla"}}, cookies)
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("create invite status = %d; body: %s", rec.Code, rec.Body.String())
	}
}

func firstInviteID(t *testing.T, srv http.Handler) string {
	t.Helper()
	rec := jsonRequestWithCookies(t, srv, http.MethodGet, "/admin/invites", nil, adminCookies)
	var resp ListInvitesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Invites) == 0 {
		t.Fatal("no invites")
	}
	return resp.Invites[0].ID
}

func jsonRequest(t *testing.T, srv http.Handler, method, path string, body interface{}, withAdminCookie bool) *httptest.ResponseRecorder {
	t.Helper()
	var cookies []*http.Cookie
	if withAdminCookie {
		if adminCookies == nil {
			adminCookies = loginAndGetCookies(t, srv)
		}
		cookies = adminCookies
	}
	return jsonRequestWithCookies(t, srv, method, path, body, cookies)
}

type updateFetchNotFoundStore struct{}

func (s *updateFetchNotFoundStore) CreateInvite(ctx context.Context, name string, minPlus, maxPlus int, guestNames []string) (db.Invite, error) {
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

func (s *updateFetchNotFoundStore) UpdateInvite(ctx context.Context, id string, name string, minPlus, maxPlus int, guestNames []string) (db.Invite, error) {
	return db.Invite{ID: id, Name: name, MinPlus: minPlus, MaxPlus: maxPlus}, nil
}

func (s *updateFetchNotFoundStore) DeleteInvite(ctx context.Context, id string) error {
	return nil
}

func (s *updateFetchNotFoundStore) SubmitRSVP(ctx context.Context, inviteID string, guests []db.Guest, submitted bool) ([]db.Guest, error) {
	return guests, nil
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
