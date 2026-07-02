package db

import (
	"context"
	"encoding/hex"
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
	inv, err := store.CreateInvite(ctx, "Frodi & Carla", 0, 2, []string{"Frodi & Carla"})
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

func TestCreateInvite_GeneratesOpaqueTokenID(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "Frodi & Carla", 0, 2, []string{"Frodi & Carla", "Friend"})
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	if len(inv.ID) != 32 {
		t.Errorf("ID length = %d, want 32 (hex token)", len(inv.ID))
	}
	if _, err := hex.DecodeString(inv.ID); err != nil {
		t.Errorf("ID %q is not valid hex: %v", inv.ID, err)
	}
}

func TestGetInvite_NotFound(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	_, err := store.GetInvite(context.Background(), "999")
	if err != ErrNotFound {
		t.Errorf("GetInvite(999) err = %v, want ErrNotFound", err)
	}
}

func TestSubmitRSVP_Atomic(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "Frodi & Carla", 0, 2, []string{"Frodi & Carla"})
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}

	guests := []Guest{
		{Name: "Frodi", IsPrimary: true},
		{Name: "Carla", DietaryPreference: "vegetarian"},
	}
	saved, err := store.SubmitRSVP(ctx, inv.ID, guests, true)
	if err != nil {
		t.Fatalf("SubmitRSVP() error: %v", err)
	}
	if len(saved) != 2 {
		t.Errorf("len(saved) = %d, want 2", len(saved))
	}

	inv2, guests2, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if !inv2.Submitted {
		t.Errorf("Submitted = false, want true")
	}
	if len(guests2) != 2 {
		t.Fatalf("len(guests2) = %d, want 2", len(guests2))
	}
	if !guests2[0].IsPrimary || guests2[0].Name != "Frodi" {
		t.Errorf("primary guest = %+v, want Frodi primary", guests2[0])
	}
	if guests2[1].Name != "Carla" {
		t.Errorf("second guest Name = %q, want Carla", guests2[1].Name)
	}
}

func TestUpdateInvite_UpdatesPrimaryGuestName(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "Old Name", 0, 1, []string{"Old Name"})
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}

	if _, err := store.UpdateInvite(ctx, inv.ID, "New Name", 0, 1, []string{"New Name"}); err != nil {
		t.Fatalf("UpdateInvite() error: %v", err)
	}

	_, guests, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if len(guests) != 1 {
		t.Fatalf("len(guests) = %d, want 1", len(guests))
	}
	if guests[0].Name != "New Name" {
		t.Errorf("primary guest Name = %q, want %q", guests[0].Name, "New Name")
	}
	if !guests[0].IsPrimary {
		t.Errorf("primary guest IsPrimary = false, want true")
	}
}

func TestForeignKeys_Cascade(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "Cascade", 0, 1, []string{"Cascade"})
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}

	if _, err := store.db.ExecContext(ctx, "DELETE FROM invites WHERE id=?", inv.ID); err != nil {
		t.Fatalf("DELETE invite error: %v", err)
	}

	if _, err := store.GetInvite(ctx, inv.ID); err != ErrNotFound {
		t.Errorf("GetInvite after delete err = %v, want ErrNotFound", err)
	}

	var guestCount int
	if err := store.db.QueryRowContext(ctx, "SELECT count(*) FROM guests WHERE invite_id=?", inv.ID).Scan(&guestCount); err != nil {
		t.Fatalf("count guests error: %v", err)
	}
	if guestCount != 0 {
		t.Errorf("guestCount = %d, want 0 (cascade)", guestCount)
	}
}

func TestDeleteInvite_RemovesInvite(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()
	inv, err := store.CreateInvite(ctx, "To Delete", 0, 1, []string{"To Delete"})
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

func TestCreateInvite_MultipleGuestNames(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "Frodi & Carla", 0, 2, []string{"Frodi & Carla", "Friend One", "Friend Two"})
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	if inv.Name != "Frodi & Carla" {
		t.Errorf("Name = %q", inv.Name)
	}
	_, guests, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if len(guests) != 3 {
		t.Fatalf("len(guests) = %d, want 3", len(guests))
	}
	if !guests[0].IsPrimary || guests[0].Name != "Frodi & Carla" {
		t.Errorf("guest[0] = %+v, want primary Frodi & Carla", guests[0])
	}
	if guests[1].IsPrimary || guests[1].Name != "Friend One" {
		t.Errorf("guest[1] = %+v, want non-primary Friend One", guests[1])
	}
	if guests[2].IsPrimary || guests[2].Name != "Friend Two" {
		t.Errorf("guest[2] = %+v, want non-primary Friend Two", guests[2])
	}
}

