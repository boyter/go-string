// SPDX-License-Identifier: MIT

package str

import (
	"reflect"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// oldFoldEqual is the verify-loop comparison IndexAllIgnoreCase used before
// foldEqual: an exact match, or n appearing in AllSimpleFold(r).
func oldFoldEqual(r, n rune) bool {
	if r == n {
		return true
	}
	for _, f := range AllSimpleFold(r) {
		if f == n {
			return true
		}
	}
	return false
}

// foldTestRunes is a broad set of runes covering ASCII, Latin-1, Latin
// Extended-A, Greek and Cyrillic, plus the folds that commonly trip up
// case-insensitive matching.
func foldTestRunes() []rune {
	var runes []rune
	add := func(lo, hi rune) {
		for r := lo; r <= hi; r++ {
			runes = append(runes, r)
		}
	}
	add(0x0000, 0x00FF) // ASCII and Latin-1
	add(0x0100, 0x017F) // Latin Extended-A (includes U+0130, U+0131, U+017F)
	add(0x0370, 0x03FF) // Greek (includes Σ σ ς, μ Μ)
	add(0x0400, 0x04FF) // Cyrillic
	runes = append(runes,
		0x212A,         // KELVIN SIGN
		0x2126,         // OHM SIGN
		0x212B,         // ANGSTROM SIGN
		0x1E9E,         // LATIN CAPITAL LETTER SHARP S
		0x1E9B,         // LATIN SMALL LETTER LONG S WITH DOT ABOVE
		0x1E60, 0x1E61, // Ṡ ṡ, which fold with U+1E9B
		0x1FBE,                         // GREEK PROSGEGRAMMENI
		0x1C80, 0x1C81, 0x1C82, 0x1C88, // Cyrillic extended-C small letter variants
		0xA64A, 0xA64B,
		0x10400, 0x10428, // Deseret, outside the BMP
		utf8.RuneError,
		utf8.MaxRune,
	)
	return runes
}

func TestFoldEqualMatchesAllSimpleFold(t *testing.T) {
	runes := foldTestRunes()
	for _, r := range runes {
		for _, n := range runes {
			if got, want := foldEqual(r, n), oldFoldEqual(r, n); got != want {
				t.Errorf("foldEqual(%U %q, %U %q) = %v, want %v", r, r, n, n, got, want)
			}
		}
	}
}

func TestFoldEqualTrickyFolds(t *testing.T) {
	cases := []struct {
		r, n rune
		want bool
	}{
		{0x212A, 'k', true}, {0x212A, 'K', true}, {'k', 0x212A, true}, {'K', 0x212A, true},
		{0x017F, 's', true}, {0x017F, 'S', true}, {'s', 0x017F, true}, {'S', 0x017F, true},
		{0x00B5, 'μ', true}, {0x00B5, 'Μ', true}, {'μ', 0x00B5, true}, {'Μ', 0x00B5, true},
		{0x00DF, 0x1E9E, true}, {0x1E9E, 0x00DF, true},
		{0x00DF, 's', false}, {0x00DF, 'S', false},
		{0x0130, 'i', false}, {0x0130, 'I', false}, {0x0131, 'i', false}, {0x0131, 'I', false},
		{0x0130, 0x0131, false},
		{'Σ', 'σ', true}, {'Σ', 'ς', true}, {'σ', 'ς', true}, {'ς', 'Σ', true},
		{'a', 'A', true}, {'z', 'Z', true}, {'@', '`', false}, {'[', '{', false}, {'1', '1', true},
	}
	for _, c := range cases {
		if got := foldEqual(c.r, c.n); got != c.want {
			t.Errorf("foldEqual(%U, %U) = %v, want %v", c.r, c.n, got, c.want)
		}
		if got := oldFoldEqual(c.r, c.n); got != c.want {
			t.Errorf("oldFoldEqual(%U, %U) = %v, want %v", c.r, c.n, got, c.want)
		}
	}
}

// TestFoldEqualEveryOrbit checks every rune against every member of its own
// fold orbit, plus its neighbours, across the whole Unicode range.
func TestFoldEqualEveryOrbit(t *testing.T) {
	for r := rune(0); r <= unicode.MaxRune; r++ {
		for _, n := range append(AllSimpleFold(r), r-1, r+1) {
			if got, want := foldEqual(r, n), oldFoldEqual(r, n); got != want {
				t.Fatalf("foldEqual(%U, %U) = %v, want %v", r, n, got, want)
			}
		}
	}
}

func TestFoldEqualDoesNotAllocate(t *testing.T) {
	pairs := [][2]rune{{'a', 'B'}, {'k', 0x212A}, {0x212A, 's'}, {'Σ', 'ς'}, {0x00DF, 0x1E9E}, {'x', 0x0416}}
	allocs := testing.AllocsPerRun(100, func() {
		for _, p := range pairs {
			foldEqual(p[0], p[1])
		}
	})
	if allocs != 0 {
		t.Fatalf("foldEqual allocated %v times per run, want 0", allocs)
	}
}

// foldHeavyRunes biases fuzz inputs towards runes with unusual folds, mixed
// byte widths, and an invalid UTF-8 byte.
var foldHeavyRunes = []string{
	"a", "A", "e", "E", "k", "K", "K", "s", "S", "ſ",
	"i", "I", "İ", "ı", "µ", "μ", "Μ", "ß", "ẞ",
	"Σ", "σ", "ς", "Ω", "ω", "Ω", "Å", "å", "Å",
	"ж", "Ж", "ṡ", "ẛ", "ι", "ι", "\U00010400", "\U00010428",
	" ", "_", "(", "\xff",
}

func foldHeavyString(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		sb.WriteString(foldHeavyRunes[int(c)%len(foldHeavyRunes)])
	}
	return sb.String()
}

func FuzzIndexAllIgnoreCaseMatchesOld(f *testing.F) {
	f.Add([]byte("ParseDuration parseduration PARSEDURATION"), []byte("ParseDuration"), uint8(0), int8(-1))
	f.Add([]byte{6, 7, 8, 9, 10, 11, 12, 6, 7, 8}, []byte{6, 9, 12, 7}, uint8(1), int8(-1))
	f.Add([]byte{19, 20, 21, 19, 20, 21, 15, 16, 17, 18}, []byte{20, 21, 19, 16}, uint8(1), int8(2))
	f.Add([]byte{0, 1, 4, 5, 39, 13, 14, 2, 3}, []byte{1, 0, 5, 4}, uint8(2), int8(0))
	f.Add([]byte("kelvin Kelvin KELVIN ſtring String"), []byte("Kelvin"), uint8(0), int8(-1))

	f.Fuzz(func(t *testing.T, hay []byte, needle []byte, mode uint8, limit int8) {
		var h, n string
		switch mode % 3 {
		case 0: // raw input
			h, n = string(hay), string(needle)
		case 1: // both mapped onto fold-heavy runes
			h, n = foldHeavyString(hay), foldHeavyString(needle)
		case 2: // fold-heavy haystack, needle drawn from the haystack so matches occur
			h = foldHeavyString(hay)
			if len(needle) > 0 && len(h) > 0 {
				start := int(needle[0]) % len(h)
				end := start + len(needle)
				if end > len(h) {
					end = len(h)
				}
				n = h[start:end]
			}
		}
		lim := int(limit)
		if lim < -1 {
			lim = -1
		}

		got := IndexAllIgnoreCase(h, n, lim)
		want := indexAllIgnoreCaseOld(h, n, lim)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("IndexAllIgnoreCase(%q, %q, %d) = %v, want %v", h, n, lim, got, want)
		}
	})
}
