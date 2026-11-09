package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func decodeJSONBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("unmarshal: %v; body: %s", err, rec.Body.String())
	}
	return v
}

func createSpeechAs(t *testing.T, srv http.Handler, cookies []*http.Cookie, speaker string) SpeechResponse {
	t.Helper()
	rec := jsonRequestWithCookies(t, srv, http.MethodPost, "/speeches",
		CreateSpeechRequest{Speaker: speaker, Title: "Toast", OriginalLang: "sv"}, cookies)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create speech status = %d; body: %s", rec.Code, rec.Body.String())
	}
	return decodeJSONBody[SpeechResponse](t, rec)
}

func speechPath(id int64, rest string) string {
	return fmt.Sprintf("/speeches/%d%s", id, rest)
}

func TestSpeeches_OpenToTranslatorAndAdmin(t *testing.T) {
	for _, password := range []string{"translator-pw", "admin-pw"} {
		srv, _ := newTestServer(t)
		cookies := loginAs(t, srv, password)

		rec := jsonRequestWithCookies(t, srv, http.MethodGet, "/speeches", nil, cookies)

		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", password, rec.Code)
		}
	}
}

func TestSpeeches_RequireLogin(t *testing.T) {
	srv, _ := newTestServer(t)

	rec := jsonRequest(t, srv, http.MethodGet, "/speeches", nil, false)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestCreateSpeech_ReturnsSpeechWithEmptyTexts(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAs(t, srv, "translator-pw")

	sp := createSpeechAs(t, srv, cookies, "Anna")

	if sp.Speaker != "Anna" || sp.OriginalLang != "sv" || sp.Version != 1 || len(sp.Texts) != 4 {
		t.Errorf("speech = %+v, want Anna/sv at version 1 with 4 texts", sp)
	}
	list := decodeJSONBody[ListSpeechesResponse](t, jsonRequestWithCookies(t, srv, http.MethodGet, "/speeches", nil, cookies))
	if len(list.Speeches) != 1 || list.Speeches[0].ID != sp.ID {
		t.Errorf("list = %+v, want the created speech", list)
	}
}

func TestCreateSpeech_RejectsInvalidInput(t *testing.T) {
	cases := map[string]CreateSpeechRequest{
		"no speaker":       {Speaker: "  ", OriginalLang: "sv"},
		"unknown language": {Speaker: "Anna", OriginalLang: "fr"},
	}
	for name, body := range cases {
		srv, _ := newTestServer(t)
		cookies := loginAs(t, srv, "translator-pw")

		rec := jsonRequestWithCookies(t, srv, http.MethodPost, "/speeches", body, cookies)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, rec.Code)
		}
	}
}

func TestSaveSpeechText_SavesAndReturnsNewVersion(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAs(t, srv, "translator-pw")
	sp := createSpeechAs(t, srv, cookies, "Anna")

	rec := jsonRequestWithCookies(t, srv, http.MethodPut, speechPath(sp.ID, "/texts/is"),
		SaveSpeechTextRequest{Body: "Kæru vinir", Done: true, Version: 1}, cookies)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	text := decodeJSONBody[SpeechTextResponse](t, rec)
	if text.Lang != "is" || text.Body != "Kæru vinir" || !text.Done || text.Version != 2 {
		t.Errorf("text = %+v, want saved is text at version 2", text)
	}
}

func TestSaveSpeechText_StaleVersionConflictsWithCurrentText(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAs(t, srv, "translator-pw")
	sp := createSpeechAs(t, srv, cookies, "Anna")
	path := speechPath(sp.ID, "/texts/en")
	jsonRequestWithCookies(t, srv, http.MethodPut, path, SaveSpeechTextRequest{Body: "Theirs", Version: 1}, cookies)

	rec := jsonRequestWithCookies(t, srv, http.MethodPut, path, SaveSpeechTextRequest{Body: "Mine", Version: 1}, cookies)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body: %s", rec.Code, rec.Body.String())
	}
	conflict := decodeJSONBody[SpeechTextConflictResponse](t, rec)
	if conflict.Current.Body != "Theirs" || conflict.Current.Version != 2 {
		t.Errorf("current = %+v, want their text at version 2", conflict.Current)
	}
}

