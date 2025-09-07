// Package wildcard contains the core implementation of the wildcard matching logic.
// It is intended for internal use by the parent gowild package.
package wildcard

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrBadPattern indicates a pattern was malformed.
var ErrBadPattern = errors.New("syntax error in pattern")

const (
	// Wildcard characters string for ContainsAny functions
	wildcardChars = "*?.[\\"

	// Individual wildcard constants
	wildcardStar     = '*'
	wildcardQuestion = '?'
	wildcardDot      = '.'
	wildcardBracket  = '['
	wildcardEscape   = '\\'
)

// isWildcard reports whether r is a wildcard character.
func isWildcard(r rune) bool {
	return r == wildcardStar || r == wildcardQuestion ||
		r == wildcardDot || r == wildcardBracket || r == wildcardEscape
}

// CharRange represents a character range in a character class like [a-z]
type CharRange struct {
	Start rune
	End   rune
}

// CharClass represents a parsed character class like [abc] or [!a-z]
type CharClass struct {
	Negated bool
	Chars   []rune      // Individual characters
	Ranges  []CharRange // Character ranges
}

// Match is the internal, generic core matching function that returns errors.
// It acts as a dispatcher, attempting several fast-path optimizations before
// falling back to a full recursive match for complex patterns.
func Match[T ~string | ~[]byte | ~[]rune](pattern, s T) (bool, error) {
	// Type-specific optimizations handled in each branch below
	if pStr, ok := any(pattern).(string); ok {
		str := any(s).(string)

		// single "*" wildcard
		if pStr == "*" {
			return true, nil
		}

		// Inlined isExactMatch check for strings
		if !strings.ContainsAny(pStr, wildcardChars) {
			return pStr == str, nil
		}

		// Fast path for simple patterns
		if matched, ok := fastPatternMatchString(pStr, str); ok {
			return matched, nil
		}

		// Use iterative algorithm for better performance
		return iterativeMatch(pStr, str)

	} else if pBytes, ok := any(pattern).([]byte); ok {
		sBytes := any(s).([]byte)

		// Inlined isExactMatch check for byte slices
		if !bytes.ContainsAny(pBytes, wildcardChars) {
			return bytes.Equal(pBytes, sBytes), nil
		}
		// single "*" wildcard
		if len(pBytes) == 1 && pBytes[0] == '*' {
			return true, nil
		}
		// Fast path for simple patterns
		if matched, ok := fastPatternMatchBytes(pBytes, sBytes); ok {
			return matched, nil
		}

		// Use iterative algorithm for better performance
		return iterativeMatch(pBytes, sBytes)

	} else if pRunes, ok := any(pattern).([]rune); ok {
		runes := any(s).([]rune)

		// Inlined isExactMatch check for rune slices

		if !slices.ContainsFunc(pRunes, isWildcard) {
			return slices.Equal(pRunes, runes), nil
		}

		// single "*" wildcard
		if len(pRunes) == 1 && pRunes[0] == '*' {
			return true, nil
		}

		// Keep recursive implementation for runes (Unicode correctness)
		return matchRecursiveRunes(pRunes, runes, 0, 0)
	}

	// Should never be reached due to generic type constraints.
	return false, nil
}

// fastPatternMatchString implements the fast path logic for strings.
func fastPatternMatchString(pattern, s string) (bool, bool) {
	// Enhanced fast paths based on Go stdlib patterns

	// 1. Handle "*word*" (contains pattern)
	if len(pattern) >= 2 && pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		middle := pattern[1 : len(pattern)-1]
		if !strings.ContainsAny(middle, wildcardChars) {
			return strings.Contains(s, middle), true
		}
	}

	// 2. Handle "prefix*" if the prefix contains no other wildcards or character classes.
	if prefix, found := strings.CutSuffix(pattern, "*"); found {
		if !strings.ContainsAny(prefix, wildcardChars) {
			return strings.HasPrefix(s, prefix), true
		}
	}

	// 3. Handle "*suffix" if the suffix contains no other wildcards or character classes.
	if suffix, found := strings.CutPrefix(pattern, "*"); found {
		if !strings.ContainsAny(suffix, wildcardChars) {
			return strings.HasSuffix(s, suffix), true
		}
	}

	// 4. Handle "prefix*suffix" if the prefix and suffix contain no other wildcards or character classes.
	if prefix, suffix, found := strings.Cut(pattern, "*"); found && prefix != "" && suffix != "" {
		if !strings.ContainsAny(prefix, wildcardChars) && !strings.ContainsAny(suffix, wildcardChars) {
			matched := len(s) >= len(prefix)+len(suffix) &&
				strings.HasPrefix(s, prefix) &&
				strings.HasSuffix(s, suffix)
			return matched, true
		}
	}

	// Note: Fast paths 5 and 6 for trailing/leading wildcards are disabled
	// because they can cause issues with multi-byte Unicode characters
	// where byte indices don't align with character boundaries.

	return false, false
}

