package email

import (
	"context"
	"errors"
	"testing"
)

func TestFake_RecordsCalls(t *testing.T) {
	f := &Fake{}
	if err := f.Send(context.Background(), "Subject", "Body"); err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if len(f.Calls) != 1 {
		t.Fatalf("len(Calls) = %d, want 1", len(f.Calls))
	}
	if f.Calls[0].Subject != "Subject" || f.Calls[0].Body != "Body" {
		t.Errorf("Call = %+v", f.Calls[0])
	}
}

func TestFake_ReturnsError(t *testing.T) {
	f := &Fake{Err: errors.New("boom")}
	if err := f.Send(context.Background(), "s", "b"); err == nil {
		t.Fatal("Send() should return configured error")
	}
}

func TestNewResend_DoesNotPanic(t *testing.T) {
	// We can't hit the real API in tests; just verify construction.
	r := NewResend("re_dummy_key", "rsvp@example.com", "to@example.com")
	if r == nil {
		t.Fatal("NewResend returned nil")
	}
}
