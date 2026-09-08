package db

import (
	"strings"
	"testing"
)

func TestGenerateID_HasConfiguredLengthAndAlphabet(t *testing.T) {
	for range 200 {
		id := generateID()

		if len(id) != idLength {
			t.Fatalf("generateID() = %q, length %d, want %d", id, len(id), idLength)
		}
		for _, c := range id {
			if !strings.ContainsRune(idAlphabet, c) {
				t.Fatalf("generateID() = %q contains %q, which is outside the alphabet", id, c)
			}
		}
	}
}

func TestIDAlphabet_ExcludesReadAlikeCharacters(t *testing.T) {
	if len(idAlphabet) != 32 {
		t.Errorf("len(idAlphabet) = %d, want 32 so a random byte maps on without modulo bias", len(idAlphabet))
	}
	for _, c := range "ILOU" {
		if strings.ContainsRune(idAlphabet, c) {
			t.Errorf("idAlphabet contains %q, which reads alike with another character", c)
		}
	}
}

// Exact uniqueness is not a property of the generator (that is the birthday
// paradox, and CreateInvite handles it by retrying). What is worth pinning down
// is that the generator is actually drawing randomly rather than returning a
// constant or cycling through a handful of values. The threshold is slack enough
// that a healthy generator can never trip it: over 200 draws from a 33.5M space
// the expected number of duplicates is 0.0006.
func TestGenerateID_DrawsFromTheFullSpace(t *testing.T) {
	const draws = 200
	const minDistinct = 195

	seen := make(map[string]bool, draws)
	for range draws {
		seen[generateID()] = true
	}

	if len(seen) < minDistinct {
		t.Errorf("generateID() produced %d distinct ids over %d draws, want at least %d", len(seen), draws, minDistinct)
	}
}

func TestCanonicalID_FoldsCaseAndReadAlikeCharacters(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"lowercase", "abcde", "ABCDE"},
		{"mixed case", "AbCdE", "ABCDE"},
		{"lowercase l reads as one", "l2345", "12345"},
		{"uppercase I reads as one", "I2345", "12345"},
		{"lowercase o reads as zero", "o2345", "02345"},
		{"already canonical", "0Z9HK", "0Z9HK"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := CanonicalID(tt.raw)

			if !ok {
				t.Fatalf("CanonicalID(%q) reported the id as malformed", tt.raw)
			}
			if got != tt.want {
				t.Errorf("CanonicalID(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestCanonicalID_AcceptsLegacyHexTokens(t *testing.T) {
	legacy := strings.Repeat("a1", 16)

	got, ok := CanonicalID(legacy)

	if !ok {
		t.Fatalf("CanonicalID(%q) rejected a legacy hex token", legacy)
	}
	if got != legacy {
		t.Errorf("CanonicalID(%q) = %q, want it unchanged", legacy, got)
	}
}

func TestCanonicalID_RejectsMalformedIDs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"empty", ""},
		{"too short", "ABCD"},
		{"too long but not legacy", "ABCDEF"},
		{"U is not in the alphabet", "ABCDU"},
		{"punctuation", "AB-DE"},
		{"multi-byte runes", "Åbcd"},
		{"legacy length but not hex", strings.Repeat("z", 32)},
		{"sql injection attempt", "' OR 1=1 --"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := CanonicalID(tt.raw); ok {
				t.Errorf("CanonicalID(%q) = %q, true; want it rejected", tt.raw, got)
			}
		})
	}
}

func TestCanonicalID_RoundTripsGeneratedIDs(t *testing.T) {
	for range 200 {
		id := generateID()

		got, ok := CanonicalID(id)

		if !ok {
			t.Fatalf("CanonicalID(%q) rejected a freshly generated id", id)
		}
		if got != id {
			t.Errorf("CanonicalID(%q) = %q, want a generated id to already be canonical", id, got)
		}
	}
}
