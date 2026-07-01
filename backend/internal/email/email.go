package email

import (
	"context"

	"github.com/resend/resend-go/v3"
)

// Sender is the email-sending contract used by the invite service.
type Sender interface {
	Send(ctx context.Context, subject, body string) error
}

// Fake is a test double that records Send calls. Set Err to force a failure.
type Fake struct {
	Calls []FakeCall
	Err   error
}

type FakeCall struct {
	Subject string
	Body    string
}

func (f *Fake) Send(ctx context.Context, subject, body string) error {
	f.Calls = append(f.Calls, FakeCall{Subject: subject, Body: body})
	return f.Err
}

// Resend sends email via the Resend API.
type Resend struct {
	client *resend.Client
	from   string
	to     string
}

func NewResend(apiKey, from, to string) *Resend {
	return &Resend{
		client: resend.NewClient(apiKey),
		from:   from,
		to:     to,
	}
}

func (r *Resend) Send(ctx context.Context, subject, body string) error {
	req := &resend.SendEmailRequest{
		From:    r.from,
		To:      []string{r.to},
		Subject: subject,
		Text:    body,
	}
	_, err := r.client.Emails.Send(req)
	return err
}
