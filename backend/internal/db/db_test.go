package db

import (
	"context"
	"errors"
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
	inv, err := store.CreateInvite(ctx, "Frodi & Carla", 0, 2, presets("Frodi & Carla"), false)
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

func TestCreateInvite_GeneratesShortOpaqueID(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "Frodi & Carla", 0, 2, presets("Frodi & Carla", "Friend"), false)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	if len(inv.ID) != idLength {
		t.Errorf("ID %q length = %d, want %d", inv.ID, len(inv.ID), idLength)
	}
	if canonical, ok := CanonicalID(inv.ID); !ok || canonical != inv.ID {
		t.Errorf("ID %q is not a canonical short id", inv.ID)
	}
}

func TestCreateInvite_RetriesWhenTheGeneratedIDCollides(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	// The second invite draws the id the first one already took, then succeeds
	// on its retry.
	draws := []string{"AAAAA", "AAAAA", "BBBBB"}
	store.newID = func() string {
		id := draws[0]
		draws = draws[1:]
		return id
	}

	first, err := store.CreateInvite(ctx, "First", 0, 0, presets("First"), false)
	if err != nil {
		t.Fatalf("CreateInvite() first error: %v", err)
	}
	second, err := store.CreateInvite(ctx, "Second", 0, 0, presets("Second"), false)
	if err != nil {
		t.Fatalf("CreateInvite() second error: %v", err)
	}

	if first.ID != "AAAAA" {
		t.Errorf("first ID = %q, want AAAAA", first.ID)
	}
	if second.ID != "BBBBB" {
		t.Errorf("second ID = %q, want BBBBB after retrying past the collision", second.ID)
	}
	if len(draws) != 0 {
		t.Errorf("%d id draws left unused, want the collision to have consumed one", len(draws))
	}
}

func TestCreateInvite_RollsBackGuestsFromACollidingAttempt(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	draws := []string{"AAAAA", "AAAAA", "BBBBB"}
	store.newID = func() string {
		id := draws[0]
		draws = draws[1:]
		return id
	}

	if _, err := store.CreateInvite(ctx, "First", 0, 0, presets("First"), false); err != nil {
		t.Fatalf("CreateInvite() first error: %v", err)
	}
	if _, err := store.CreateInvite(ctx, "Second", 0, 0, presets("Second"), false); err != nil {
		t.Fatalf("CreateInvite() second error: %v", err)
	}

	// The abandoned attempt inserted a guest row before the invite insert
	// failed. It must not survive as an orphan attached to the first invite.
	_, guests, err := store.GetInviteWithGuests(ctx, "AAAAA")
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if len(guests) != 1 {
		t.Errorf("invite AAAAA has %d guests, want 1", len(guests))
	}
	if len(guests) > 0 && guests[0].Name != "First" {
		t.Errorf("invite AAAAA guest = %q, want First", guests[0].Name)
	}
}

