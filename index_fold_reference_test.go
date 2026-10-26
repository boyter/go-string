// SPDX-License-Identifier: MIT

package str

import (
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// indexAllIgnoreCaseOld is a verbatim copy of IndexAllIgnoreCase as it was
// before the verify loop switched from AllSimpleFold to foldEqual. It is kept
// as the reference implementation for FuzzIndexAllIgnoreCaseMatchesOld.
func indexAllIgnoreCaseOld(haystack string, needle string, limit int) [][]int {
	// The below needed to avoid timeout crash found using go-fuzz
	if len(haystack) == 0 || len(needle) == 0 {
		return nil
	}

	// One of the problems with finding locations ignoring case is that
	// the different case representations can have different byte counts
	// which means the locations using strings or bytes Index can be off
	// if you apply strings.ToLower to your haystack then use strings.Index.
	//
	// This can be overcome using regular expressions but suffers the penalty
	// of hitting the regex engine and paying the price of case-
	// insensitive match there.
	//
	// This method tries something else which is used by some regex engines
	// such as the one in Rust where given a str literal if you get
	// all the case options of that such as turning foo into foo Foo fOo FOo foO FoO fOO FOO
	// and then use Boyer-Moore or some such for those. Of course using something
	// like Aho-Corasick or Rabin-Karp to get multi match would be a better idea so you
	// can match all of the input in one pass.
	//
	// If the needle is over some amount of characters long you chop off the first few
	// and then search for those. However this means you are not finding actual matches and as such
	// you the need to validate a potential match after you have found one.
	// The confirmation match is done in a loop because for some literals regular expression
	// is still to slow, although for most its a valid option.
	var locs [][]int

	// Char limit is the cut-off where we switch from all case permutations
	// to just the first 3 and then check for an actual match
	// in my tests 3 speeds things up the most against test data
	// of many famous books concatenated together and large
	// amounts of data from /dev/urandom
	var charLimit = 3

	if utf8.RuneCountInString(needle) <= charLimit {
		// We are below the limit we set, so get all the search
		// terms and search for that

		// Generally, I am against caches inside libraries. However, in this case...
		// when the IndexAllIgnoreCase method is called repeatedly, it quite often
		// ends up performing case folding on the same thing over and over again, which
		// can become the most expensive operation. So we keep a VERY small cache
		// to avoid that being an issue.
		// Try a read lock first.
		// This allows multiple goroutines to hit the cache simultaneously
		// without blocking each other.
		_permuteCacheLock.RLock()
		searchTerms, ok := _permuteCache[needle]
		_permuteCacheLock.RUnlock()

		if !ok {
			// we now have to do something about it
			_permuteCacheLock.Lock()

			// Another goroutine might have
			// inserted the key while we were waiting for the lock.
			searchTerms, ok = _permuteCache[needle]
			if !ok {
				if len(_permuteCache) > CacheSize {
					_permuteCache = map[string][]string{}
				}
				searchTerms = PermuteCaseFolding(needle)
				_permuteCache[needle] = searchTerms
			}
			_permuteCacheLock.Unlock()
		}

		// Stream the permutation matches out in regexp.FindAllIndex order,
		// stopping as soon as we have limit of them, so a small limit on a huge
		// haystack stays cheap. collectFoldedMatches already returns ordered,
		// non-overlapping results, so we can return directly.
		locs = collectFoldedMatches(haystack, searchTerms, limit)

		if len(locs) == 0 {
			return nil
		}
		return locs
	} else {
		// Over the character limit so look for potential matches and only then check to find real ones

		// Note that we have to use runes here to avoid cutting bytes off so
		// cast things around to ensure it works
		needleRune := []rune(needle)

		searchStart := bestCharOffset(needleRune, 1)
		searchChar := string(needleRune[searchStart : searchStart+1])

		_permuteCacheLock.RLock()
		searchTerms, ok := _permuteCache[searchChar]
		_permuteCacheLock.RUnlock()

		if !ok {
			// nothing found so we have to do something
			_permuteCacheLock.Lock()
			searchTerms, ok = _permuteCache[searchChar]
			if !ok {
				if len(_permuteCache) > CacheSize {
					_permuteCache = map[string][]string{}
				}
				searchTerms = PermuteCaseFolding(searchChar)
				_permuteCache[searchChar] = searchTerms
			}
			_permuteCacheLock.Unlock()
		}

		// Fast path: when the search character is an ASCII letter with exactly
		// two fold variants (upper + lower), use indexByteTwo to find both in
		// a single SIMD pass instead of two separate IndexAll scans.
		// This covers all ASCII letters except s/S (which also fold to ſ)
		// and k/K (which also fold to K); those have three fold variants so
		// the len == 2 guard below excludes them. The rarity table (see init)
		// omits s and k so bestCharOffset avoids anchoring on them whenever the
		// needle has a SIMD-capable letter, keeping needles like "kelvin" on
		// this fast path.
		searchRune := needleRune[searchStart]
		useSIMD := len(searchTerms) == 2 && len(searchTerms[0]) == 1 && len(searchTerms[1]) == 1 &&
			searchRune < utf8.RuneSelf &&
			unicode.ToLower(searchRune) != unicode.ToUpper(searchRune)

		if useSIMD {
			b1 := searchTerms[0][0]
			b2 := searchTerms[1][0]
			haystackBytes := unsafe.Slice(unsafe.StringData(haystack), len(haystack))
			offset := 0
			for offset < len(haystackBytes) {
				idx := indexByteTwo(haystackBytes[offset:], b1, b2)
				if idx < 0 {
					break
				}
				bytePos := offset + idx

				needleStart := bytePos
				skip := false
				for k := 0; k < searchStart; k++ {
					if needleStart <= 0 {
						skip = true
						break
					}
					_, size := utf8.DecodeLastRuneInString(haystack[:needleStart])
					needleStart -= size
				}
				if !skip {
					pos := needleStart
					isMatch := true

					for i := 0; i < len(needleRune); i++ {
						if pos >= len(haystack) {
							isMatch = false
							break
						}

						r, size := utf8.DecodeRuneInString(haystack[pos:])

						if r != needleRune[i] {
							foldMatch := false
							for _, f := range AllSimpleFold(r) {
								if f == needleRune[i] {
									foldMatch = true
									break
								}
							}
							if !foldMatch {
								isMatch = false
								break
							}
						}
						pos += size
					}

					if isMatch {
						locs = append(locs, []int{needleStart, pos})
					}
				}
				offset = bytePos + 1
			}
		} else {
			// Slow path for characters with >2 fold variants (e.g. s/S/ſ, k/K/K)
			// or non-ASCII characters. Uses multiple IndexAll passes.
			for _, term := range searchTerms {
				potentialMatches := IndexAll(haystack, term, -1)

				for _, match := range potentialMatches {
					needleStart := match[0]
					skip := false
					for k := 0; k < searchStart; k++ {
						if needleStart <= 0 {
							skip = true
							break
						}
						_, size := utf8.DecodeLastRuneInString(haystack[:needleStart])
						needleStart -= size
					}
					if skip {
						continue
					}
					pos := needleStart
					isMatch := true

					for i := 0; i < len(needleRune); i++ {
						if pos >= len(haystack) {
							isMatch = false
							break
						}

						r, size := utf8.DecodeRuneInString(haystack[pos:])

						if r != needleRune[i] {
							foldMatch := false
							for _, f := range AllSimpleFold(r) {
								if f == needleRune[i] {
									foldMatch = true
									break
								}
							}
							if !foldMatch {
								isMatch = false
								break
							}
						}
						pos += size
					}

					if isMatch {
						locs = append(locs, []int{needleStart, pos})
					}
				}
			}
		}

	}

	// The case-fold search above runs each permutation independently and
	// concatenates the hits, so the slice can be unordered (a lowercase match
	// found after an uppercase one further left) and can contain overlaps (e.g.
	// needle "aa" matching both [0,2] and [1,3] in "aAa", or different byte
	// lengths via the ſ/KELVIN SIGN folds). regexp.FindAllIndex, which this
	// method mirrors, returns matches left-to-right and non-overlapping, so we
	// order and resolve overlaps before applying the limit.
	locs = sortAndDedupe(locs)

	// Match regexp.FindAllIndex's limit semantics: a negative limit returns all
	// matches, while a non-negative limit returns at most that many (so 0 returns
	// none). The case-sensitive IndexAll already behaves this way.
	if limit >= 0 && len(locs) > limit {
		locs = locs[:limit]
	}

	// Retain compatibility with the FindAllIndex method
	if len(locs) == 0 {
		return nil
	}

	return locs
}