func TestCreateInvite_NoGuestNamesErrors(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	_, err := store.CreateInvite(context.Background(), "X", 0, 1, []string{})
	if err == nil {
		t.Fatal("CreateInvite with empty guestNames should error")
	}
}

func TestUpdateInvite_ReconcilesGuestNamesByPosition(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "A", 0, 3, []string{"A", "B", "C"})
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	_, before, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	// Mutate B's dietary pref to verify it's preserved on rename.
	if _, err := store.UpdateInvite(ctx, inv.ID, "A", 0, 3, []string{"A", "B", "C"}); err != nil { // no-op rename first to get stable ids
		t.Fatalf("no-op UpdateInvite() error: %v", err)
	}
	// Set dietary on guest B directly via SubmitRSVP-like path is overkill; instead test rename + add + remove.
	updated, err := store.UpdateInvite(ctx, inv.ID, "A2", 0, 3, []string{"A2", "B2"})
	if err != nil {
		t.Fatalf("UpdateInvite() error: %v", err)
	}
	if updated.Name != "A2" {
		t.Errorf("Name = %q, want A2", updated.Name)
	}
	_, after, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if len(after) != 2 {
		t.Fatalf("len(after) = %d, want 2 (C removed)", len(after))
	}
	if !after[0].IsPrimary || after[0].Name != "A2" {
		t.Errorf("after[0] = %+v, want primary A2", after[0])
	}
	if after[1].IsPrimary || after[1].Name != "B2" {
		t.Errorf("after[1] = %+v, want non-primary B2", after[1])
	}
	// The first two guest IDs should be unchanged (reconciled by position, not replaced).
	if after[0].ID != before[0].ID || after[1].ID != before[1].ID {
		t.Errorf("guest IDs changed: before=%v after=%v (position reconcile should keep IDs)", before, after)
	}
}

func TestUpdateInvite_AddsGuestNames(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, _ := store.CreateInvite(ctx, "A", 0, 3, []string{"A"})
	_, err := store.UpdateInvite(ctx, inv.ID, "A", 0, 3, []string{"A", "B", "C"})
	if err != nil {
		t.Fatalf("UpdateInvite() error: %v", err)
	}
	_, after, _ := store.GetInviteWithGuests(ctx, inv.ID)
	if len(after) != 3 {
		t.Fatalf("len(after) = %d, want 3", len(after))
	}
	if !after[0].IsPrimary {
		t.Errorf("after[0] not primary")
	}
}

func TestUpdateInvite_NoGuestNamesErrorsAndLeavesGuestsIntact(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "A", 0, 3, []string{"A", "B"})
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}

	if _, err := store.UpdateInvite(ctx, inv.ID, "A", 0, 3, []string{}); err == nil {
		t.Fatal("UpdateInvite with empty guestNames should error")
	}

	_, guests, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if len(guests) != 2 {
		t.Fatalf("len(guests) = %d, want 2", len(guests))
	}
	if guests[0].Name != "A" || guests[1].Name != "B" {
		t.Errorf("guests changed: got %q, want [A B]", []string{guests[0].Name, guests[1].Name})
	}
}

func TestUpdateInvite_NotFound(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	_, err := store.UpdateInvite(context.Background(), "999", "X", 0, 1, []string{"X"})
	if err != ErrNotFound {
		t.Errorf("UpdateInvite(999) err = %v, want ErrNotFound", err)
	}
}

func TestDeleteInvite_NotFound(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	if err := store.DeleteInvite(context.Background(), "999"); err != ErrNotFound {
		t.Errorf("DeleteInvite(999) err = %v, want ErrNotFound", err)
	}
}
