// SPDX-License-Identifier: MIT

package str

import (
	"regexp"
	"testing"
)

func TestHighlightStringSimple(t *testing.T) {
	loc := [][]int{}
	loc = append(loc, []int{0, 4})

	got := HighlightString("this", loc, "[in]", "[out]")

	expected := "[in]this[out]"
	if got != expected {
		t.Error("Expected", expected, "got", got)
	}
}

func TestHighlightStringUnicode(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		pattern  string
		expected string
	}{
		{"two-byte ending", "café", "é", "caf[é]"},
		{"three-byte ending", "a猫", "猫", "a[猫]"},
		{"four-byte ending", "a🙂", "🙂", "a[🙂]"},
		{"adjacent Unicode matches", "é猫🙂", ".", "[é][猫][🙂]"},
		{"adjacent Unicode and ASCII matches", "éa", ".", "[é][a]"},
		{"separated Unicode matches", "é 猫 🙂", `\S`, "[é] [猫] [🙂]"},
		{"unhighlighted suffix", "café!", "é", "caf[é]!"},
		{"replacement rune", "\ufffd", ".", "[\ufffd]"},
		{"invalid byte within match", "\xffab", "..", "[\ufffda]b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locations := regexp.MustCompile(tt.pattern).FindAllIndex([]byte(tt.content), -1)
			got := HighlightString(tt.content, locations, "[", "]")
			if got != tt.expected {
				t.Errorf("HighlightString(%q, %v) = %q; want %q", tt.content, locations, got, tt.expected)
			}
		})
	}
}

func TestHighlightStringUnicodeOverlap(t *testing.T) {
	content := "é猫🙂"
	var locations [][]int
	for _, pattern := range []string{"é猫", "猫🙂"} {
		locations = append(locations, regexp.MustCompile(pattern).FindAllIndex([]byte(content), -1)...)
	}
	got := HighlightString(content, locations, "[", "]")
	if expected := "[é猫🙂]"; got != expected {
		t.Errorf("HighlightString(%q, %v) = %q; want %q", content, locations, got, expected)
	}
}

func TestHighlightStringCheckInOut(t *testing.T) {
	loc := [][]int{}
	loc = append(loc, []int{0, 4})

	got := HighlightString("this", loc, "__", "__")

	expected := "__this__"
	if got != expected {
		t.Error("Expected", expected, "got", got)
	}
}

func TestHighlightStringCheck2(t *testing.T) {
	loc := [][]int{}
	loc = append(loc, []int{0, 4})

	got := HighlightString("bing", loc, "__", "__")

	expected := "__bing__"
	if got != expected {
		t.Error("Expected", expected, "got", got)
	}
}

func TestHighlightStringCheckTwoWords(t *testing.T) {
	loc := [][]int{}
	loc = append(loc, []int{0, 4})
	loc = append(loc, []int{5, 9})

	got := HighlightString("this this", loc, "__", "__")

	expected := "__this__ __this__"
	if got != expected {
		t.Error("Expected", expected, "got", got)
	}
}

func TestHighlightStringCheckMixedWords(t *testing.T) {
	loc := [][]int{}
	loc = append(loc, []int{0, 4})
	loc = append(loc, []int{5, 9})
	loc = append(loc, []int{10, 19})

	got := HighlightString("this this something", loc, "__", "__")

	expected := "__this__ __this__ __something__"
	if got != expected {
		t.Error("Expected", expected, "got", got)
	}
}

func TestHighlightStringOverlapStart(t *testing.T) {
	loc := [][]int{}
	loc = append(loc, []int{0, 1})
	loc = append(loc, []int{0, 4})

	got := HighlightString("THIS", loc, "__", "__")

	expected := "__THIS__"
	if got != expected {
		t.Error("Expected", expected, "got", got)
	}
}

func TestHighlightStringOverlapMiddle(t *testing.T) {
	loc := [][]int{}
	loc = append(loc, []int{0, 4})
	loc = append(loc, []int{1, 2})

	got := HighlightString("this", loc, "__", "__")

	expected := "__this__"
	if got != expected {
		t.Error("Expected", expected, "got", got)
	}
}

func TestHighlightStringNoOverlapMiddleNextSame(t *testing.T) {
	loc := [][]int{}
	loc = append(loc, []int{0, 1})
	loc = append(loc, []int{1, 2})

	got := HighlightString("this", loc, "__", "__")

	expected := "__t____h__is"
	if got != expected {
		t.Error("Expected", expected, "got", got)
	}
}

func TestHighlightStringOverlapMiddleLonger(t *testing.T) {
	loc := [][]int{}
	loc = append(loc, []int{0, 2})
	loc = append(loc, []int{1, 4})

	got := HighlightString("this", loc, "__", "__")

	expected := "__this__"
	if got != expected {
		t.Error("Expected", expected, "got", got)
	}
}

func TestBugOne(t *testing.T) {
	loc := [][]int{}
	loc = append(loc, []int{10, 18})

	got := HighlightString("this is unexpected", loc, "__", "__")

	expected := "this is un__expected__"
	if got != expected {
		t.Error("Expected", expected, "got", got)
	}
}

func TestIntegrationRegex(t *testing.T) {
	r := regexp.MustCompile(`1`)
	haystack := "111"

	loc := r.FindAllIndex([]byte(haystack), -1)
	got := HighlightString(haystack, loc, "__", "__")

	if got != "__1____1____1__" {
		t.Error("Expected", "__1____1____1__", "got", got)
	}
}

func TestIntegrationIndexAll(t *testing.T) {
	haystack := "111"

	loc := IndexAll(haystack, "1", -1)
	got := HighlightString(haystack, loc, "__", "__")

	if got != "__1____1____1__" {
		t.Error("Expected", "__1____1____1__", "got", got)
	}
}

func TestIntegrationIndexAllIgnoreCaseUnicode(t *testing.T) {
	haystack := "111"

	loc := IndexAllIgnoreCase(haystack, "1", -1)
	got := HighlightString(haystack, loc, "__", "__")

	if got != "__1____1____1__" {
		t.Error("Expected", "__1____1____1__", "got", got)
	}
}