func TestSaveSpeechText_RejectsUnknownLanguageAndSpeech(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAs(t, srv, "translator-pw")
	sp := createSpeechAs(t, srv, cookies, "Anna")
	body := SaveSpeechTextRequest{Body: "x", Version: 1}

	badLang := jsonRequestWithCookies(t, srv, http.MethodPut, speechPath(sp.ID, "/texts/fr"), body, cookies)
	missing := jsonRequestWithCookies(t, srv, http.MethodPut, speechPath(sp.ID+100, "/texts/en"), body, cookies)
	badID := jsonRequestWithCookies(t, srv, http.MethodPut, "/speeches/abc/texts/en", body, cookies)

	if badLang.Code != http.StatusNotFound || missing.Code != http.StatusNotFound || badID.Code != http.StatusNotFound {
		t.Errorf("statuses = %d %d %d, want 404 404 404", badLang.Code, missing.Code, badID.Code)
	}
}

func TestUpdateSpeech_SavesDetailsAndConflictsWhenStale(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAs(t, srv, "translator-pw")
	sp := createSpeechAs(t, srv, cookies, "Anna")
	path := speechPath(sp.ID, "")

	ok := jsonRequestWithCookies(t, srv, http.MethodPut, path,
		UpdateSpeechRequest{Speaker: "Anna K", Title: "Ræða", OriginalLang: "is", Version: 1}, cookies)
	stale := jsonRequestWithCookies(t, srv, http.MethodPut, path,
		UpdateSpeechRequest{Speaker: "Other", OriginalLang: "sv", Version: 1}, cookies)

	if ok.Code != http.StatusOK {
		t.Fatalf("update status = %d; body: %s", ok.Code, ok.Body.String())
	}
	updated := decodeJSONBody[SpeechResponse](t, ok)
	if updated.Speaker != "Anna K" || updated.OriginalLang != "is" || updated.Version != 2 {
		t.Errorf("updated = %+v, want new details at version 2", updated)
	}
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale status = %d, want 409", stale.Code)
	}
	if current := decodeJSONBody[SpeechConflictResponse](t, stale).Current; current.Speaker != "Anna K" {
		t.Errorf("conflict current = %+v, want the saved details", current)
	}
}

func TestMoveSpeech_ReturnsNewOrder(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAs(t, srv, "translator-pw")
	createSpeechAs(t, srv, cookies, "A")
	b := createSpeechAs(t, srv, cookies, "B")

	rec := jsonRequestWithCookies(t, srv, http.MethodPost, speechPath(b.ID, "/move"), MoveSpeechRequest{Delta: -1}, cookies)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	list := decodeJSONBody[ListSpeechesResponse](t, rec)
	if len(list.Speeches) != 2 || list.Speeches[0].Speaker != "B" || list.Speeches[1].Speaker != "A" {
		t.Errorf("order = %+v, want B then A", list.Speeches)
	}
	bad := jsonRequestWithCookies(t, srv, http.MethodPost, speechPath(b.ID, "/move"), MoveSpeechRequest{Delta: 2}, cookies)
	if bad.Code != http.StatusBadRequest {
		t.Errorf("delta 2 status = %d, want 400", bad.Code)
	}
}

func TestDeleteSpeech_RemovesIt(t *testing.T) {
	srv, _ := newTestServer(t)
	cookies := loginAs(t, srv, "translator-pw")
	sp := createSpeechAs(t, srv, cookies, "Anna")

	rec := jsonRequestWithCookies(t, srv, http.MethodDelete, speechPath(sp.ID, ""), nil, cookies)
	again := jsonRequestWithCookies(t, srv, http.MethodDelete, speechPath(sp.ID, ""), nil, cookies)

	if rec.Code != http.StatusNoContent || again.Code != http.StatusNotFound {
		t.Errorf("statuses = %d %d, want 204 then 404", rec.Code, again.Code)
	}
}
