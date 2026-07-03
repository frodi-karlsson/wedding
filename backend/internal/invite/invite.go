package invite

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"wedding/backend/internal/db"
)

// emailSendTimeout bounds the best-effort RSVP notification send, which runs on
// a context detached from the request so a client disconnect can't cancel it.
const emailSendTimeout = 10 * time.Second

// Emailer is satisfied by email.Sender. Defined here to avoid an import cycle
// (email imports nothing from invite; invite depends only on this interface).
type Emailer interface {
	Send(ctx context.Context, subject, body string) error
}

// fieldGuests is the ValidationError field name for guest-list rules.
const fieldGuests = "guests"

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

func (s *Service) GetInvite(ctx context.Context, id string) (db.Invite, []db.Guest, error) {
	return s.store.GetInviteWithGuests(ctx, id)
}

func (s *Service) CreateInvite(ctx context.Context, name string, minPlus, maxPlus int, guestNames []string, group bool) (db.Invite, error) {
	name, minPlus, maxPlus = normalizeInviteShape(name, minPlus, maxPlus, guestNames, group)
	return s.store.CreateInvite(ctx, name, minPlus, maxPlus, guestNames, group)
}

func (s *Service) ListInvites(ctx context.Context) ([]db.Invite, error) {
	return s.store.ListInvites(ctx)
}

func (s *Service) UpdateInvite(ctx context.Context, id, name string, minPlus, maxPlus int, guestNames []string, group bool) (db.Invite, error) {
	name, minPlus, maxPlus = normalizeInviteShape(name, minPlus, maxPlus, guestNames, group)
	return s.store.UpdateInvite(ctx, id, name, minPlus, maxPlus, guestNames, group)
}

// normalizeInviteShape enforces the group-invite invariants at the service
// boundary: a group invite has no single primary, no plus allowance, and its
// display name is derived from the co-primary names joined together.
func normalizeInviteShape(name string, minPlus, maxPlus int, guestNames []string, group bool) (outName string, outMin, outMax int) {
	if group {
		return joinNames(guestNames), 0, 0
	}
	return name, minPlus, maxPlus
}

// joinNames renders a list of names as "A", "A & B", or "A, B & C".
func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " & " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + " & " + names[len(names)-1]
	}
}

func (s *Service) DeleteInvite(ctx context.Context, id string) error {
	return s.store.DeleteInvite(ctx, id)
}

// SubmitRSVP validates the guests, transactionally upserts them, and marks the
// invite submitted. The notification email is best-effort: the RSVP is persisted
// first so a mail outage can never lose or block a guest's response. A send
// failure is logged, and the admin dashboard remains the source of truth.
func (s *Service) SubmitRSVP(ctx context.Context, id string, guests []db.Guest, message string) (db.Invite, []db.Guest, error) {
	inv, err := s.store.GetInvite(ctx, id)
	if err != nil {
		return db.Invite{}, nil, err
	}

	if err := validate(&inv, guests); err != nil {
		return db.Invite{}, nil, err
	}

	savedInv, savedGuests, err := s.store.SubmitRSVP(ctx, id, guests, true, message)
	if err != nil {
		return db.Invite{}, nil, err
	}

	// Best-effort notification, never fail the RSVP because email failed. The
	// RSVP is already persisted above. WithoutCancel keeps the request's values
	// but detaches from its cancellation, so a client disconnect right after the
	// DB commit cannot cancel the send. The timeout still bounds it.
	subject := "New RSVP for " + savedInv.Name
	body := buildRSVPEmailBody(&savedInv, savedGuests, message)
	emailCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), emailSendTimeout)
	defer cancel()
	if err := s.email.Send(emailCtx, subject, body); err != nil {
		log.Printf("rsvp notification email failed for invite %s: %v", id, err)
	}

	return savedInv, savedGuests, nil
}

func validate(inv *db.Invite, guests []db.Guest) error {
	primaryCount := 0
	coPrimaryCount := 0
	plusCount := 0
	for _, g := range guests {
		if strings.TrimSpace(g.Name) == "" {
			return &ValidationError{Field: "name", Message: "guest name cannot be empty"}
		}
		switch {
		case g.IsPrimary && g.CoPrimary:
			return &ValidationError{Field: fieldGuests, Message: "a guest cannot be both primary and co-primary"}
		case g.IsPrimary:
			primaryCount++
		case g.CoPrimary:
			coPrimaryCount++
		default:
			plusCount++
		}
	}

	// Group invite: all guests are co-primaries — no single primary and no plain
	// pluses. At least one must remain.
	if coPrimaryCount > 0 {
		if primaryCount > 0 || plusCount > 0 {
			return &ValidationError{Field: fieldGuests, Message: "cannot mix co-primary guests with a primary or plus guests"}
		}
		return nil
	}

	// Standard invite: exactly one primary + pluses within [min, max].
	if primaryCount != 1 {
		return &ValidationError{Field: "is_primary", Message: "exactly one primary guest is required"}
	}
	if plusCount < inv.MinPlus {
		return &ValidationError{Field: fieldGuests, Message: fmt.Sprintf("at least %d plus(es) required, got %d", inv.MinPlus, plusCount)}
	}
	if plusCount > inv.MaxPlus {
		return &ValidationError{Field: fieldGuests, Message: fmt.Sprintf("at most %d plus(es) allowed, got %d", inv.MaxPlus, plusCount)}
	}
	return nil
}

func buildRSVPEmailBody(inv *db.Invite, guests []db.Guest, message string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "RSVP submitted for %s\n\n", inv.Name)
	for _, g := range guests {
		role := "Plus"
		switch {
		case g.IsPrimary:
			role = "Primary"
		case g.CoPrimary:
			role = "Co-primary"
		}
		fmt.Fprintf(&b, "- %s (%s)\n", g.Name, role)
		if g.DietaryPreference != "" {
			fmt.Fprintf(&b, "  Dietary: %s\n", g.DietaryPreference)
		}
		if g.AlcoholFree {
			fmt.Fprintf(&b, "  Alcohol-free: yes\n")
		}
	}
	if message != "" {
		fmt.Fprintf(&b, "\nMessage: %s\n", message)
	}
	return b.String()
}
