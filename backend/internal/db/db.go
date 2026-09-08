package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"modernc.org/sqlite"
)

// Open opens a SQLite database at path with WAL mode and a 5s busy timeout.
// Use ":memory:" for an in-memory database (tests).
func Open(path string) (*sql.DB, error) {
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// SQLite pragmas set with Exec are connection-local. Restrict the pool to
	// a single connection so WAL, busy timeout, and foreign key settings always
	// apply to every query on this *sql.DB.
	d.SetMaxOpenConns(1)
	if _, err := d.ExecContext(context.Background(), "PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;"); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

// ErrNotFound is returned when a query finds no row.
var ErrNotFound = errors.New("not found")

// ErrNoGuestNames is returned when creating or updating an invite with no guest names.
var ErrNoGuestNames = errors.New("at least one guest name is required")

// ErrIDCollision is returned when CreateInvite drew an already-taken id on
// every attempt. At the configured id length this needs the id space to be
// close to exhausted, so in practice it signals a broken id generator rather
// than bad luck.
var ErrIDCollision = errors.New("could not generate an unused invite id")

// Invite is the persisted invite record.
type Invite struct {
	ID        string
	Name      string
	MinPlus   int
	MaxPlus   int
	Submitted bool
	Message   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Guest is a persisted guest belonging to an invite.
type Guest struct {
	ID                int64
	InviteID          string
	Name              string
	DietaryPreference string
	AlcoholFree       bool
	IsPrimary         bool
	CoPrimary         bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Store is the storage contract used by the invite service and handlers.
// group=true marks a "group" invite: every preset guest is stored as a
// co-primary (no single primary). group=false is the standard shape where the
// first guest is the primary and the rest are plain pluses.
type Store interface {
	CreateInvite(ctx context.Context, name string, minPlus, maxPlus int, guestNames []string, group bool) (Invite, error)
	GetInvite(ctx context.Context, id string) (Invite, error)
	GetInviteWithGuests(ctx context.Context, id string) (Invite, []Guest, error)
	ListInvites(ctx context.Context) ([]Invite, error)
	UpdateInvite(ctx context.Context, id string, name string, minPlus, maxPlus int, guestNames []string, group bool) (Invite, error)
	DeleteInvite(ctx context.Context, id string) error
	SubmitRSVP(ctx context.Context, inviteID string, guests []Guest, submitted bool, message string) (Invite, []Guest, error)
}

// presetFlags returns the (is_primary, co_primary) column values for a preset
// guest at position i, given the invite shape.
func presetFlags(i int, group bool) (isPrimary, coPrimary int) {
	if group {
		return 0, 1
	}
	if i == 0 {
		return 1, 0
	}
	return 0, 0
}

// SQLiteStore implements Store against a *sql.DB. newID is a field rather than
// a direct call to generateID so tests can force the id collisions that are
// otherwise too rare to reach.
type SQLiteStore struct {
	db    *sql.DB
	newID func() string
}

func NewSQLiteStore(d *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: d, newID: generateID}
}

// createInviteIDAttempts bounds the retry loop that handles an id collision.
// Ids are short enough that a collision is possible (roughly 1 in 6800 across a
// hundred invites), so the insert has to cope with one, but the retry must not
// spin forever if the id space really is exhausted.
const createInviteIDAttempts = 5

// SQLite extended result codes for a uniqueness violation, which is how a
// colliding invite id surfaces from the INSERT.
const (
	sqliteConstraintUnique     = 2067
	sqliteConstraintPrimaryKey = 1555
)

// isIDConflict reports whether err is SQLite rejecting an insert because the id
// is already taken, as opposed to any other failure.
func isIDConflict(err error) bool {
	sqliteErr, ok := errors.AsType[*sqlite.Error](err)
	if !ok {
		return false
	}
	code := sqliteErr.Code()
	return code == sqliteConstraintPrimaryKey || code == sqliteConstraintUnique
}

func (s *SQLiteStore) CreateInvite(ctx context.Context, name string, minPlus, maxPlus int, guestNames []string, group bool) (Invite, error) {
	if len(guestNames) == 0 {
		return Invite{}, ErrNoGuestNames
	}
	for range createInviteIDAttempts {
		id, err := s.insertInvite(ctx, name, minPlus, maxPlus, guestNames, group)
		if err == nil {
			return s.GetInvite(ctx, id)
		}
		if !isIDConflict(err) {
			return Invite{}, err
		}
	}
	return Invite{}, fmt.Errorf("%w after %d attempts", ErrIDCollision, createInviteIDAttempts)
}

func (s *SQLiteStore) GetInvite(ctx context.Context, id string) (Invite, error) {
	var inv Invite
	var submitted int
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, min_plus, max_plus, submitted, message, created_at, updated_at FROM invites WHERE id=?",
		id).Scan(&inv.ID, &inv.Name, &inv.MinPlus, &inv.MaxPlus, &submitted, &inv.Message, &inv.CreatedAt, &inv.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Invite{}, ErrNotFound
	}
	if err != nil {
		return Invite{}, err
	}
	inv.Submitted = submitted == 1
	return inv, nil
}

func (s *SQLiteStore) GetInviteWithGuests(ctx context.Context, id string) (Invite, []Guest, error) {
	inv, err := s.GetInvite(ctx, id)
	if err != nil {
		return Invite{}, nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, invite_id, name, dietary_preference, alcohol_free, is_primary, co_primary, created_at, updated_at FROM guests WHERE invite_id=? ORDER BY is_primary DESC, co_primary DESC, id ASC",
		id)
	if err != nil {
		return Invite{}, nil, err
	}
	defer rows.Close()

	var guests []Guest
	for rows.Next() {
		var g Guest
		var alcoholFree, isPrimary, coPrimary int
		if err := rows.Scan(&g.ID, &g.InviteID, &g.Name, &g.DietaryPreference, &alcoholFree, &isPrimary, &coPrimary, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return Invite{}, nil, err
		}
		g.AlcoholFree = alcoholFree == 1
		g.IsPrimary = isPrimary == 1
		g.CoPrimary = coPrimary == 1
		guests = append(guests, g)
	}
	return inv, guests, rows.Err()
}

func (s *SQLiteStore) ListInvites(ctx context.Context) ([]Invite, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, min_plus, max_plus, submitted, message, created_at, updated_at FROM invites ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []Invite
	for rows.Next() {
		var inv Invite
		var submitted int
		if err := rows.Scan(&inv.ID, &inv.Name, &inv.MinPlus, &inv.MaxPlus, &submitted, &inv.Message, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		inv.Submitted = submitted == 1
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

func (s *SQLiteStore) UpdateInvite(ctx context.Context, id, name string, minPlus, maxPlus int, guestNames []string, group bool) (Invite, error) {
	if len(guestNames) == 0 {
		return Invite{}, ErrNoGuestNames
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Invite{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		"UPDATE invites SET name=?, min_plus=?, max_plus=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		name, minPlus, maxPlus, id)
	if err != nil {
		return Invite{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Invite{}, ErrNotFound
	}

	rows, err := tx.QueryContext(ctx,
		"SELECT id FROM guests WHERE invite_id=? ORDER BY is_primary DESC, id ASC", id)
	if err != nil {
		return Invite{}, err
	}
	defer rows.Close()

	var existingIDs []int64
	for rows.Next() {
		var gid int64
		if err := rows.Scan(&gid); err != nil {
			return Invite{}, err
		}
		existingIDs = append(existingIDs, gid)
	}
	if err := rows.Err(); err != nil {
		return Invite{}, err
	}

	// Reconcile existing guest rows by position; insert new rows for extra names;
	// delete trailing extras. This preserves dietary_preference/alcohol_free on retained rows.
	for i, gname := range guestNames {
		isPrimary, coPrimary := presetFlags(i, group)
		if i < len(existingIDs) {
			if _, err := tx.ExecContext(ctx,
				"UPDATE guests SET name=?, is_primary=?, co_primary=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
				gname, isPrimary, coPrimary, existingIDs[i]); err != nil {
				return Invite{}, err
			}
		} else {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO guests (invite_id, name, is_primary, co_primary) VALUES (?, ?, ?, ?)",
				id, gname, isPrimary, coPrimary); err != nil {
				return Invite{}, err
			}
		}
	}
	if len(guestNames) < len(existingIDs) {
		for _, gid := range existingIDs[len(guestNames):] {
			if _, err := tx.ExecContext(ctx, "DELETE FROM guests WHERE id=?", gid); err != nil {
				return Invite{}, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return Invite{}, err
	}
	return s.GetInvite(ctx, id)
}

func (s *SQLiteStore) DeleteInvite(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM invites WHERE id=?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SubmitRSVP replaces the invite's guests, marks it submitted, and stores the
// message (all in one transaction), then returns the resulting invite and guest
// rows read back from within that same transaction. Returning the post-write
// state lets callers use it directly (email body, HTTP response) without a second
// round-trip that could observe stale or concurrently-modified data.
func (s *SQLiteStore) SubmitRSVP(ctx context.Context, inviteID string, guests []Guest, submitted bool, message string) (Invite, []Guest, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Invite{}, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "DELETE FROM guests WHERE invite_id=?", inviteID); err != nil {
		return Invite{}, nil, err
	}

	for _, g := range guests {
		var isPrimary int
		if g.IsPrimary {
			isPrimary = 1
		}
		var coPrimary int
		if g.CoPrimary {
			coPrimary = 1
		}
		var alcoholFree int
		if g.AlcoholFree {
			alcoholFree = 1
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO guests (invite_id, name, dietary_preference, alcohol_free, is_primary, co_primary) VALUES (?, ?, ?, ?, ?, ?)",
			inviteID, g.Name, g.DietaryPreference, alcoholFree, isPrimary, coPrimary); err != nil {
			return Invite{}, nil, err
		}
	}

	var v int
	if submitted {
		v = 1
	}
	res, err := tx.ExecContext(ctx,
		"UPDATE invites SET submitted=?, message=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		v, message, inviteID)
	if err != nil {
		return Invite{}, nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Invite{}, nil, ErrNotFound
	}

	// Read the post-write invite + guests from within the same transaction so
	// the returned state is exactly what was just committed.
	inv, err := getInviteTx(ctx, tx, inviteID)
	if err != nil {
		return Invite{}, nil, err
	}
	savedGuests, err := getGuestsTx(ctx, tx, inviteID)
	if err != nil {
		return Invite{}, nil, err
	}

	if err := tx.Commit(); err != nil {
		return Invite{}, nil, err
	}
	return inv, savedGuests, nil
}

// insertInvite writes one invite and its preset guests under a freshly drawn
// id, returning that id. The whole attempt is a single transaction, so an id
// collision rolls back the guest rows along with it and leaves nothing behind
// for the caller to retry around.
func (s *SQLiteStore) insertInvite(ctx context.Context, name string, minPlus, maxPlus int, guestNames []string, group bool) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	id := s.newID()
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO invites (id, name, min_plus, max_plus) VALUES (?, ?, ?, ?)",
		id, name, minPlus, maxPlus); err != nil {
		return "", err
	}
	for i, gname := range guestNames {
		isPrimary, coPrimary := presetFlags(i, group)
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO guests (invite_id, name, is_primary, co_primary) VALUES (?, ?, ?, ?)",
			id, gname, isPrimary, coPrimary); err != nil {
			return "", err
		}
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}

// getInviteTx reads a single invite within a transaction.
func getInviteTx(ctx context.Context, tx *sql.Tx, id string) (Invite, error) {
	var inv Invite
	var submitted int
	err := tx.QueryRowContext(ctx,
		"SELECT id, name, min_plus, max_plus, submitted, message, created_at, updated_at FROM invites WHERE id=?",
		id).Scan(&inv.ID, &inv.Name, &inv.MinPlus, &inv.MaxPlus, &submitted, &inv.Message, &inv.CreatedAt, &inv.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Invite{}, ErrNotFound
	}
	if err != nil {
		return Invite{}, err
	}
	inv.Submitted = submitted == 1
	return inv, nil
}

// getGuestsTx reads an invite's guests within a transaction, in the same order
// as GetInviteWithGuests.
func getGuestsTx(ctx context.Context, tx *sql.Tx, inviteID string) ([]Guest, error) {
	rows, err := tx.QueryContext(ctx,
		"SELECT id, invite_id, name, dietary_preference, alcohol_free, is_primary, co_primary, created_at, updated_at FROM guests WHERE invite_id=? ORDER BY is_primary DESC, co_primary DESC, id ASC",
		inviteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guests []Guest
	for rows.Next() {
		var g Guest
		var alcoholFree, isPrimary, coPrimary int
		if err := rows.Scan(&g.ID, &g.InviteID, &g.Name, &g.DietaryPreference, &alcoholFree, &isPrimary, &coPrimary, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		g.AlcoholFree = alcoholFree == 1
		g.IsPrimary = isPrimary == 1
		g.CoPrimary = coPrimary == 1
		guests = append(guests, g)
	}
	return guests, rows.Err()
}