// fastPatternMatchBytes implements the fast path logic for byte slices.
func fastPatternMatchBytes(pattern, s []byte) (bool, bool) {
	// Enhanced fast paths for byte slices

	// 1. Handle "*word*" (contains pattern)
	if len(pattern) >= 2 && pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		middle := pattern[1 : len(pattern)-1]
		if !bytes.ContainsAny(middle, wildcardChars) {
			return bytes.Contains(s, middle), true
		}
	}

	// 2. Handle "prefix*" if the prefix contains no other wildcards or character classes.
	if prefix, found := bytes.CutSuffix(pattern, []byte("*")); found {
		if !bytes.ContainsAny(prefix, wildcardChars) {
			return bytes.HasPrefix(s, prefix), true
		}
	}

	// 3. Handle "*suffix" if the suffix contains no other wildcards or character classes.
	if suffix, found := bytes.CutPrefix(pattern, []byte("*")); found {
		if !bytes.ContainsAny(suffix, wildcardChars) {
			return bytes.HasSuffix(s, suffix), true
		}
	}

	// 4. Handle "prefix*suffix" if the prefix and suffix contain no other wildcards or character classes.
	if prefix, suffix, found := bytes.Cut(pattern, []byte("*")); found && len(prefix) > 0 && len(suffix) > 0 {
		if !bytes.ContainsAny(prefix, wildcardChars) && !bytes.ContainsAny(suffix, wildcardChars) {
			matched := len(s) >= len(prefix)+len(suffix) &&
				bytes.HasPrefix(s, prefix) &&
				bytes.HasSuffix(s, suffix)
			return matched, true
		}
	}

	// Note: Fast paths 5 and 6 for trailing/leading wildcards are disabled
	// because they can cause issues with multi-byte Unicode characters
	// where byte indices don't align with character boundaries.

	return false, false
}

// NewCharClass creates a new CharClass by parsing the pattern at the given position.
// Returns the parsed CharClass, the new position after the class, and any error.
func NewCharClass[T ~string | ~[]byte](pattern T, pi int) (*CharClass, int, error) {
	switch p := any(pattern).(type) {
	case string:
		return parseCharClassString(p, pi)
	case []byte:
		return parseCharClassString(string(pattern), pi)
	}
	return nil, pi, ErrBadPattern
}

// Matches checks if the given rune matches this character class.
func (cc *CharClass) Matches(char rune) bool {
	matched := slices.Contains(cc.Chars, char)

	// Check ranges if not matched yet
	if !matched {
		matched = slices.ContainsFunc(cc.Ranges, func(r CharRange) bool {
			return char >= r.Start && char <= r.End
		})
	}

	// Apply negation if needed
	if cc.Negated {
		matched = !matched
	}

	return matched
}

// parseCharClassString parses a character class from a string pattern.
func parseCharClassString(pattern string, pi int) (*CharClass, int, error) {
	if pi >= len(pattern) || rune(pattern[pi]) != '[' {
		return nil, pi, ErrBadPattern
	}

	pi++ // Skip the opening '['
	if pi >= len(pattern) {
		return nil, pi, ErrBadPattern
	}

	cc := &CharClass{}

	// Check for negation
	if pi < len(pattern) && (rune(pattern[pi]) == '^' || rune(pattern[pi]) == '!') {
		cc.Negated = true
		pi++
		if pi >= len(pattern) {
			return nil, pi, ErrBadPattern
		}
	}

	firstChar := true // First character after opening bracket (and optional negation)

	for pi < len(pattern) {
		// ']' is only treated as closing bracket if it's not the first character
		if rune(pattern[pi]) == ']' && !firstChar {
			pi++ // Skip the closing ']'
			break
		}
		firstChar = false

		// Handle escape sequences
		var c1 rune
		if rune(pattern[pi]) == '\\' {
			if pi+1 >= len(pattern) {
				return nil, pi, ErrBadPattern
			}
			pi++
			if pi >= len(pattern) {
				return nil, pi, ErrBadPattern
			}
		}
		c1 = rune(pattern[pi])
		pi++

		// Check for range
		if pi+1 < len(pattern) && rune(pattern[pi]) == '-' && rune(pattern[pi+1]) != ']' {
			pi++ // Skip the '-'

			// Handle escape in range end
			var c2 rune
			if rune(pattern[pi]) == '\\' {
				if pi+1 >= len(pattern) {
					return nil, pi, ErrBadPattern
				}
				pi++
			}
			if pi >= len(pattern) {
				return nil, pi, ErrBadPattern
			}
			c2 = rune(pattern[pi])
			pi++

			// Validate range
			if c1 > c2 {
				return nil, pi, ErrBadPattern // Invalid range like [z-a]
			}
			// Add range
			cc.Ranges = append(cc.Ranges, CharRange{Start: c1, End: c2})
		} else {
			// Add single character
			cc.Chars = append(cc.Chars, c1)
		}
	}

	return cc, pi, nil
}

