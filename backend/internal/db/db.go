package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
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
	if _, err := d.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;"); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

// ErrNotFound is returned when a query finds no row.
var ErrNotFound = errors.New("not found")

// Invite is the persisted invite record.
type Invite struct {
	ID        int64
	Name      string
	MinPlus   int
	MaxPlus   int
	Submitted bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Guest is a persisted guest belonging to an invite.
type Guest struct {
	ID                int64
	InviteID          int64
	Name              string
	DietaryPreference string
	AlcoholFree       bool
	IsPrimary         bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Store is the storage contract used by the invite service and handlers.
type Store interface {
	CreateInvite(ctx context.Context, name string, minPlus, maxPlus int, guestNames []string) (Invite, error)
	GetInvite(ctx context.Context, id int64) (Invite, error)
	GetInviteWithGuests(ctx context.Context, id int64) (Invite, []Guest, error)
	ListInvites(ctx context.Context) ([]Invite, error)
	UpdateInvite(ctx context.Context, id int64, name string, minPlus, maxPlus int, guestNames []string) (Invite, error)
	DeleteInvite(ctx context.Context, id int64) error
	SubmitRSVP(ctx context.Context, inviteID int64, guests []Guest, submitted bool) ([]Guest, error)
}

// SQLiteStore implements Store against a *sql.DB.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(d *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: d}
}

func (s *SQLiteStore) CreateInvite(ctx context.Context, name string, minPlus, maxPlus int, guestNames []string) (Invite, error) {
	if len(guestNames) == 0 {
		return Invite{}, errors.New("at least one guest name is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Invite{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		"INSERT INTO invites (name, min_plus, max_plus) VALUES (?, ?, ?)",
		name, minPlus, maxPlus)
	if err != nil {
		return Invite{}, err
	}
	id, _ := res.LastInsertId()
	for i, gname := range guestNames {
		var isPrimary int
		if i == 0 {
			isPrimary = 1
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO guests (invite_id, name, is_primary) VALUES (?, ?, ?)",
			id, gname, isPrimary); err != nil {
			return Invite{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Invite{}, err
	}
	return s.GetInvite(ctx, id)
}

func (s *SQLiteStore) GetInvite(ctx context.Context, id int64) (Invite, error) {
	var inv Invite
	var submitted int
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, min_plus, max_plus, submitted, created_at, updated_at FROM invites WHERE id=?",
		id).Scan(&inv.ID, &inv.Name, &inv.MinPlus, &inv.MaxPlus, &submitted, &inv.CreatedAt, &inv.UpdatedAt)
	if err == sql.ErrNoRows {
		return Invite{}, ErrNotFound
	}
	if err != nil {
		return Invite{}, err
	}
	inv.Submitted = submitted == 1
	return inv, nil
}

func (s *SQLiteStore) GetInviteWithGuests(ctx context.Context, id int64) (Invite, []Guest, error) {
	inv, err := s.GetInvite(ctx, id)
	if err != nil {
		return Invite{}, nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, invite_id, name, dietary_preference, alcohol_free, is_primary, created_at, updated_at FROM guests WHERE invite_id=? ORDER BY is_primary DESC, id ASC",
		id)
	if err != nil {
		return Invite{}, nil, err
	}
	defer rows.Close()

	var guests []Guest
	for rows.Next() {
		var g Guest
		var alcoholFree, isPrimary int
		if err := rows.Scan(&g.ID, &g.InviteID, &g.Name, &g.DietaryPreference, &alcoholFree, &isPrimary, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return Invite{}, nil, err
		}
		g.AlcoholFree = alcoholFree == 1
		g.IsPrimary = isPrimary == 1
		guests = append(guests, g)
	}
	return inv, guests, rows.Err()
}

func (s *SQLiteStore) ListInvites(ctx context.Context) ([]Invite, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, min_plus, max_plus, submitted, created_at, updated_at FROM invites ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []Invite
	for rows.Next() {
		var inv Invite
		var submitted int
		if err := rows.Scan(&inv.ID, &inv.Name, &inv.MinPlus, &inv.MaxPlus, &submitted, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		inv.Submitted = submitted == 1
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

func (s *SQLiteStore) UpdateInvite(ctx context.Context, id int64, name string, minPlus, maxPlus int, guestNames []string) (Invite, error) {
	if len(guestNames) == 0 {
		return Invite{}, errors.New("at least one guest name is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Invite{}, err
	}
	defer tx.Rollback()

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
	var existingIDs []int64
	for rows.Next() {
		var gid int64
		if err := rows.Scan(&gid); err != nil {
			rows.Close()
			return Invite{}, err
		}
		existingIDs = append(existingIDs, gid)
	}
	rows.Close()

	// Reconcile existing guest rows by position; insert new rows for extra names;
	// delete trailing extras. This preserves dietary_preference/alcohol_free on retained rows.
	for i, gname := range guestNames {
		var isPrimary int
		if i == 0 {
			isPrimary = 1
		}
		if i < len(existingIDs) {
			if _, err := tx.ExecContext(ctx,
				"UPDATE guests SET name=?, is_primary=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
				gname, isPrimary, existingIDs[i]); err != nil {
				return Invite{}, err
			}
		} else {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO guests (invite_id, name, is_primary) VALUES (?, ?, ?)",
				id, gname, isPrimary); err != nil {
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

func (s *SQLiteStore) DeleteInvite(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM guests WHERE invite_id=?", id)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, "DELETE FROM invites WHERE id=?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) SubmitRSVP(ctx context.Context, inviteID int64, guests []Guest, submitted bool) ([]Guest, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM guests WHERE invite_id=?", inviteID); err != nil {
		return nil, err
	}

	var result []Guest
	for _, g := range guests {
		var isPrimary int
		if g.IsPrimary {
			isPrimary = 1
		}
		var alcoholFree int
		if g.AlcoholFree {
			alcoholFree = 1
		}
		res, err := tx.ExecContext(ctx,
			"INSERT INTO guests (invite_id, name, dietary_preference, alcohol_free, is_primary) VALUES (?, ?, ?, ?, ?)",
			inviteID, g.Name, g.DietaryPreference, alcoholFree, isPrimary)
		if err != nil {
			return nil, err
		}
		id, _ := res.LastInsertId()
		g.ID = id
		g.InviteID = inviteID
		result = append(result, g)
	}

	var v int
	if submitted {
		v = 1
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE invites SET submitted=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		v, inviteID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
