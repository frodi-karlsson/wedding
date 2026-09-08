package db

import (
	"crypto/rand"
	"strings"
)

// idAlphabet is Crockford base32: the ten digits plus the twenty-two letters
// left after dropping I, L, O and U. Two properties earn it the job. Dropping
// the read-alike letters means a five-character id survives being read aloud or
// typed by hand off a printed card, and 256 is an exact multiple of 32, so a
// random byte maps onto the alphabet with no modulo bias and therefore no
// rejection sampling.
const idAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// idLength is the character count of a generated invite id. Five Crockford
// characters span 32^5 = 33,554,432 ids.
const idLength = 5

// legacyIDLength is the length of the 32-character hex tokens issued before ids
// were shortened. Invite links already handed out contain them, so lookups
// still accept them.
const legacyIDLength = 32

// generateID returns a fresh random invite id in canonical form.
func generateID() string {
	b := make([]byte, idLength)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	id := make([]byte, idLength)
	for i, v := range b {
		id[i] = idAlphabet[int(v)%len(idAlphabet)]
	}
	return string(id)
}

// CanonicalID folds a caller-supplied invite id into the form stored in the
// database and reports whether it is well-formed at all. Short ids are matched
// case-insensitively and with Crockford's read-alike folding (I and L stand in
// for 1, O for 0) so an id copied off a card by hand still resolves. Legacy hex
// tokens pass through unchanged. Anything else is rejected here rather than
// becoming a pointless database lookup.
func CanonicalID(raw string) (string, bool) {
	switch len(raw) {
	case idLength:
		return foldShortID(raw)
	case legacyIDLength:
		return foldLegacyID(raw)
	default:
		return "", false
	}
}

// foldShortID canonicalizes a candidate short id. It walks bytes rather than
// runes so that a multi-byte character can never fold down to a shorter id that
// then passes the length check.
func foldShortID(raw string) (string, bool) {
	id := make([]byte, idLength)
	for i := range idLength {
		c := raw[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		switch c {
		case 'I', 'L':
			c = '1'
		case 'O':
			c = '0'
		}
		if strings.IndexByte(idAlphabet, c) < 0 {
			return "", false
		}
		id[i] = c
	}
	return string(id), true
}

// foldLegacyID canonicalizes a candidate legacy hex token to lowercase, the
// casing hex.EncodeToString produced when these ids were issued.
func foldLegacyID(raw string) (string, bool) {
	id := make([]byte, legacyIDLength)
	for i := range legacyIDLength {
		c := raw[i]
		if c >= 'A' && c <= 'F' {
			c += 'a' - 'A'
		}
		isDigit := c >= '0' && c <= '9'
		isHexLetter := c >= 'a' && c <= 'f'
		if !isDigit && !isHexLetter {
			return "", false
		}
		id[i] = c
	}
	return string(id), true
}