func TestCreateInvite_FailsAfterExhaustingIDAttempts(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	store.newID = func() string { return "AAAAA" }

	if _, err := store.CreateInvite(ctx, "First", 0, 0, presets("First"), false); err != nil {
		t.Fatalf("CreateInvite() first error: %v", err)
	}
	_, err := store.CreateInvite(ctx, "Second", 0, 0, presets("Second"), false)

	if !errors.Is(err, ErrIDCollision) {
		t.Errorf("CreateInvite() err = %v, want ErrIDCollision", err)
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

	inv, err := store.CreateInvite(ctx, "Frodi & Carla", 0, 2, presets("Frodi & Carla"), false)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}

	guests := []Guest{
		{Name: "Frodi", IsPrimary: true},
		{Name: "Carla", DietaryPreference: "vegetarian"},
	}
	savedInv, saved, err := store.SubmitRSVP(ctx, inv.ID, guests, true, "")
	if err != nil {
		t.Fatalf("SubmitRSVP() error: %v", err)
	}
	if len(saved) != 2 {
		t.Errorf("len(saved) = %d, want 2", len(saved))
	}
	// The returned invite must reflect the post-write state (submitted) directly,
	// without a second read.
	if !savedInv.Submitted {
		t.Errorf("returned invite Submitted = false, want true")
	}
	if !saved[0].IsPrimary || saved[0].Name != "Frodi" {
		t.Errorf("returned primary guest = %+v, want Frodi primary", saved[0])
	}
	if saved[1].Name != "Carla" {
		t.Errorf("returned second guest Name = %q, want Carla", saved[1].Name)
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

	inv, err := store.CreateInvite(ctx, "Old Name", 0, 1, presets("Old Name"), false)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}

	_, before, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if _, err := store.UpdateInvite(ctx, inv.ID, "New Name", 0, 1, []PresetGuest{{ID: before[0].ID, Name: "New Name"}}, false); err != nil {
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

	inv, err := store.CreateInvite(ctx, "Cascade", 0, 1, presets("Cascade"), false)
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
	inv, err := store.CreateInvite(ctx, "To Delete", 0, 1, presets("To Delete"), false)
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

	inv, err := store.CreateInvite(ctx, "Frodi & Carla", 0, 2, presets("Frodi & Carla", "Friend One", "Friend Two"), false)
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

func TestCreateInvite_Group_AllCoPrimary(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "ignored", 5, 9, presets("Alice", "Bob", "Carla"), true)
	if err != nil {
		t.Fatalf("CreateInvite(group) error: %v", err)
	}
	_, guests, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if len(guests) != 3 {
		t.Fatalf("len(guests) = %d, want 3", len(guests))
	}
	for i, g := range guests {
		if g.IsPrimary {
			t.Errorf("guest[%d] IsPrimary = true, want false (group)", i)
		}
		if !g.CoPrimary {
			t.Errorf("guest[%d] CoPrimary = false, want true (group)", i)
		}
	}
}

func TestCreateInvite_NoGuestNamesErrors(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	_, err := store.CreateInvite(context.Background(), "X", 0, 1, presets(), false)
	if err == nil {
		t.Fatal("CreateInvite with no presets should error")
	}
}

// submittedGuests puts an invite through an RSVP so its guests carry dietary
// preferences, and returns the rows as stored (with their fresh ids).
func submittedGuests(t *testing.T, store *SQLiteStore, inviteID string, guests []Guest) []Guest {
	t.Helper()
	_, saved, err := store.SubmitRSVP(context.Background(), inviteID, guests, true, "")
	if err != nil {
		t.Fatalf("SubmitRSVP() error: %v", err)
	}
	return saved
}

func TestUpdateInvite_ReconcilesGuestsByID(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()
	inv, err := store.CreateInvite(ctx, "A", 0, 3, presets("A", "B", "C"), false)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	before := submittedGuests(t, store, inv.ID, []Guest{
		{Name: "A", IsPrimary: true},
		{Name: "B", DietaryPreference: "vegan"},
		{Name: "C", DietaryPreference: "no nuts"},
	})

	updated, err := store.UpdateInvite(ctx, inv.ID, "A2", 0, 3, []PresetGuest{
		{ID: before[0].ID, Name: "A2"},
		{ID: before[2].ID, Name: "C2"},
		{Name: "D"},
	}, false)
	if err != nil {
		t.Fatalf("UpdateInvite() error: %v", err)
	}

	if updated.Name != "A2" {
		t.Errorf("Name = %q, want A2", updated.Name)
	}
	_, after, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if len(after) != 3 {
		t.Fatalf("len(after) = %d, want 3 (B removed, D added)", len(after))
	}
	if after[0].ID != before[0].ID || !after[0].IsPrimary || after[0].Name != "A2" {
		t.Errorf("after[0] = %+v, want primary A2 keeping id %d", after[0], before[0].ID)
	}
	if after[1].ID != before[2].ID || after[1].Name != "C2" || after[1].DietaryPreference != "no nuts" {
		t.Errorf("after[1] = %+v, want C2 keeping id %d and dietary %q", after[1], before[2].ID, "no nuts")
	}
	if after[2].Name != "D" || after[2].DietaryPreference != "" || after[2].IsPrimary {
		t.Errorf("after[2] = %+v, want new plus D with no dietary preference", after[2])
	}
}

func TestUpdateInvite_UnknownGuestIDErrorsAndLeavesGuestsIntact(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()
	inv, err := store.CreateInvite(ctx, "A", 0, 3, presets("A", "B"), false)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	other, err := store.CreateInvite(ctx, "Other", 0, 0, presets("Other"), false)
	if err != nil {
		t.Fatalf("CreateInvite(other) error: %v", err)
	}
	_, otherGuests, err := store.GetInviteWithGuests(ctx, other.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests(other) error: %v", err)
	}
	_, before, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}

	cases := map[string][]PresetGuest{
		"stale id":          {{ID: before[0].ID, Name: "A"}, {ID: 9999, Name: "B"}},
		"other invite's id": {{ID: before[0].ID, Name: "A"}, {ID: otherGuests[0].ID, Name: "B"}},
		"same id twice":     {{ID: before[0].ID, Name: "A"}, {ID: before[0].ID, Name: "B"}},
	}
	for name, guests := range cases {
		_, err := store.UpdateInvite(ctx, inv.ID, "A", 0, 3, guests, false)

		if !errors.Is(err, ErrGuestNotOnInvite) {
			t.Errorf("%s: err = %v, want ErrGuestNotOnInvite", name, err)
		}
	}
	_, after, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if len(after) != 2 || after[0].ID != before[0].ID || after[1].ID != before[1].ID {
		t.Errorf("guests changed: before=%+v after=%+v", before, after)
	}
}

