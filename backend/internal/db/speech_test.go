package db

import (
	"context"
	"errors"
	"testing"

	"wedding/backend/internal/langs"
)

func createSpeech(t *testing.T, store *SQLiteStore, speaker string) Speech {
	t.Helper()
	sp, err := store.CreateSpeech(context.Background(), speaker, "Toast", "sv")
	if err != nil {
		t.Fatalf("CreateSpeech() error: %v", err)
	}
	return sp
}

func speakers(t *testing.T, store *SQLiteStore) []string {
	t.Helper()
	speeches, err := store.ListSpeeches(context.Background())
	if err != nil {
		t.Fatalf("ListSpeeches() error: %v", err)
	}
	names := make([]string, len(speeches))
	for i, sp := range speeches {
		names[i] = sp.Speaker
	}
	return names
}

func TestCreateSpeech_AddsEmptyTextPerLanguageAtTheEnd(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	createSpeech(t, store, "Anna")

	sp := createSpeech(t, store, "Björn")

	if sp.Speaker != "Björn" || sp.Title != "Toast" || sp.OriginalLang != "sv" || sp.Version != 1 {
		t.Errorf("speech = %+v, want Björn/Toast/sv at version 1", sp)
	}
	if got := speakers(t, store); len(got) != 2 || got[1] != "Björn" {
		t.Errorf("speakers = %q, want Björn last", got)
	}
	if len(sp.Texts) != len(langs.All()) {
		t.Fatalf("len(Texts) = %d, want %d", len(sp.Texts), len(langs.All()))
	}
	for i, text := range sp.Texts {
		if text.Lang != langs.All()[i] || text.Body != "" || text.Done || text.Version != 1 {
			t.Errorf("Texts[%d] = %+v, want empty %s at version 1", i, text, langs.All()[i])
		}
	}
}

func TestSaveSpeechText_SavesAndBumpsVersion(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()
	sp := createSpeech(t, store, "Anna")

	saved, err := store.SaveSpeechText(ctx, sp.ID, "is", "Kæru vinir", true, 1)

	if err != nil {
		t.Fatalf("SaveSpeechText() error: %v", err)
	}
	if saved.Body != "Kæru vinir" || !saved.Done || saved.Version != 2 {
		t.Errorf("saved = %+v, want body, done, version 2", saved)
	}
	speeches, err := store.ListSpeeches(ctx)
	if err != nil {
		t.Fatalf("ListSpeeches() error: %v", err)
	}
	for _, text := range speeches[0].Texts {
		if text.Lang == "is" && text.Body != "Kæru vinir" {
			t.Errorf("listed is text = %+v, want saved body", text)
		}
		if text.Lang != "is" && text.Version != 1 {
			t.Errorf("%s text version = %d, want untouched 1", text.Lang, text.Version)
		}
	}
}

func TestSaveSpeechText_StaleVersionIsRejected(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()
	sp := createSpeech(t, store, "Anna")
	if _, err := store.SaveSpeechText(ctx, sp.ID, "en", "First", false, 1); err != nil {
		t.Fatalf("first save error: %v", err)
	}

	_, err := store.SaveSpeechText(ctx, sp.ID, "en", "Second", false, 1)

	if !errors.Is(err, ErrStaleVersion) {
		t.Fatalf("err = %v, want ErrStaleVersion", err)
	}
	current, err := store.GetSpeechText(ctx, sp.ID, "en")
	if err != nil {
		t.Fatalf("GetSpeechText() error: %v", err)
	}
	if current.Body != "First" || current.Version != 2 {
		t.Errorf("current = %+v, want the first save at version 2", current)
	}
}

func TestSaveSpeechText_UnknownSpeechIsNotFound(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	_, err := store.SaveSpeechText(context.Background(), 999, "en", "x", false, 1)

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestUpdateSpeech_ChecksVersion(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()
	sp := createSpeech(t, store, "Anna")

	updated, err := store.UpdateSpeech(ctx, sp.ID, "Anna Karlsdóttir", "Ræða", "is", 1)
	if err != nil {
		t.Fatalf("UpdateSpeech() error: %v", err)
	}
	_, staleErr := store.UpdateSpeech(ctx, sp.ID, "Someone else", "", "sv", 1)

	if updated.Speaker != "Anna Karlsdóttir" || updated.Title != "Ræða" || updated.OriginalLang != "is" || updated.Version != 2 {
		t.Errorf("updated = %+v, want new details at version 2", updated)
	}
	if !errors.Is(staleErr, ErrStaleVersion) {
		t.Errorf("stale update err = %v, want ErrStaleVersion", staleErr)
	}
	if _, err := store.UpdateSpeech(ctx, 999, "x", "", "sv", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown speech err = %v, want ErrNotFound", err)
	}
}

func TestMoveSpeech_SwapsWithNeighbourAndStopsAtTheEnds(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()
	a := createSpeech(t, store, "A")
	createSpeech(t, store, "B")
	c := createSpeech(t, store, "C")

	if err := store.MoveSpeech(ctx, c.ID, -1); err != nil {
		t.Fatalf("MoveSpeech(C up) error: %v", err)
	}
	if err := store.MoveSpeech(ctx, a.ID, -1); err != nil {
		t.Fatalf("MoveSpeech(A up at the top) error: %v", err)
	}

	got := speakers(t, store)
	if len(got) != 3 || got[0] != "A" || got[1] != "C" || got[2] != "B" {
		t.Errorf("order = %q, want [A C B]", got)
	}
	if err := store.MoveSpeech(ctx, 999, 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown speech err = %v, want ErrNotFound", err)
	}
}

func TestDeleteSpeech_RemovesSpeechAndTexts(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()
	sp := createSpeech(t, store, "Anna")

	if err := store.DeleteSpeech(ctx, sp.ID); err != nil {
		t.Fatalf("DeleteSpeech() error: %v", err)
	}

	if got := speakers(t, store); len(got) != 0 {
		t.Errorf("speakers = %q, want none", got)
	}
	var texts int
	if err := store.db.QueryRowContext(ctx, "SELECT count(*) FROM speech_texts").Scan(&texts); err != nil {
		t.Fatalf("count texts: %v", err)
	}
	if texts != 0 {
		t.Errorf("speech_texts rows = %d, want 0 (cascade)", texts)
	}
	if err := store.DeleteSpeech(ctx, sp.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete err = %v, want ErrNotFound", err)
	}
}
