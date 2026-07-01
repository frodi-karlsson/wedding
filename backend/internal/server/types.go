package server

import "wedding/backend/internal/db"

// --- Requests ---

type GuestInput struct {
	Name              string `json:"name"`
	DietaryPreference string `json:"dietary_preference"`
	AlcoholFree       bool   `json:"alcohol_free"`
	IsPrimary         bool   `json:"is_primary"`
}

type RSVPRequest struct {
	Guests []GuestInput `json:"guests"`
}

type CreateInviteRequest struct {
	Name       string   `json:"name"`
	MinPlus    int      `json:"min_plus"`
	MaxPlus    int      `json:"max_plus"`
	GuestNames []string `json:"guest_names"`
}

type UpdateInviteRequest struct {
	Name       string   `json:"name"`
	MinPlus    int      `json:"min_plus"`
	MaxPlus    int      `json:"max_plus"`
	GuestNames []string `json:"guest_names"`
}

type LoginRequest struct {
	Password string `json:"password"`
}

// --- Generic responses ---

type StatusResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Responses ---

type GuestResponse struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	DietaryPreference string `json:"dietary_preference"`
	AlcoholFree       bool   `json:"alcohol_free"`
	IsPrimary         bool   `json:"is_primary"`
}

type InviteResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	MinPlus   int    `json:"min_plus"`
	MaxPlus   int    `json:"max_plus"`
	Submitted bool   `json:"submitted"`
}

type InviteWithGuestsResponse struct {
	Invite InviteResponse  `json:"invite"`
	Guests []GuestResponse `json:"guests"`
}

type ListInvitesResponse struct {
	Invites []InviteResponse `json:"invites"`
}

// toGuestResponse converts a db.Guest to a GuestResponse.
func toGuestResponse(g *db.Guest) GuestResponse {
	return GuestResponse{
		ID:                g.ID,
		Name:              g.Name,
		DietaryPreference: g.DietaryPreference,
		AlcoholFree:       g.AlcoholFree,
		IsPrimary:         g.IsPrimary,
	}
}

func toInviteResponse(inv *db.Invite) InviteResponse {
	return InviteResponse{
		ID:        inv.ID,
		Name:      inv.Name,
		MinPlus:   inv.MinPlus,
		MaxPlus:   inv.MaxPlus,
		Submitted: inv.Submitted,
	}
}
