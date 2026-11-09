// Package langs names the site's languages. tygo exports these constants to the
// frontend (src/scripts/langs.gen.ts), where a test checks them against the
// locales the site serves, so the two lists cannot drift apart.
package langs

const (
	Sv = "sv"
	Is = "is"
	En = "en"
	De = "de"
)

// All returns every language, in the order the booklet lists them.
func All() []string {
	return []string{Sv, Is, En, De}
}
