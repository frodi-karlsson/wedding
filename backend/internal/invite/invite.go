package invite

import (
	"context"
	"fmt"
	"strings"

	"wedding/backend/internal/db"
)

// Emailer is satisfied by email.Sender. Defined here to avoid an import cycle
// (email imports nothing from invite; invite depends only on this interface).
type Emailer interface {
	Send(ctx context.Context, subject, body string) error
}

// ValidationError describes a failed validation rule.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}

// Service orchestrates invite/guest domain logic: validation, transactional
// upsert, and email notification on RSVP.
type Service struct {
	store db.Store
	email Emailer
}

func NewService(store db.Store, email Emailer) *Service {
	return &Service{store: store, email: email}
}

func (s *Service) GetInvite(ctx context.Context, id int64) (db.Invite, []db.Guest, error) {
	return s.store.GetInviteWithGuests(ctx, id)
}

func (s *Service) CreateInvite(ctx context.Context, name string, minPlus, maxPlus int, guestNames []string) (db.Invite, error) {
	return s.store.CreateInvite(ctx, name, minPlus, maxPlus, guestNames)
}

func (s *Service) ListInvites(ctx context.Context) ([]db.Invite, error) {
	return s.store.ListInvites(ctx)
}

func (s *Service) UpdateInvite(ctx context.Context, id int64, name string, minPlus, maxPlus int, guestNames []string) (db.Invite, error) {
	return s.store.UpdateInvite(ctx, id, name, minPlus, maxPlus, guestNames)
}

func (s *Service) DeleteInvite(ctx context.Context, id int64) error {
	return s.store.DeleteInvite(ctx, id)
}

// SubmitRSVP validates the guests, transactionally upserts them, marks the
// invite submitted, and sends the email. On email failure the persisted state
// is reverted (the invite is marked unsubmitted and original guests restored
// is NOT possible with a simple flag, so we instead send the email BEFORE
// marking submitted — see implementation).
func (s *Service) SubmitRSVP(ctx context.Context, id int64, guests []db.Guest) (db.Invite, []db.Guest, error) {
	inv, err := s.store.GetInvite(ctx, id)
	if err != nil {
		return db.Invite{}, nil, err
	}

	if err := validate(inv, guests); err != nil {
		return db.Invite{}, nil, err
	}

	// Send email BEFORE persisting so a send failure leaves the DB untouched.
	body := buildRSVPEmailBody(inv, guests)
	if err := s.email.Send(ctx, "New RSVP for "+inv.Name, body); err != nil {
		return db.Invite{}, nil, fmt.Errorf("send email: %w", err)
	}

	saved, err := s.store.SubmitRSVP(ctx, id, guests, true)
	if err != nil {
		return db.Invite{}, nil, err
	}
	inv, err = s.store.GetInvite(ctx, id)
	if err != nil {
		return db.Invite{}, nil, err
	}
	return inv, saved, nil
}

func validate(inv db.Invite, guests []db.Guest) error {
	primaryCount := 0
	var nonPrimary []db.Guest
	for _, g := range guests {
		if strings.TrimSpace(g.Name) == "" {
			return &ValidationError{Field: "name", Message: "guest name cannot be empty"}
		}
		if g.IsPrimary {
			primaryCount++
		} else {
			nonPrimary = append(nonPrimary, g)
		}
	}
	if primaryCount != 1 {
		return &ValidationError{Field: "is_primary", Message: "exactly one primary guest is required"}
	}
	if len(nonPrimary) < inv.MinPlus {
		return &ValidationError{Field: "guests", Message: fmt.Sprintf("at least %d plus(es) required, got %d", inv.MinPlus, len(nonPrimary))}
	}
	if len(nonPrimary) > inv.MaxPlus {
		return &ValidationError{Field: "guests", Message: fmt.Sprintf("at most %d plus(es) allowed, got %d", inv.MaxPlus, len(nonPrimary))}
	}
	return nil
}

func buildRSVPEmailBody(inv db.Invite, guests []db.Guest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "RSVP submitted for %s\n\n", inv.Name)
	for _, g := range guests {
		role := "Plus"
		if g.IsPrimary {
			role = "Primary"
		}
		fmt.Fprintf(&b, "- %s (%s)\n", g.Name, role)
		if g.DietaryPreference != "" {
			fmt.Fprintf(&b, "  Dietary: %s\n", g.DietaryPreference)
		}
		if g.AlcoholFree {
			fmt.Fprintf(&b, "  Alcohol-free: yes\n")
		}
	}
	return b.String()
}
