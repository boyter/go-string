package str

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var broken = `list1=[0,1,2,3,4,5,6,7,8,9]#制作一个0-9的列表
list1.reverse()#reverse()函数直接对列表中的元素践行反向
print(list1)

# the following line is where it is breaking
list2=[str(i) for i in list1]#将列表中的每一个数字转换成字符串
print(list2)

str1="".join(list2)#通过join()函数，将列表中的单个字符串拼接成一整个字符串
print(str1)

str2=str1[2:8]#对字符串中的第三到第八字符进行切片
print(str2)

str3=str2[::-1]#通过右边第一个开始对整个字符串开始切片，以实现其翻转
print(str3)

i=int(str3)#int()函数试讲字符串转换为整数
print(i)#这里输出的结果虽然与print(str3)相同，但是性质是不同的

#转换成二进制、八进制、十六进制
print('转换成二进制:',bin(i),'转换成八进制:',oct(i), '转换成十六进制:',hex(i))
#二进制、八进制、十六进制这几个进制相互转换的时候，都要先转换为十进制int()`

func TestIndexAllUnicodeOffset(t *testing.T) {
	lines := strings.Split(strings.Replace(broken, "\r\n", "\n", -1), "\n")

	// this has an exception
	for _, l := range lines {
		IndexAllIgnoreCase(l, "list1=[0,1,2,3,4,5,6,7,8,9]#制作一个0", -1)
	}
}

// The KELVIN SIGN (U+212A) case-folds to 'k'/'K'. When a needle contains two
// (or more) runes that each fold to a non-ASCII form, IndexAllIgnoreCase must
// still find a haystack where several of those positions appear in their folded
// form simultaneously. PermuteCaseFolding used to fold only one position at a
// time, so "kk" never produced the double KELVIN SIGN permutation and a haystack
// of "KK" (two KELVIN SIGNs) was missed. We verify against regexp's documented
// case-insensitive behaviour, which IndexAllIgnoreCase is a drop-in for.
func TestIndexAllIgnoreCaseKelvin(t *testing.T) {
	const kelvin = "K" // KELVIN SIGN, folds to k/K
	const longS = "ſ"  // LATIN SMALL LETTER LONG S, folds to s/S

	cases := []struct {
		haystack string
		needle   string
	}{
		{kelvin + kelvin, "kk"},               // two folded runes adjacent
		{kelvin + "e" + kelvin, "kek"},        // folded runes either side of a plain one
		{longS + longS, "ss"},                 // same class of bug via long-s
		{"a" + kelvin + kelvin + "b", "akkb"}, // embedded in a longer (short-path) needle
	}

	for _, c := range cases {
		got := IndexAllIgnoreCase(c.haystack, c.needle, -1)
		want := regexp.MustCompile("(?i)"+regexp.QuoteMeta(c.needle)).FindAllIndex([]byte(c.haystack), -1)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("IndexAllIgnoreCase(%q, %q) = %v, want %v", c.haystack, c.needle, got, want)
		}
	}
}

// PermuteCaseFolding must enumerate the full cross-product of each rune's fold
// equivalents, not just one position at a time, otherwise multi-position folds
// (e.g. both characters of "kk" appearing as the KELVIN SIGN) are never produced.
func TestPermuteCaseFoldingCrossProduct(t *testing.T) {
	const kelvin = "K"
	folded := PermuteCaseFolding("kk")
	if !Contains(folded, kelvin+kelvin) {
		t.Errorf("PermuteCaseFolding(\"kk\") = %q, missing double KELVIN SIGN %q", folded, kelvin+kelvin)
	}
}

// 's' and 'k' fold to a non-ASCII third form, so the two-byte indexByteTwo SIMD
// scan cannot be used when one of them is the anchor. bestCharOffset must
// therefore never prefer them over a SIMD-capable letter, otherwise needles
// like "kelvin" (where 'k' is the rarest letter by English frequency) silently
// fall back to the slow multi-pass scan. This guards the rarity-table override.
func TestAnchorAvoidsNonSIMDLetters(t *testing.T) {
	// A char is SIMD-capable iff it has exactly two single-byte fold variants.
	simdCapable := func(r rune) bool {
		f := PermuteCaseFolding(string(r))
		return len(f) == 2 && len(f[0]) == 1 && len(f[1]) == 1
	}

	for _, needle := range []string{"kelvin", "session", "skunk", "book", "ask", "kiss"} {
		runes := []rune(needle)

		// If the needle contains any SIMD-capable letter, the chosen anchor
		// must be one (i.e. it must not land on 's'/'k').
		hasCapable := false
		for _, r := range runes {
			if simdCapable(r) {
				hasCapable = true
				break
			}
		}
		if !hasCapable {
			continue
		}

		anchor := runes[bestCharOffset(runes, 1)]
		if !simdCapable(anchor) {
			t.Errorf("needle %q selected non-SIMD anchor %q; expected a SIMD-capable letter", needle, string(anchor))
		}
	}
}
