// SPDX-License-Identifier: MIT

package str

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sourceHaystack builds a ~1MB haystack of real Go source by repeating this
// package's own non-test .go files. It is realistic code (identifiers, comments,
// punctuation) with lots of near-miss anchors for typical identifier needles.
func sourceHaystack(b *testing.B) string {
	b.Helper()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		b.Fatal(err)
	}
	var src strings.Builder
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		content, err := os.ReadFile(p)
		if err != nil {
			b.Fatal(err)
		}
		src.Write(content)
	}
	if src.Len() == 0 {
		b.Fatal("no source files found for haystack")
	}
	return strings.Repeat(src.String(), (1<<20)/src.Len()+1)
}

func BenchmarkIndexAllIgnoreCaseSourceNearMiss(b *testing.B) {
	haystack := sourceHaystack(b)
	needle := "ParseDuration"
	if strings.Contains(strings.ToLower(haystack), strings.ToLower(needle)) {
		b.Fatalf("haystack unexpectedly contains %q", needle)
	}

	b.SetBytes(int64(len(haystack)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if matches := IndexAllIgnoreCase(haystack, needle, -1); matches != nil {
			b.Fatalf("expected no matches, got %d", len(matches))
		}
	}
}
