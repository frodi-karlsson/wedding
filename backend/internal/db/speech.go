package db

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"time"

	"wedding/backend/internal/langs"
)

// ErrStaleVersion is returned when a save names a version that is no longer
// current, meaning someone else saved the same row in the meantime.
var ErrStaleVersion = errors.New("changed since it was loaded")

// Speech is one speech in the booklet, with a text per language.
type Speech struct {
	ID           int64
	Position     int
	Speaker      string
	Title        string
	OriginalLang string
	Version      int
	UpdatedAt    time.Time
	Texts        []SpeechText
}

// SpeechText is a speech in one language.
type SpeechText struct {
	SpeechID  int64
	Lang      string
	Body      string
	Done      bool
	Version   int
	UpdatedAt time.Time
}

// SpeechStore is the storage contract for the speech booklet.
type SpeechStore interface {
	ListSpeeches(ctx context.Context) ([]Speech, error)
	GetSpeech(ctx context.Context, id int64) (Speech, error)
	CreateSpeech(ctx context.Context, speaker, title, originalLang string) (Speech, error)
	UpdateSpeech(ctx context.Context, id int64, speaker, title, originalLang string, version int) (Speech, error)
	DeleteSpeech(ctx context.Context, id int64) error
	MoveSpeech(ctx context.Context, id int64, delta int) error
	GetSpeechText(ctx context.Context, speechID int64, lang string) (SpeechText, error)
	SaveSpeechText(ctx context.Context, speechID int64, lang, body string, done bool, version int) (SpeechText, error)
}

const speechColumns = "id, position, speaker, title, original_lang, version, updated_at"

const speechTextColumns = "speech_id, lang, body, done, version, updated_at"

// sortTexts puts a speech's texts into langs.All order.
func sortTexts(texts []SpeechText) {
	order := langs.All()
	slices.SortFunc(texts, func(a, b SpeechText) int {
		return slices.Index(order, a.Lang) - slices.Index(order, b.Lang)
	})
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSpeech(row rowScanner) (Speech, error) {
	var sp Speech
	err := row.Scan(&sp.ID, &sp.Position, &sp.Speaker, &sp.Title, &sp.OriginalLang, &sp.Version, &sp.UpdatedAt)
	return sp, err
}

func scanSpeechText(row rowScanner) (SpeechText, error) {
	var text SpeechText
	var done int
	err := row.Scan(&text.SpeechID, &text.Lang, &text.Body, &done, &text.Version, &text.UpdatedAt)
	text.Done = done == 1
	return text, err
}

func (s *SQLiteStore) speechesInOrder(ctx context.Context) ([]Speech, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+speechColumns+" FROM speeches ORDER BY position, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var speeches []Speech
	for rows.Next() {
		sp, err := scanSpeech(rows)
		if err != nil {
			return nil, err
		}
		speeches = append(speeches, sp)
	}
	return speeches, rows.Err()
}

func (s *SQLiteStore) ListSpeeches(ctx context.Context) ([]Speech, error) {
	speeches, err := s.speechesInOrder(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]int, len(speeches))
	for i, sp := range speeches {
		byID[sp.ID] = i
	}

	textRows, err := s.db.QueryContext(ctx, "SELECT "+speechTextColumns+" FROM speech_texts")
	if err != nil {
		return nil, err
	}
	defer textRows.Close()
	for textRows.Next() {
		text, err := scanSpeechText(textRows)
		if err != nil {
			return nil, err
		}
		if i, ok := byID[text.SpeechID]; ok {
			speeches[i].Texts = append(speeches[i].Texts, text)
		}
	}
	for i := range speeches {
		sortTexts(speeches[i].Texts)
	}
	return speeches, textRows.Err()
}

