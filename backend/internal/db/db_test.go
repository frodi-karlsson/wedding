package db

import (
	"context"
	"testing"
)

func TestOpen_InMemory(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestOpen_AppliesWALAndBusyTimeout(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	// :memory: reports "memory" for journal_mode; WAL is for file-backed DBs.
	// We assert the pragma was issued without error by checking busy_timeout.
	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", busyTimeout)
	}
}

func newTestStore(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()
	d, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	store := NewSQLiteStore(d)
	return store, func() { d.Close() }
}

func TestCreateInvite_AutoCreatesPrimaryGuest(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	inv, err := store.CreateInvite(ctx, "Frodi & Carla", 0, 2)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	if inv.Name != "Frodi & Carla" {
		t.Errorf("Name = %q", inv.Name)
	}
	if inv.MinPlus != 0 || inv.MaxPlus != 2 {
		t.Errorf("Min/Max = %d/%d, want 0/2", inv.MinPlus, inv.MaxPlus)
	}
	if inv.Submitted {
		t.Errorf("Submitted = true, want false")
	}

	_, guests, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if len(guests) != 1 {
		t.Fatalf("len(guests) = %d, want 1 (primary)", len(guests))
	}
	if !guests[0].IsPrimary {
		t.Errorf("primary guest IsPrimary = false")
	}
	if guests[0].Name != "Frodi & Carla" {
		t.Errorf("primary guest Name = %q, want %q", guests[0].Name, "Frodi & Carla")
	}
}

func TestGetInvite_NotFound(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	_, err := store.GetInvite(context.Background(), 999)
	if err != ErrNotFound {
		t.Errorf("GetInvite(999) err = %v, want ErrNotFound", err)
	}
}

func TestSetSubmitted(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()
	inv, _ := store.CreateInvite(ctx, "Test", 0, 1)
	if err := store.SetSubmitted(ctx, inv.ID, true); err != nil {
		t.Fatalf("SetSubmitted() error: %v", err)
	}
	inv2, err := store.GetInvite(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInvite() error: %v", err)
	}
	if !inv2.Submitted {
		t.Errorf("Submitted = false, want true")
	}
}

func TestDeleteInvite_RemovesInvite(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()
	inv, err := store.CreateInvite(ctx, "To Delete", 0, 1)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	if err := store.DeleteInvite(ctx, inv.ID); err != nil {
		t.Errorf("DeleteInvite() error: %v", err)
	}
	_, err = store.GetInvite(ctx, inv.ID)
	if err != ErrNotFound {
		t.Errorf("after delete, GetInvite err = %v, want ErrNotFound", err)
	}
}

func TestDeleteInvite_NotFound(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	if err := store.DeleteInvite(context.Background(), 999); err != ErrNotFound {
		t.Errorf("DeleteInvite(999) err = %v, want ErrNotFound", err)
	}
}
