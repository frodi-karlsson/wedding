package server

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"wedding/backend/internal/db"
	"wedding/backend/internal/langs"
)

// maxSpeechTextRunes bounds one language's text. A three minute speech is a few
// thousand characters, so this leaves plenty of room.
const maxSpeechTextRunes = 20000

const msgSpeechNotFound = "speech not found"

func registerSpeechRoutes(mux *http.ServeMux, speeches db.SpeechStore) {
	mux.HandleFunc("GET /speeches", handleListSpeeches(speeches))
	mux.HandleFunc("POST /speeches", handleCreateSpeech(speeches))
	mux.HandleFunc("PUT /speeches/{id}", handleUpdateSpeech(speeches))
	mux.HandleFunc("DELETE /speeches/{id}", handleDeleteSpeech(speeches))
	mux.HandleFunc("POST /speeches/{id}/move", handleMoveSpeech(speeches))
	mux.HandleFunc("PUT /speeches/{id}/texts/{lang}", handleSaveSpeechText(speeches))
}

func toSpeechTextResponse(t *db.SpeechText) SpeechTextResponse {
	return SpeechTextResponse{Lang: t.Lang, Body: t.Body, Done: t.Done, Version: t.Version}
}

func toSpeechResponse(sp *db.Speech) SpeechResponse {
	texts := make([]SpeechTextResponse, len(sp.Texts))
	for i := range sp.Texts {
		texts[i] = toSpeechTextResponse(&sp.Texts[i])
	}
	return SpeechResponse{
		ID:           sp.ID,
		Position:     sp.Position,
		Speaker:      sp.Speaker,
		Title:        sp.Title,
		OriginalLang: sp.OriginalLang,
		Version:      sp.Version,
		Texts:        texts,
	}
}

func isSpeechLang(lang string) bool {
	return slices.Contains(langs.All(), lang)
}

// validateSpeechDetails returns a message describing the first problem, or ""
// when the details are acceptable.
func validateSpeechDetails(speaker, originalLang string) string {
	if strings.TrimSpace(speaker) == "" {
		return "speaker is required"
	}
	if !isSpeechLang(originalLang) {
		return "original_lang must be one of sv, is, en, de"
	}
	return ""
}

func speechIDFromPath(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

// decodeBody decodes the JSON request body into v, answering 413 or 400 itself
// and reporting false when the body was unusable.
func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := decodeJSON(r, v); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func writeSpeechList(w http.ResponseWriter, r *http.Request, speeches db.SpeechStore) {
	list, err := speeches.ListSpeeches(r.Context())
	if err != nil {
		logInternalError(w, err)
		return
	}
	resp := ListSpeechesResponse{Speeches: make([]SpeechResponse, len(list))}
	for i := range list {
		resp.Speeches[i] = toSpeechResponse(&list[i])
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleListSpeeches(speeches db.SpeechStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeSpeechList(w, r, speeches)
	}
}

func handleCreateSpeech(speeches db.SpeechStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateSpeechRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if msg := validateSpeechDetails(req.Speaker, req.OriginalLang); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		sp, err := speeches.CreateSpeech(r.Context(), strings.TrimSpace(req.Speaker), strings.TrimSpace(req.Title), req.OriginalLang)
		if err != nil {
			logInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toSpeechResponse(&sp))
	}
}

func handleUpdateSpeech(speeches db.SpeechStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := speechIDFromPath(r)
		if !ok {
			writeError(w, http.StatusNotFound, msgSpeechNotFound)
			return
		}
		var req UpdateSpeechRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if msg := validateSpeechDetails(req.Speaker, req.OriginalLang); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		sp, err := speeches.UpdateSpeech(r.Context(), id, strings.TrimSpace(req.Speaker), strings.TrimSpace(req.Title), req.OriginalLang, req.Version)
		switch {
		case errors.Is(err, db.ErrNotFound):
			writeError(w, http.StatusNotFound, msgSpeechNotFound)
		case errors.Is(err, db.ErrStaleVersion):
			writeSpeechConflict(w, r, speeches, id)
		case err != nil:
			logInternalError(w, err)
		default:
			writeJSON(w, http.StatusOK, toSpeechResponse(&sp))
		}
	}
}

// writeSpeechConflict answers a stale speech save with the speech as it now
// stands, so the editor can show it next to the rejected edit.
func writeSpeechConflict(w http.ResponseWriter, r *http.Request, speeches db.SpeechStore, id int64) {
	sp, err := speeches.GetSpeech(r.Context(), id)
	switch {
	case errors.Is(err, db.ErrNotFound):
		writeError(w, http.StatusNotFound, msgSpeechNotFound)
	case err != nil:
		logInternalError(w, err)
	default:
		writeJSON(w, http.StatusConflict, SpeechConflictResponse{
			Error:   "someone else changed this speech",
			Current: toSpeechResponse(&sp),
		})
	}
}

func handleDeleteSpeech(speeches db.SpeechStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := speechIDFromPath(r)
		if !ok {
			writeError(w, http.StatusNotFound, msgSpeechNotFound)
			return
		}
		err := speeches.DeleteSpeech(r.Context(), id)
		switch {
		case errors.Is(err, db.ErrNotFound):
			writeError(w, http.StatusNotFound, msgSpeechNotFound)
		case err != nil:
			logInternalError(w, err)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func handleMoveSpeech(speeches db.SpeechStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := speechIDFromPath(r)
		if !ok {
			writeError(w, http.StatusNotFound, msgSpeechNotFound)
			return
		}
		var req MoveSpeechRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if req.Delta != -1 && req.Delta != 1 {
			writeError(w, http.StatusBadRequest, "delta must be -1 or 1")
			return
		}
		err := speeches.MoveSpeech(r.Context(), id, req.Delta)
		switch {
		case errors.Is(err, db.ErrNotFound):
			writeError(w, http.StatusNotFound, msgSpeechNotFound)
		case err != nil:
			logInternalError(w, err)
		default:
			writeSpeechList(w, r, speeches)
		}
	}
}

func handleSaveSpeechText(speeches db.SpeechStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := speechIDFromPath(r)
		lang := r.PathValue("lang")
		if !ok || !isSpeechLang(lang) {
			writeError(w, http.StatusNotFound, msgSpeechNotFound)
			return
		}
		var req SaveSpeechTextRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if utf8.RuneCountInString(req.Body) > maxSpeechTextRunes {
			writeError(w, http.StatusBadRequest, "text is too long")
			return
		}
		text, err := speeches.SaveSpeechText(r.Context(), id, lang, req.Body, req.Done, req.Version)
		switch {
		case errors.Is(err, db.ErrNotFound):
			writeError(w, http.StatusNotFound, msgSpeechNotFound)
		case errors.Is(err, db.ErrStaleVersion):
			current, getErr := speeches.GetSpeechText(r.Context(), id, lang)
			if getErr != nil {
				logInternalError(w, getErr)
				return
			}
			writeJSON(w, http.StatusConflict, SpeechTextConflictResponse{
				Error:   "someone else changed this text",
				Current: toSpeechTextResponse(&current),
			})
		case err != nil:
			logInternalError(w, err)
		default:
			writeJSON(w, http.StatusOK, toSpeechTextResponse(&text))
		}
	}
}