// matchRecursiveRunes is the core backtracking algorithm for rune-based matching.
// It is structurally similar to matchRecursive but operates on slices of runes
// to correctly handle multi-byte Unicode characters.
func matchRecursiveRunes(pattern, s []rune, pi, si int) (bool, error) {
	plen, slen := len(pattern), len(s)

	for pi < plen {
		pc := pattern[pi]

		switch pc {
		case wildcardStar:
			// Coalesce consecutive stars into one.
			remaining := pattern[pi:]
			idx := slices.IndexFunc(remaining, func(r rune) bool { return r != wildcardStar })
			if idx == -1 {
				return true, nil
			} // Pattern ends with stars.
			pi = pi + idx

			// For a `*`, try to match the rest of the pattern against every suffix.
			for si <= slen {
				if matched, err := matchRecursiveRunes(pattern, s, pi, si); err != nil {
					return false, err
				} else if matched {
					return true, nil
				}
				si++
			}
			return false, nil

		case wildcardQuestion:
			// `?` matches exactly one character in glob patterns
			if si >= slen {
				return false, nil // No more characters to match
			}
			pi++
			si++

		case wildcardDot:
			// `.` matches exactly one non-whitespace character.
			if si >= slen {
				return false, nil
			}
			// Check if current character is whitespace
			if unicode.IsSpace(rune(s[si])) {
				return false, nil
			}
			pi++
			si++

		default:
			// Standard rune match.
			if si >= slen || s[si] != pc {
				return false, nil
			}
			pi++
			si++
		}
	}

	return si == slen, nil
}

// iterativeMatch case-sensitive version of the iterative matching algorithm.
// It handles backtracking for both `*` and `?`.
func iterativeMatch[T ~string | ~[]byte](pattern, s T) (bool, error) {
	pLen, sLen := len(pattern), len(s)
	pIdx, sIdx := 0, 0

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

			if sStr, ok := any(s).(string); ok {
				// Count runes in string
				for tempSIdx < sLen && runeCount < qCount {
					_, runeWidth := utf8.DecodeRuneInString(sStr[tempSIdx:])
					tempSIdx += runeWidth
					runeCount++
				}
			} else {
				// Count runes in byte slice
				sBytes := any(s).([]byte)
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
			// Escape sequence handling (must be before regular character match!)
			if pIdx+1 >= pLen {
				// Trailing backslash should match literal backslash character
				if sIdx < sLen && s[sIdx] == wildcardEscape {
					pIdx++ // Move past the backslash in pattern
					sIdx++ // Move past the backslash in string
					// Check for immediate success after escape sequence
					if pIdx >= pLen && sIdx >= sLen {
						return true, nil
					}
					continue
				}
				// No more characters in string or doesn't match backslash, fall through to backtrack
			} else {
				// Check if escaped character matches
				if sIdx < sLen && pattern[pIdx+1] == s[sIdx] {
					pIdx += 2 // Skip backslash and escaped character
					sIdx++
					// Check for immediate success after escape sequence
					if pIdx >= pLen && sIdx >= sLen {
						return true, nil
					}
					continue
				}
			}
			// Escaped character doesn't match, fall through to backtrack
		} else if pIdx < pLen && pattern[pIdx] == wildcardDot {
			// `.` matches exactly one non-whitespace character.
			if sIdx >= sLen || unicode.IsSpace(rune(s[sIdx])) {
				// No character available or character is whitespace, fall through to backtrack
			} else {
				pIdx++
				sIdx++
				continue
			}
		} else if pIdx < pLen && pattern[pIdx] == s[sIdx] {
			// Standard character match.
			pIdx++
			sIdx++
			continue
		} else if pIdx < pLen && pattern[pIdx] == wildcardBracket {
			// Character class matching
			cc, newPIdx, err := NewCharClass(pattern, pIdx)
			if err != nil {
				return false, err
			}
			if cc.Matches(rune(s[sIdx])) {
				pIdx = newPIdx
				sIdx++
				continue
			}
			// Character class doesn't match, fall through to backtrack
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

// HasWildcards checks if a string pattern contains wildcard characters
func HasWildcards(pattern string) bool {
	return strings.ContainsAny(pattern, wildcardChars)
}

// HasWildcardsBytes checks if a byte slice pattern contains wildcard characters
func HasWildcardsBytes(pattern []byte) bool {
	return bytes.ContainsAny(pattern, wildcardChars)
}

// HasWildcardsRunes checks if a rune slice pattern contains wildcard characters
func HasWildcardsRunes(pattern []rune) bool {
	return slices.ContainsFunc(pattern, isWildcard)
}

// EqualBytes performs efficient byte slice comparison
func EqualBytes(a, b []byte) bool {
	return bytes.Equal(a, b)
}

// EqualRunes performs efficient rune slice comparison
func EqualRunes(a, b []rune) bool {
	return slices.Equal(a, b)
}
