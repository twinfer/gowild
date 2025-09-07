// Package wildcard contains the core implementation of the wildcard matching logic.
// It is intended for internal use by the parent gowild package.
package wildcard

import (
	"errors"
	"slices"
	"unicode"
	"unicode/utf8"
)

// ErrBadPattern indicates a pattern was malformed.
var ErrBadPattern = errors.New("syntax error in pattern")

const (
	// Wildcard characters string (kept for compatibility)
	WildcardChars = "*?.[\\"

	// Individual wildcard constants
	wildcardStar     = '*'
	wildcardQuestion = '?'
	wildcardDot      = '.'
	wildcardBracket  = '['
	wildcardEscape   = '\\'
)

// IsWildcard reports whether r is a wildcard character.
func IsWildcard(r rune) bool {
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
	if pi >= len(pattern) || rune(pattern[pi]) != wildcardBracket {
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

// iterativeMatch case-sensitive version of the iterative matching algorithm.
// It handles backtracking for both `*` and `?`.
func Match[T ~string | ~[]byte](pattern, s T) (bool, error) {
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
			// Escape sequence handling with proper UTF-8 decoding (must be before regular character match!)
			if pIdx+1 >= pLen {
				// Trailing backslash should match literal backslash character
				if sIdx < sLen {
					var sRune rune
					var sRuneWidth int
					if isString {
						sRune, sRuneWidth = utf8.DecodeRuneInString(sStr[sIdx:])
					} else {
						sRune, sRuneWidth = utf8.DecodeRune(sBytes[sIdx:])
					}

					if sRune == wildcardEscape {
						pIdx++             // Move past the backslash in pattern
						sIdx += sRuneWidth // Move past the backslash in string
						// Check for immediate success after escape sequence
						if pIdx >= pLen && sIdx >= sLen {
							return true, nil
						}
						continue
					}
				}
				// No more characters in string or doesn't match backslash, fall through to backtrack
			} else {
				// Check if escaped character matches with proper UTF-8 decoding
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

					if pRune == sRune {
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
			// Character class matching with proper UTF-8 decoding
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
			// Standard character match with proper UTF-8 decoding
			var pRune, sRune rune
			var pRuneWidth, sRuneWidth int

			if isString {
				pRune, pRuneWidth = utf8.DecodeRuneInString(pStr[pIdx:])
				sRune, sRuneWidth = utf8.DecodeRuneInString(sStr[sIdx:])
			} else {
				pRune, pRuneWidth = utf8.DecodeRune(pBytes[pIdx:])
				sRune, sRuneWidth = utf8.DecodeRune(sBytes[sIdx:])
			}

			if pRune == sRune {
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