func TestUpdateInvite_AddsGuestNames(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, _ := store.CreateInvite(ctx, "A", 0, 3, presets("A"), false)
	_, err := store.UpdateInvite(ctx, inv.ID, "A", 0, 3, presets("A", "B", "C"), false)
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

	inv, err := store.CreateInvite(ctx, "A", 0, 3, presets("A", "B"), false)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}

	if _, err := store.UpdateInvite(ctx, inv.ID, "A", 0, 3, presets(), false); err == nil {
		t.Fatal("UpdateInvite with no presets should error")
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
	_, err := store.UpdateInvite(context.Background(), "999", "X", 0, 1, presets("X"), false)
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

func presets(names ...string) []PresetGuest {
	out := make([]PresetGuest, len(names))
	for i, n := range names {
		out[i] = PresetGuest{Name: n}
	}
	return out
}

func TestCreateInvite_StoresPresetAlcoholFree(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "Parent", 0, 2, []PresetGuest{
		{Name: "Parent"},
		{Name: "Kid", AlcoholFree: true},
	}, false)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	_, guests, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if guests[0].AlcoholFree {
		t.Errorf("guest[0] AlcoholFree = true, want false")
	}
	if !guests[1].AlcoholFree {
		t.Errorf("guest[1] AlcoholFree = false, want true")
	}
}

func TestUpdateInvite_SetsPresetAlcoholFree(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := store.CreateInvite(ctx, "", 0, 0, []PresetGuest{
		{Name: "Adult", AlcoholFree: true},
		{Name: "Kid"},
	}, true)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	_, before, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	if _, err := store.UpdateInvite(ctx, inv.ID, "", 0, 0, []PresetGuest{
		{ID: before[0].ID, Name: "Adult"},
		{ID: before[1].ID, Name: "Kid", AlcoholFree: true},
		{Name: "Baby", AlcoholFree: true},
	}, true); err != nil {
		t.Fatalf("UpdateInvite() error: %v", err)
	}
	_, guests, err := store.GetInviteWithGuests(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInviteWithGuests() error: %v", err)
	}
	got := []bool{guests[0].AlcoholFree, guests[1].AlcoholFree, guests[2].AlcoholFree}
	if got[0] || !got[1] || !got[2] {
		t.Errorf("AlcoholFree = %v, want [false true true]", got)
	}
}
