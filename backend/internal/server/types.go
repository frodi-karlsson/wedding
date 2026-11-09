package server

import "wedding/backend/internal/db"

// --- Requests ---

type GuestInput struct {
	Name              string `json:"name"`
	DietaryPreference string `json:"dietary_preference"`
	AlcoholFree       bool   `json:"alcohol_free"`
	IsPrimary         bool   `json:"is_primary"`
	CoPrimary         bool   `json:"co_primary"`
}

type RSVPRequest struct {
	Guests  []GuestInput `json:"guests"`
	Message string       `json:"message"`
}

type PresetGuestInput struct {
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name"`
	AlcoholFree bool   `json:"alcohol_free"`
}

type CreateInviteRequest struct {
	Name    string             `json:"name"`
	MinPlus int                `json:"min_plus"`
	MaxPlus int                `json:"max_plus"`
	Guests  []PresetGuestInput `json:"guests"`
	Group   bool               `json:"group"`
}

type UpdateInviteRequest struct {
	Name    string             `json:"name"`
	MinPlus int                `json:"min_plus"`
	MaxPlus int                `json:"max_plus"`
	Guests  []PresetGuestInput `json:"guests"`
	Group   bool               `json:"group"`
}

type CreateSpeechRequest struct {
	Speaker      string `json:"speaker"`
	Title        string `json:"title"`
	OriginalLang string `json:"original_lang"`
}

type UpdateSpeechRequest struct {
	Speaker      string `json:"speaker"`
	Title        string `json:"title"`
	OriginalLang string `json:"original_lang"`
	Version      int    `json:"version"`
}

type MoveSpeechRequest struct {
	Delta int `json:"delta"`
}

type SaveSpeechTextRequest struct {
	Body    string `json:"body"`
	Done    bool   `json:"done"`
	Version int    `json:"version"`
}

type LoginRequest struct {
	Password string `json:"password"`
}

// --- Generic responses ---

type StatusResponse struct {
	Status string `json:"status"`
}

type SessionResponse struct {
	Role string `json:"role"`
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
	CoPrimary         bool   `json:"co_primary"`
}

type InviteResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MinPlus   int    `json:"min_plus"`
	MaxPlus   int    `json:"max_plus"`
	Submitted bool   `json:"submitted"`
	Message   string `json:"message"`
}

type InviteWithGuestsResponse struct {
	Invite InviteResponse  `json:"invite"`
	Guests []GuestResponse `json:"guests"`
}

type SpeechTextResponse struct {
	Lang    string `json:"lang"`
	Body    string `json:"body"`
	Done    bool   `json:"done"`
	Version int    `json:"version"`
}

type SpeechResponse struct {
	ID           int64                `json:"id"`
	Position     int                  `json:"position"`
	Speaker      string               `json:"speaker"`
	Title        string               `json:"title"`
	OriginalLang string               `json:"original_lang"`
	Version      int                  `json:"version"`
	Texts        []SpeechTextResponse `json:"texts"`
}

type ListSpeechesResponse struct {
	Speeches []SpeechResponse `json:"speeches"`
}

type SpeechConflictResponse struct {
	Error   string         `json:"error"`
	Current SpeechResponse `json:"current"`
}

type SpeechTextConflictResponse struct {
	Error   string             `json:"error"`
	Current SpeechTextResponse `json:"current"`
}

type ListInvitesResponse struct {
	Invites []InviteResponse `json:"invites"`
}

func toPresetGuests(in []PresetGuestInput) []db.PresetGuest {
	out := make([]db.PresetGuest, len(in))
	for i, g := range in {
		out[i] = db.PresetGuest{ID: g.ID, Name: g.Name, AlcoholFree: g.AlcoholFree}
	}
	return out
}

// toGuestResponse converts a db.Guest to a GuestResponse.
func toGuestResponse(g *db.Guest) GuestResponse {
	return GuestResponse{
		ID:                g.ID,
		Name:              g.Name,
		DietaryPreference: g.DietaryPreference,
		AlcoholFree:       g.AlcoholFree,
		IsPrimary:         g.IsPrimary,
		CoPrimary:         g.CoPrimary,
	}
}

func toInviteResponse(inv *db.Invite) InviteResponse {
	return InviteResponse{
		ID:        inv.ID,
		Name:      inv.Name,
		MinPlus:   inv.MinPlus,
		MaxPlus:   inv.MaxPlus,
		Submitted: inv.Submitted,
		Message:   inv.Message,
	}
}
