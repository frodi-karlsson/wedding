package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	a := auth.New("admin-pw", "session-secret")
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
		CreateInviteRequest{Name: "New Couple", MinPlus: 0, MaxPlus: 1}, true)
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
		CreateInviteRequest{Name: "Frodi & Carla", MinPlus: 0, MaxPlus: 2}, cookies)
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("create invite status = %d; body: %s", rec.Code, rec.Body.String())
	}
}

func firstInviteID(t *testing.T, srv http.Handler) int64 {
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
