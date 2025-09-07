package wildcard

import (
	"slices"
	"unicode"
	"unicode/utf8"
)

// MatchesFold performs case-insensitive matching against this character class.
func (cc *CharClass) MatchesFold(char rune) bool {
	// Check individual characters with case folding
	matched := slices.ContainsFunc(cc.Chars, func(c rune) bool {
		return equalFoldRune(c, char)
	})

	// Check ranges with case folding if not matched yet
	if !matched {
		matched = slices.ContainsFunc(cc.Ranges, func(r CharRange) bool {
			// Simple case: direct range check
			if char >= r.Start && char <= r.End {
				return true
			}

			// Check case variants of the character
			for f := unicode.SimpleFold(char); f != char; f = unicode.SimpleFold(f) {
				if f >= r.Start && f <= r.End {
					return true
				}
			}

			return false
		})
	}

	// Apply negation if needed
	if cc.Negated {
		matched = !matched
	}

	return matched
}

// iterativeMatchFold case-insensitive version of the iterative matching algorithm.
// It handles backtracking for both `*` and `?`, with Unicode case folding.
func MatchFold[T ~string | ~[]byte](pattern, s T) (bool, error) {
	pLen, sLen := len(pattern), len(s)
	pIdx, sIdx := 0, 0

	// Do type assertion once at the start for performance
	var isString bool
	var pStr, sStr string
	var pBytes, sBytes []byte

	if ps, ok := any(pattern).(string); ok {
		isString = true
		pStr = ps
		sStr = any(s).(string)
	} else {
		isString = false
		pBytes = any(pattern).([]byte)
		sBytes = any(s).([]byte)
	}

	type backtrackState struct {
		pIdx int
		sIdx int
	}
	backtrackStack := []backtrackState{}

	starIdx, sTmpIdx := -1, -1

	for { // The loop continues as long as there are characters to match or states to backtrack to.
		// Check for success: both pattern and string fully consumed
		if pIdx >= pLen && sIdx >= sLen {
			return true, nil
		}

		// Case 1: `*` wildcard. Save its state and continue.
		if pIdx < pLen && pattern[pIdx] == wildcardStar {
			starIdx = pIdx
			sTmpIdx = sIdx
			pIdx++
			continue
		}

		// Case 2: `?` wildcard - each `?` matches exactly one character
		if pIdx < pLen && pattern[pIdx] == wildcardQuestion {
			// Count consecutive `?` wildcards
			qCount := 0
			tempPIdx := pIdx
			for tempPIdx < pLen && pattern[tempPIdx] == wildcardQuestion {
				qCount++
				tempPIdx++
			}

			// Check if we have enough characters left to match all `?` wildcards
			// For strings and byte slices, we need to decode runes to count properly
			runeCount := 0
			tempSIdx := sIdx

			if isString {
				// Count runes in string
				for tempSIdx < sLen && runeCount < qCount {
					_, runeWidth := utf8.DecodeRuneInString(sStr[tempSIdx:])
					tempSIdx += runeWidth
					runeCount++
				}
			} else {
				// Count runes in byte slice
				for tempSIdx < sLen && runeCount < qCount {
					_, runeWidth := utf8.DecodeRune(sBytes[tempSIdx:])
					tempSIdx += runeWidth
					runeCount++
				}
			}

			if runeCount < qCount {
				// Not enough characters, this path fails, need to backtrack
			} else {
				// We can match all `?` wildcards, consume them
				pIdx += qCount
				sIdx = tempSIdx
				continue
			}
		}

		// Case 3: We have a potential match (literal, `.`, or end of pattern).
		// If we're at the end of the string, we might still have a match if the rest of the pattern is optional.
		if sIdx == sLen {
			// Consume trailing star wildcards only (? is not optional in standard glob)
			for pIdx < pLen && pattern[pIdx] == wildcardStar {
				pIdx++
			}
			if pIdx == pLen {
				return true, nil // Matched successfully
			}
			// Mismatch, fall through to backtrack
		} else if pIdx < pLen && pattern[pIdx] == wildcardEscape {
			// Escape sequence handling (case-insensitive) - must be before character match!
			if pIdx+1 >= pLen {
				// Trailing backslash should match literal backslash character
				if sIdx < sLen && s[sIdx] == '\\' {
					pIdx++
					sIdx++
					// Check for immediate success after escape sequence
					if pIdx >= pLen && sIdx >= sLen {
						return true, nil
					}
					continue
				}
				// No more characters or doesn't match backslash
			} else {
				// Check if escaped character matches (case-insensitive) with proper UTF-8 decoding
				if sIdx < sLen {
					var pRune, sRune rune
					var sRuneWidth int

					// Get the escaped character (next byte after backslash)
					pRune = rune(pattern[pIdx+1])

					// Decode the input character properly
					if isString {
						sRune, sRuneWidth = utf8.DecodeRuneInString(sStr[sIdx:])
					} else {
						sRune, sRuneWidth = utf8.DecodeRune(sBytes[sIdx:])
					}

					if equalFoldRune(pRune, sRune) {
						pIdx += 2 // Skip backslash and escaped character
						sIdx += sRuneWidth
						// Check for immediate success after escape sequence
						if pIdx >= pLen && sIdx >= sLen {
							return true, nil
						}
						continue
					}
				}
			}
			// Escaped character doesn't match, fall through to backtrack
		} else if pIdx < pLen && pattern[pIdx] == wildcardDot {
			// `.` matches exactly one non-whitespace character with proper UTF-8 decoding
			if sIdx >= sLen {
				// No character available, fall through to backtrack
			} else {
				// Properly decode the input character
				var sRune rune
				var sRuneWidth int
				if isString {
					sRune, sRuneWidth = utf8.DecodeRuneInString(sStr[sIdx:])
				} else {
					sRune, sRuneWidth = utf8.DecodeRune(sBytes[sIdx:])
				}

				if unicode.IsSpace(sRune) {
					// Character is whitespace, fall through to backtrack
				} else {
					pIdx++
					sIdx += sRuneWidth
					continue
				}
			}
		} else if pIdx < pLen && pattern[pIdx] == wildcardBracket {
			// Character class matching (case-insensitive)
			cc, newPIdx, err := NewCharClass(pattern, pIdx)
			if err != nil {
				return false, err
			}

			if sIdx >= sLen {
				// No character to match against, fall through to backtrack
			} else {
				// Properly decode the input character
				var sRune rune
				var sRuneWidth int
				if isString {
					sRune, sRuneWidth = utf8.DecodeRuneInString(sStr[sIdx:])
				} else {
					sRune, sRuneWidth = utf8.DecodeRune(sBytes[sIdx:])
				}

				if cc.Matches(sRune) {
					pIdx = newPIdx
					sIdx += sRuneWidth
					continue
				}
			}
			// Character class doesn't match or no character available, fall through to backtrack
		} else if pIdx < pLen && sIdx < sLen {
			// Case-insensitive character match with proper UTF-8 decoding
			var pRune, sRune rune
			var pRuneWidth, sRuneWidth int

			if isString {
				pRune, pRuneWidth = utf8.DecodeRuneInString(pStr[pIdx:])
				sRune, sRuneWidth = utf8.DecodeRuneInString(sStr[sIdx:])
			} else {
				pRune, pRuneWidth = utf8.DecodeRune(pBytes[pIdx:])
				sRune, sRuneWidth = utf8.DecodeRune(sBytes[sIdx:])
			}

			if equalFoldRune(pRune, sRune) {
				pIdx += pRuneWidth
				sIdx += sRuneWidth
				continue
			}
		}

		// Case 4: Mismatch or end of pattern. We must backtrack.
		// First, try the `?` stack, which holds the most recent decision points.
		if len(backtrackStack) > 0 {
			lastState := backtrackStack[len(backtrackStack)-1]
			backtrackStack = backtrackStack[:len(backtrackStack)-1]
			// Only use states that are valid for the current string position
			if lastState.sIdx <= sLen {
				pIdx = lastState.pIdx
				sIdx = lastState.sIdx
				continue
			}
		}

		// If the `?` stack is empty, try the `*` backtrack.
		if starIdx != -1 && sTmpIdx < sLen {
			pIdx = starIdx + 1
			sTmpIdx++
			sIdx = sTmpIdx
			continue
		}

		// No backtracking options left.
		return false, nil
	}
}

// // equalFoldRune performs case-insensitive rune comparison using Unicode simple folding.
// // This is more efficient than converting to lowercase and comparing.
func equalFoldRune(r1, r2 rune) bool {
	if r1 == r2 {
		return true
	}
	// Use unicode.SimpleFold for proper case-insensitive comparison
	// This handles all Unicode case folding rules efficiently
	if r1 < r2 {
		r1, r2 = r2, r1
	}
	// SimpleFold cycles through case variants
	for f := unicode.SimpleFold(r2); f != r2; f = unicode.SimpleFold(f) {
		if f == r1 {
			return true
		}
	}
	return false
}
