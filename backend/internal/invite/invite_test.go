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

	inv, err := svc.CreateInvite(ctx, "Frodi", 0, 2, []string{"Frodi"})
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}

	guests := []db.Guest{
		{Name: "Frodi", IsPrimary: true},
		{Name: "Carla", DietaryPreference: "vegetarian", AlcoholFree: true},
	}
	savedInv, savedGuests, err := svc.SubmitRSVP(ctx, inv.ID, guests, "")
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

	inv, _ := svc.CreateInvite(ctx, "Frodi", 0, 2, []string{"Frodi"})
	guests := []db.Guest{
		{Name: "Frodi"},
	}
	_, _, err := svc.SubmitRSVP(ctx, inv.ID, guests, "")
	if err == nil {
		t.Fatal("SubmitRSVP should error when no primary guest")
	}
}

func TestSubmitRSVP_TooManyPluses(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	inv, _ := svc.CreateInvite(ctx, "Frodi", 0, 1, []string{"Frodi"})
	guests := []db.Guest{
		{Name: "Frodi", IsPrimary: true},
		{Name: "Plus1"},
		{Name: "Plus2"},
	}
	_, _, err := svc.SubmitRSVP(ctx, inv.ID, guests, "")
	if err == nil {
		t.Fatal("SubmitRSVP should error when pluses exceed max_plus")
	}
}

func TestSubmitRSVP_TooFewPluses(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	inv, _ := svc.CreateInvite(ctx, "Frodi", 2, 2, []string{"Frodi"})
	guests := []db.Guest{
		{Name: "Frodi", IsPrimary: true},
	}
	_, _, err := svc.SubmitRSVP(ctx, inv.ID, guests, "")
	if err == nil {
		t.Fatal("SubmitRSVP should error when pluses below min_plus")
	}
}

func TestSubmitRSVP_EmptyName(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	inv, _ := svc.CreateInvite(ctx, "Frodi", 0, 2, []string{"Frodi"})
	guests := []db.Guest{
		{Name: "Frodi", IsPrimary: true},
		{Name: ""},
	}
	_, _, err := svc.SubmitRSVP(ctx, inv.ID, guests, "")
	if err == nil {
		t.Fatal("SubmitRSVP should error when a guest name is empty")
	}
}

func TestSubmitRSVP_InviteNotFound(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	guests := []db.Guest{{Name: "Frodi", IsPrimary: true}}
	_, _, err := svc.SubmitRSVP(context.Background(), "nonexistent", guests, "")
	if err == nil {
		t.Fatal("SubmitRSVP should error when invite not found")
	}
}

func TestCreateInvite_MultipleGuests(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	inv, err := svc.CreateInvite(ctx, "Frodi", 0, 2, []string{"Frodi", "Carla"})
	if err != nil {
		t.Fatalf("CreateInvite() error: %v", err)
	}
	_, guests, err := svc.GetInvite(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInvite() error: %v", err)
	}
	if len(guests) != 2 {
		t.Fatalf("len(guests) = %d, want 2", len(guests))
	}
	if !guests[0].IsPrimary {
		t.Errorf("guests[0] not primary")
	}
}

func TestSubmitRSVP_EmailSendFailure_StillPersists(t *testing.T) {
	svc, emailer, cleanup := newTestService(t)
	defer cleanup()
	emailer.sendErr = errors.New("send failed")
	ctx := context.Background()

	inv, _ := svc.CreateInvite(ctx, "Frodi", 0, 2, []string{"Frodi"})
	guests := []db.Guest{{Name: "Frodi", IsPrimary: true}}

	savedInv, _, err := svc.SubmitRSVP(ctx, inv.ID, guests, "hi")
	if err != nil {
		t.Fatalf("SubmitRSVP should succeed even when email fails: %v", err)
	}
	if !savedInv.Submitted {
		t.Errorf("Submitted = false, want true (persisted despite email failure)")
	}

	// The RSVP must be persisted even though the notification email failed —
	// email is best-effort, never a gate on the guest's response.
	inv2, guests2, err := svc.GetInvite(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInvite() error: %v", err)
	}
	if !inv2.Submitted {
		t.Errorf("Submitted = false, want true (persisted)")
	}
	if inv2.Message != "hi" {
		t.Errorf("Message = %q, want %q", inv2.Message, "hi")
	}
	if len(guests2) != 1 {
		t.Errorf("len(guests2) = %d, want 1", len(guests2))
	}
	// The email should still have been attempted.
	if emailer.lastSubject == "" {
		t.Error("expected a notification email send attempt")
	}
}
