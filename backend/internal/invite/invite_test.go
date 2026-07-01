package invite

import (
	"context"
	"errors"
	"testing"

	"wedding/backend/internal/db"
)

// fakeEmailer records the last email it was asked to send.
type fakeEmailer struct {
	lastSubject string
	lastBody    string
	sendErr     error
}

func (f *fakeEmailer) Send(ctx context.Context, subject, body string) error {
	f.lastSubject = subject
	f.lastBody = body
	return f.sendErr
}

func newTestService(t *testing.T) (*Service, *fakeEmailer, func()) {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if err := db.Migrate(d); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	store := db.NewSQLiteStore(d)
	emailer := &fakeEmailer{}
	svc := NewService(store, emailer)
	return svc, emailer, func() { d.Close() }
}

func TestSubmitRSVP_Valid(t *testing.T) {
	svc, emailer, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	inv, err := svc.CreateInvite(ctx, "Frodi", 0, 2)
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}

	guests := []db.Guest{
		{Name: "Frodi", IsPrimary: true},
		{Name: "Carla", DietaryPreference: "vegetarian", AlcoholFree: true},
	}
	savedInv, savedGuests, err := svc.SubmitRSVP(ctx, inv.ID, guests)
	if err != nil {
		t.Fatalf("SubmitRSVP() error: %v", err)
	}
	if !savedInv.Submitted {
		t.Errorf("Submitted = false, want true")
	}
	if len(savedGuests) != 2 {
		t.Errorf("len(savedGuests) = %d, want 2", len(savedGuests))
	}
	if emailer.lastSubject == "" {
		t.Errorf("email not sent: subject empty")
	}
}

func TestSubmitRSVP_NoPrimary(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	inv, _ := svc.CreateInvite(ctx, "Frodi", 0, 2)
	guests := []db.Guest{
		{Name: "Frodi"},
	}
	_, _, err := svc.SubmitRSVP(ctx, inv.ID, guests)
	if err == nil {
		t.Fatal("SubmitRSVP should error when no primary guest")
	}
}

func TestSubmitRSVP_TooManyPluses(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	inv, _ := svc.CreateInvite(ctx, "Frodi", 0, 1)
	guests := []db.Guest{
		{Name: "Frodi", IsPrimary: true},
		{Name: "Plus1"},
		{Name: "Plus2"},
	}
	_, _, err := svc.SubmitRSVP(ctx, inv.ID, guests)
	if err == nil {
		t.Fatal("SubmitRSVP should error when pluses exceed max_plus")
	}
}

func TestSubmitRSVP_TooFewPluses(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	inv, _ := svc.CreateInvite(ctx, "Frodi", 2, 2)
	guests := []db.Guest{
		{Name: "Frodi", IsPrimary: true},
	}
	_, _, err := svc.SubmitRSVP(ctx, inv.ID, guests)
	if err == nil {
		t.Fatal("SubmitRSVP should error when pluses below min_plus")
	}
}

func TestSubmitRSVP_EmptyName(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	inv, _ := svc.CreateInvite(ctx, "Frodi", 0, 2)
	guests := []db.Guest{
		{Name: "Frodi", IsPrimary: true},
		{Name: ""},
	}
	_, _, err := svc.SubmitRSVP(ctx, inv.ID, guests)
	if err == nil {
		t.Fatal("SubmitRSVP should error when a guest name is empty")
	}
}

func TestSubmitRSVP_InviteNotFound(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	guests := []db.Guest{{Name: "Frodi", IsPrimary: true}}
	_, _, err := svc.SubmitRSVP(context.Background(), 999, guests)
	if err == nil {
		t.Fatal("SubmitRSVP should error when invite not found")
	}
}

func TestSubmitRSVP_EmailSendFailure_RollsBack(t *testing.T) {
	svc, emailer, cleanup := newTestService(t)
	defer cleanup()
	emailer.sendErr = errors.New("send failed")
	ctx := context.Background()

	inv, _ := svc.CreateInvite(ctx, "Frodi", 0, 2)
	guests := []db.Guest{{Name: "Frodi", IsPrimary: true}}

	_, _, err := svc.SubmitRSVP(ctx, inv.ID, guests)
	if err == nil {
		t.Fatal("SubmitRSVP should error when email send fails")
	}

	// Verify nothing was persisted: invite should still be unsubmitted and
	// only the original primary guest should remain.
	inv2, guests2, err := svc.GetInvite(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInvite() error: %v", err)
	}
	if inv2.Submitted {
		t.Errorf("Submitted = true, want false (rollback)")
	}
	if len(guests2) != 1 {
		t.Errorf("len(guests2) = %d, want 1 (rollback kept primary)", len(guests2))
	}
}