func (s *SQLiteStore) GetSpeech(ctx context.Context, id int64) (Speech, error) {
	sp, err := scanSpeech(s.db.QueryRowContext(ctx, "SELECT "+speechColumns+" FROM speeches WHERE id=?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Speech{}, ErrNotFound
	}
	if err != nil {
		return Speech{}, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+speechTextColumns+" FROM speech_texts WHERE speech_id=?", id)
	if err != nil {
		return Speech{}, err
	}
	defer rows.Close()
	for rows.Next() {
		text, err := scanSpeechText(rows)
		if err != nil {
			return Speech{}, err
		}
		sp.Texts = append(sp.Texts, text)
	}
	sortTexts(sp.Texts)
	return sp, rows.Err()
}

func (s *SQLiteStore) CreateSpeech(ctx context.Context, speaker, title, originalLang string) (Speech, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Speech{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		"INSERT INTO speeches (position, speaker, title, original_lang) VALUES ((SELECT COALESCE(MAX(position), 0) + 1 FROM speeches), ?, ?, ?)",
		speaker, title, originalLang)
	if err != nil {
		return Speech{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Speech{}, err
	}
	for _, lang := range langs.All() {
		if _, err := tx.ExecContext(ctx, "INSERT INTO speech_texts (speech_id, lang) VALUES (?, ?)", id, lang); err != nil {
			return Speech{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Speech{}, err
	}
	return s.GetSpeech(ctx, id)
}

// staleOrMissing tells apart the two reasons a version-checked UPDATE touched
// no rows: the row is gone, or its version moved on.
func (s *SQLiteStore) staleOrMissing(ctx context.Context, query string, args ...any) error {
	var n int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return ErrStaleVersion
}

func (s *SQLiteStore) UpdateSpeech(ctx context.Context, id int64, speaker, title, originalLang string, version int) (Speech, error) {
	res, err := s.db.ExecContext(ctx,
		"UPDATE speeches SET speaker=?, title=?, original_lang=?, version=version+1, updated_at=CURRENT_TIMESTAMP WHERE id=? AND version=?",
		speaker, title, originalLang, id, version)
	if err != nil {
		return Speech{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Speech{}, s.staleOrMissing(ctx, "SELECT count(*) FROM speeches WHERE id=?", id)
	}
	return s.GetSpeech(ctx, id)
}

func (s *SQLiteStore) DeleteSpeech(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM speeches WHERE id=?", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// MoveSpeech swaps the speech with its neighbour delta places away (-1 up,
// +1 down). Moving past either end leaves the order unchanged.
func (s *SQLiteStore) MoveSpeech(ctx context.Context, id int64, delta int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	order, err := speechIDsInOrderTx(ctx, tx)
	if err != nil {
		return err
	}

	from := -1
	for i, sid := range order {
		if sid == id {
			from = i
		}
	}
	if from < 0 {
		return ErrNotFound
	}
	to := from + delta
	if to < 0 || to >= len(order) {
		return nil
	}
	order[from], order[to] = order[to], order[from]
	// Renumber everything so positions stay dense and unique.
	for i, sid := range order {
		if _, err := tx.ExecContext(ctx, "UPDATE speeches SET position=? WHERE id=?", i+1, sid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) GetSpeechText(ctx context.Context, speechID int64, lang string) (SpeechText, error) {
	text, err := scanSpeechText(s.db.QueryRowContext(ctx,
		"SELECT "+speechTextColumns+" FROM speech_texts WHERE speech_id=? AND lang=?", speechID, lang))
	if errors.Is(err, sql.ErrNoRows) {
		return SpeechText{}, ErrNotFound
	}
	return text, err
}

func (s *SQLiteStore) SaveSpeechText(ctx context.Context, speechID int64, lang, body string, done bool, version int) (SpeechText, error) {
	res, err := s.db.ExecContext(ctx,
		"UPDATE speech_texts SET body=?, done=?, version=version+1, updated_at=CURRENT_TIMESTAMP WHERE speech_id=? AND lang=? AND version=?",
		body, done, speechID, lang, version)
	if err != nil {
		return SpeechText{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return SpeechText{}, s.staleOrMissing(ctx, "SELECT count(*) FROM speech_texts WHERE speech_id=? AND lang=?", speechID, lang)
	}
	return s.GetSpeechText(ctx, speechID, lang)
}

func speechIDsInOrderTx(ctx context.Context, tx *sql.Tx) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id FROM speeches ORDER BY position, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
