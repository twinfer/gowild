/*
Copyright (c) 2025 twinfer.com contact@twinfer.com Copyright (c) 2025 Khalid Daoud mohamed.khalid@gmail.com

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
*/

// Package wildcard contains optimized wildcard matching implementations.
// This file provides the Unicode-aware matching engine with full UTF-8 support
// and case-insensitive matching capabilities using Unicode simple folding.
// For maximum performance with ASCII-only input, see match.go.
package wildcard

import (
	"bytes"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// IsWildcard reports whether r is a wildcard character.
// func IsWildcard(r rune) bool {
// 	return r == wildcardStar || r == wildcardQuestion ||
// 		r == wildcardDot || r == wildcardBracket || r == wildcardEscape
// }

// charRangeFold represents a character range in a character class like [a-z]
type charRangeFold struct {
	Start rune
	End   rune
}

// charClassFold represents a parsed character class like [abc] or [!a-z]
type charClassFold struct {
	Negated bool
	Chars   []rune          // Individual characters
	Ranges  []charRangeFold // Character ranges
}

// MatchesWithFold checks if the given rune matches this character class.
// Note: Character classes are always case-sensitive, regardless of the fold parameter.
// This maintains compatibility with standard glob behavior where [a-z] should not match 'A'.
func (cc *charClassFold) MatchesWithFold(char rune, fold bool) bool {
	// Character classes are always case-sensitive
	matched := slices.Contains(cc.Chars, char)

	// Check ranges if not matched yet
	if !matched {
		matched = slices.ContainsFunc(cc.Ranges, func(r charRangeFold) bool {
			return char >= r.Start && char <= r.End
		})
	}

	// Apply negation if needed
	if cc.Negated {
		matched = !matched
	}

	return matched
}

// NewcharClassFold creates a new charClassFold by parsing the pattern at the given position.
// Returns the parsed charClassFold, the new position after the class, and any error.
func NewcharClassFold[T ~string | ~[]byte](pattern T, pi int) (*charClassFold, int, error) {
	// Use proper UTF-8 decoding for consistent behavior
	var isString bool
	var pStr string
	var pBytes []byte

	if ps, ok := any(pattern).(string); ok {
		isString = true
		pStr = ps
	} else {
		isString = false
		pBytes = any(pattern).([]byte)
	}

	// Helper function to decode rune at position
	decodeRune := func(pos int) (rune, int) {
		if isString {
			if pos >= len(pStr) {
				return 0, 0
			}
			return utf8.DecodeRuneInString(pStr[pos:])
		} else {
			if pos >= len(pBytes) {
				return 0, 0
			}
			return utf8.DecodeRune(pBytes[pos:])
		}
	}

	if pi >= len(pattern) {
		return nil, pi, ErrBadPattern
	}

	r, width := decodeRune(pi)
	if r != wildcardBracket {
		return nil, pi, ErrBadPattern
	}

	pi += width // Skip the opening '['
	if pi >= len(pattern) {
		return nil, pi, ErrBadPattern
	}

	cc := &charClassFold{}

	// Check for negation
	if pi < len(pattern) {
		r, width = decodeRune(pi)
		if r == '^' || r == '!' {
			cc.Negated = true
			pi += width
			if pi >= len(pattern) {
				return nil, pi, ErrBadPattern
			}
		}
	}

	firstChar := true // First character after opening bracket (and optional negation)

	closed := false
	for pi < len(pattern) {
		// Check for closing bracket
		r, width := decodeRune(pi)
		if r == ']' && !firstChar {
			pi += width // Skip the closing ']'
			closed = true
			break
		}
		firstChar = false

		// Handle escape sequences and character reading
		var c1 rune
		if r == '\\' {
			pi += width // Skip the backslash
			if pi >= len(pattern) {
				return nil, pi, ErrBadPattern
			}
			// The escaped character is treated as a literal rune
			r2, width2 := decodeRune(pi)
			c1 = r2
			pi += width2
		} else {
			// Regular character
			c1 = r
			pi += width
		}

		// Check for range (need to check current position after advancing)
		if pi < len(pattern) {
			dashRune, dashWidth := decodeRune(pi)
			if dashRune == '-' && pi+dashWidth < len(pattern) {
				// Check if character after dash is not ']'
				afterDash, _ := decodeRune(pi + dashWidth)
				if afterDash != ']' {
					// This is a range, skip the '-' and parse end character
					pi += dashWidth

					// Handle escape in range end
					var c2 rune
					if pi >= len(pattern) {
						return nil, pi, ErrBadPattern
					}
					r3, width3 := decodeRune(pi)
					if r3 == '\\' {
						pi += width3 // Skip the backslash
						if pi >= len(pattern) {
							return nil, pi, ErrBadPattern
						}
						r4, width4 := decodeRune(pi)
						c2 = r4
						pi += width4
					} else {
						c2 = r3
						pi += width3
					}

					// Validate range
					if c1 > c2 {
						return nil, pi, ErrBadPattern // Invalid range like [z-a]
					}
					// Add range
					cc.Ranges = append(cc.Ranges, charRangeFold{Start: c1, End: c2})
				} else {
					// Dash followed by ']', treat dash as literal character
					cc.Chars = append(cc.Chars, c1)
				}
			} else {
				// No dash, treat as single character
				cc.Chars = append(cc.Chars, c1)
			}
		} else {
			// End of pattern, treat as single character
			cc.Chars = append(cc.Chars, c1)
		}
	}

	// Check if character class was properly closed
	if !closed {
		return nil, pi, ErrBadPattern
	}

	return cc, pi, nil
}

// equalFoldRune performs case-insensitive rune comparison using Unicode simple folding.
// This is more efficient than converting to lowercase and comparing.
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

// MatchInternalFold is the Unicode-aware matching algorithm that handles both case-sensitive
// and case-insensitive matching with full UTF-8 support. It uses proper Unicode simple folding
// for case-insensitive comparisons and handles multi-byte UTF-8 sequences correctly.
//
// Unicode capabilities:
//   - Full UTF-8 decoding with utf8.DecodeRune* functions
//   - Proper multi-byte character handling
//   - Unicode simple folding for case-insensitive matching
//   - Support for any Unicode character in patterns and input
//   - Correct character width calculation for backtracking
//
// The algorithm supports:
//   - `*`: Matches any sequence of characters (greedy with backtracking)
//   - `?`: Matches zero or one character (with backtracking for both options)
//   - `.`: Matches any single character except newline
//   - `[abc]`: Character classes with full Unicode support (always case-sensitive)
//   - `\x`: Escape sequences for literal characters
//
// The fold parameter controls case-insensitive matching using Unicode simple folding.
// Character classes remain case-sensitive even when fold=true to maintain standard
// glob behavior compatibility.
//
// For ASCII-only input, consider using the optimized MatchInternal function in match.go
// for 2-5x better performance.
func MatchInternalFold[T ~string | ~[]byte](pattern, s T, fold bool) (bool, error) {
	pLen, sLen := len(pattern), len(s)

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

	pIdx, sIdx := 0, 0

	// Optimized backtracking: simple state tracking for both wildcards
	starIdx, sTmpIdx := -1, -1     // For * wildcard backtracking
	questionIdx, qTmpIdx := -1, -1 // For ? wildcard backtracking
	qCount, qMatched := 0, 0       // Track ? wildcard limits

	// Star optimization: store literal sequence after * for index-based search
	var starLiteral string
	var starLiteralBytes []byte
	hasStarLiteral := false

	for { // The loop continues as long as there are characters to match or states to backtrack to.
		// Check for success: both pattern and string fully consumed
		if pIdx >= pLen && sIdx >= sLen {
			return true, nil
		}

		// Case 1: `*` wildcard. Optimize consecutive stars and absorb ? wildcards.
		if pIdx < pLen && pattern[pIdx] == wildcardStar {
			// Skip all consecutive * and ? wildcards - * absorbs ? capabilities
			for pIdx < pLen && (pattern[pIdx] == wildcardStar || pattern[pIdx] == wildcardQuestion) {
				pIdx++
			}
			// Save the position after all absorbed wildcards for backtracking
			starIdx = pIdx
			sTmpIdx = sIdx

			// Extract literal sequence after star for optimization (only for case-sensitive)
			hasStarLiteral = false
			if !fold && starIdx < pLen && !IsWildcardByte(pattern[starIdx]) {
				// Find end of literal sequence
				literalEnd := starIdx
				for literalEnd < pLen && !IsWildcardByte(pattern[literalEnd]) {
					literalEnd++
				}

				// Store the literal for fast search during backtracking
				if isString {
					starLiteral = pStr[starIdx:literalEnd]
				} else {
					starLiteralBytes = pBytes[starIdx:literalEnd]
				}
				hasStarLiteral = true
			}
			continue
		}

		// Case 2: `?` wildcard. Optimize consecutive ? wildcards and save state.
		if pIdx < pLen && pattern[pIdx] == wildcardQuestion {
			// Count and skip all consecutive ? wildcards
			qCount = 0
			for pIdx < pLen && pattern[pIdx] == wildcardQuestion {
				qCount++
				pIdx++
			}

			// Save state for backtracking with question count limit
			questionIdx = pIdx
			qTmpIdx = sIdx
			qMatched = 0 // Reset matched count

			// Try matching zero characters first (greedy approach - match as few as possible)
			continue
		}

		// Case 3: We have a potential match (literal, `.`, or end of pattern).
		// If we're at the end of the string, we might still have a match if the rest of the pattern is optional.
		if sIdx == sLen {
			// Consume trailing wildcards that can match an empty string
			for pIdx < pLen && (pattern[pIdx] == wildcardStar || pattern[pIdx] == wildcardQuestion) {
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

					var matches bool
					if fold {
						matches = equalFoldRune(pRune, sRune)
					} else {
						matches = pRune == sRune
					}

					if matches {
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
			// `.` matches any single character except newline with proper UTF-8 decoding
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

				if sRune == '\n' {
					// Character is newline, fall through to backtrack
				} else {
					pIdx++
					sIdx += sRuneWidth
					continue
				}
			}
		} else if pIdx < pLen && pattern[pIdx] == wildcardBracket {
			// Character class matching with proper UTF-8 decoding
			cc, newPIdx, err := NewcharClassFold(pattern, pIdx)
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

				if cc.MatchesWithFold(sRune, fold) {
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

			var matches bool
			if fold {
				matches = equalFoldRune(pRune, sRune)
			} else {
				matches = pRune == sRune
			}

			if matches {
				pIdx += pRuneWidth
				sIdx += sRuneWidth
				continue
			}
		}

		// Case 4: Mismatch or end of pattern. We must backtrack.
		// First, try ? wildcard backtracking (most recent decisions)
		if questionIdx != -1 && qTmpIdx < sLen && qMatched < qCount {
			// Try matching one more character with ? and retry
			var runeWidth int
			if isString {
				_, runeWidth = utf8.DecodeRuneInString(sStr[qTmpIdx:])
			} else {
				_, runeWidth = utf8.DecodeRune(sBytes[qTmpIdx:])
			}
			qTmpIdx += runeWidth
			qMatched++
			pIdx = questionIdx
			sIdx = qTmpIdx
			continue
		}

		// If ? backtracking exhausted, try * wildcard backtracking
		if starIdx != -1 && sTmpIdx < sLen {
			// Reset ? state since we're trying a different path
			questionIdx, qTmpIdx = -1, -1
			qCount, qMatched = 0, 0
			pIdx = starIdx

			// Optimize: use index-based search if we have a literal after *
			if hasStarLiteral {
				// Find next occurrence of the literal sequence
				var nextPos int
				if isString {
					nextPos = strings.Index(sStr[sTmpIdx+1:], starLiteral)
				} else {
					nextPos = bytes.Index(sBytes[sTmpIdx+1:], starLiteralBytes)
				}

				if nextPos == -1 {
					// No more occurrences of the literal - match fails
					return false, nil
				}
				sTmpIdx += nextPos + 1
			} else {
				// Fall back to incremental advancement
				sTmpIdx++
			}

			sIdx = sTmpIdx
			continue
		}

		// No backtracking options left.
		return false, nil
	}
}
