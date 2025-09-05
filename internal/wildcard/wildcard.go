// Package wildcard contains the core implementation of the wildcard matching logic.
// It is intended for internal use by the parent gowild package.
package wildcard

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"unicode"
)

// ErrBadPattern indicates a pattern was malformed.
var ErrBadPattern = errors.New("syntax error in pattern")

// PatternSegment represents a parsed segment of the pattern
type PatternSegment struct {
	Type     SegmentType
	Literal  []byte
	Position int
}

type SegmentType int

const (
	SegmentLiteral   SegmentType = iota
	SegmentStar                  // *
	SegmentQuestion              // ?
	SegmentDot                   // .
	SegmentCharClass             // [abc] or [!abc]
)

// Match is the internal, generic core matching function that returns errors.
// It acts as a dispatcher, attempting several fast-path optimizations before
// falling back to a full recursive match for complex patterns.
func Match[T ~string | ~[]byte | ~[]rune](pattern, s T) (bool, error) {
	if len(pattern) == 0 {
		return len(s) == 0, nil
	}

	// Fast path for the most common case: a universal wildcard.
	switch p := any(pattern).(type) {
	case string:
		if p == "*" {
			return true, nil
		}
	case []byte:
		if len(p) == 1 && p[0] == '*' {
			return true, nil
		}
	case []rune:
		if len(p) == 1 && p[0] == '*' {
			return true, nil
		}
	}

	// Fast path for patterns without any wildcards.
	if isExactMatch(pattern, s) {
		return true, nil
	}

	// Fast path for simple patterns like "prefix*", "*suffix", or "prefix*suffix".
	if matched, ok := fastPatternMatch(pattern, s); ok {
		return matched, nil
	}

	// Fallback to the full, recursive implementation for complex patterns.
	return matchGenericRecursive(pattern, s)
}

// fastPatternMatch handles common simple patterns (e.g., "prefix*") to avoid
// the overhead of the recursive matcher. It returns (matched, handled) where
// handled indicates whether the pattern could be handled by the fast path.
func fastPatternMatch[T ~string | ~[]byte | ~[]rune](pattern, s T) (bool, bool) {
	// This optimization is only implemented for byte-oriented types.
	switch p := any(pattern).(type) {
	case string:
		str := any(s).(string)
		matched, handled := fastPatternMatchString(p, str)
		return matched, handled
	case []byte:
		bytes := any(s).([]byte)
		matched, handled := fastPatternMatchBytes(p, bytes)
		return matched, handled
	}
	return false, false
}

// fastPatternMatchString implements the fast path logic for strings.
func fastPatternMatchString(pattern, s string) (bool, bool) {
	// Handles "prefix*" if the prefix contains no other wildcards or character classes.
	if prefix, found := strings.CutSuffix(pattern, "*"); found {
		if !strings.ContainsAny(prefix, "*?.[\\") {
			return strings.HasPrefix(s, prefix), true
		}
	}

	// Handles "*suffix" if the suffix contains no other wildcards or character classes.
	if suffix, found := strings.CutPrefix(pattern, "*"); found {
		if !strings.ContainsAny(suffix, "*?.[\\") {
			return strings.HasSuffix(s, suffix), true
		}
	}

	// Handles "prefix*suffix" if the prefix and suffix contain no other wildcards or character classes.
	if prefix, suffix, found := strings.Cut(pattern, "*"); found && prefix != "" && suffix != "" {
		if !strings.ContainsAny(prefix, "*?.[\\") && !strings.ContainsAny(suffix, "*?.[\\") {
			matched := len(s) >= len(prefix)+len(suffix) &&
				strings.HasPrefix(s, prefix) &&
				strings.HasSuffix(s, suffix)
			return matched, true
		}
	}

	return false, false
}

// fastPatternMatchBytes implements the fast path logic for byte slices.
func fastPatternMatchBytes(pattern, s []byte) (bool, bool) {
	// Handles "prefix*" if the prefix contains no other wildcards or character classes.
	if prefix, found := bytes.CutSuffix(pattern, []byte("*")); found {
		if !bytes.ContainsAny(prefix, "*?.[\\") {
			return bytes.HasPrefix(s, prefix), true
		}
	}

	// Handles "*suffix" if the suffix contains no other wildcards or character classes.
	if suffix, found := bytes.CutPrefix(pattern, []byte("*")); found {
		if !bytes.ContainsAny(suffix, "*?.[\\") {
			return bytes.HasSuffix(s, suffix), true
		}
	}

	// Handles "prefix*suffix" if the prefix and suffix contain no other wildcards or character classes.
	if prefix, suffix, found := bytes.Cut(pattern, []byte("*")); found && len(prefix) > 0 && len(suffix) > 0 {
		if !bytes.ContainsAny(prefix, "*?.[\\") && !bytes.ContainsAny(suffix, "*?.[\\") {
			matched := len(s) >= len(prefix)+len(suffix) &&
				bytes.HasPrefix(s, prefix) &&
				bytes.HasSuffix(s, suffix)
			return matched, true
		}
	}

	return false, false
}

// isExactMatch is an optimization that checks if the pattern contains no wildcards
// and, if so, performs a simple equality check.
func isExactMatch[T ~string | ~[]byte | ~[]rune](pattern, s T) bool {
	if len(pattern) != len(s) {
		return false
	}

	// Check if pattern has no wildcards using optimized methods for each type.
	switch p := any(pattern).(type) {
	case string:
		if strings.ContainsAny(p, "*?.[\\") {
			return false
		}
	case []byte:
		if bytes.ContainsAny(p, "*?.[\\") {
			return false
		}
	case []rune:
		if slices.ContainsFunc(p, func(r rune) bool {
			return r == '*' || r == '?' || r == '.' || r == '[' || r == '\\'
		}) {
			return false
		}
	}

	// If no wildcards are found, perform a direct equality comparison.
	return equal(pattern, s)
}

// equal provides a generic way to compare two values of the same supported type.
func equal[T ~string | ~[]byte | ~[]rune](a, b T) bool {
	switch va := any(a).(type) {
	case string:
		return va == any(b).(string)
	case []byte:
		return bytes.Equal(va, any(b).([]byte))
	case []rune:
		return slices.Equal(va, any(b).([]rune))
	}
	return false
}

// matchGenericRecursive dispatches to the appropriate recursive implementation
// based on the type of the pattern and string.
func matchGenericRecursive[T ~string | ~[]byte | ~[]rune](pattern, s T) (bool, error) {
	switch p := any(pattern).(type) {
	case string:
		return matchRecursive(p, any(s).(string), 0, 0)
	case []byte:
		return matchRecursive(p, any(s).([]byte), 0, 0)
	case []rune:
		return matchRecursiveRunes(p, any(s).([]rune), 0, 0)
	}
	// Should never be reached due to generic type constraints.
	return false, nil
}

// matchCharClass matches a character against a character class pattern like [abc] or [!a-z].
// It returns (matched, nextPatternIndex, error).
func matchCharClass[T ~string | ~[]byte](pattern T, char rune, pi int) (bool, int, error) {
	if pi >= len(pattern) || rune(pattern[pi]) != '[' {
		return false, pi, ErrBadPattern
	}

	pi++ // Skip the opening '['
	if pi >= len(pattern) {
		return false, pi, ErrBadPattern
	}

	// Check for negation
	negated := false
	if pi < len(pattern) && (rune(pattern[pi]) == '^' || rune(pattern[pi]) == '!') {
		negated = true
		pi++
		if pi >= len(pattern) {
			return false, pi, ErrBadPattern
		}
	}

	matched := false
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
				return false, pi, ErrBadPattern
			}
			pi++
			if pi >= len(pattern) {
				return false, pi, ErrBadPattern
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
					return false, pi, ErrBadPattern
				}
				pi++
			}
			if pi >= len(pattern) {
				return false, pi, ErrBadPattern
			}
			c2 = rune(pattern[pi])
			pi++

			// Check if char is in range
			if char >= c1 && char <= c2 {
				matched = true
			}
		} else {
			// Single character match
			if char == c1 {
				matched = true
			}
		}
	}

	// Apply negation if needed
	if negated {
		matched = !matched
	}

	return matched, pi, nil
}

// matchRecursive is the core backtracking algorithm for byte-based types (string, []byte).
// It iterates through the pattern and string, handling wildcards as follows:
// - `.` matches any single byte.
// - `?` matches zero or one byte.
// - `*` matches zero or more bytes through recursion.
// - `[abc]` matches any character in the class.
// - `[!abc]` or `[^abc]` matches any character not in the class.
// - `\x` matches the literal character x (except on Windows).
func matchRecursive[T ~string | ~[]byte](pattern, s T, pi, si int) (bool, error) {
	plen, slen := len(pattern), len(s)

	for pi < plen {
		pc := rune(pattern[pi]) // Note: This is a byte cast to rune, not a true rune conversion.

		switch pc {
		case '*':
			// Coalesce consecutive stars into one.
			switch p := any(pattern).(type) {
			case string:
				remaining := p[pi:]
				idx := strings.IndexFunc(remaining, func(r rune) bool { return r != '*' })
				if idx == -1 {
					return true, nil
				} // Pattern ends with stars.
				pi = pi + idx
			case []byte:
				remaining := p[pi:]
				idx := bytes.IndexFunc(remaining, func(r rune) bool { return r != '*' })
				if idx == -1 {
					return true, nil
				} // Pattern ends with stars.
				pi = pi + idx
			}

			// If the star is at the end of the pattern, it's an automatic match.
			if pi == plen {
				return true, nil
			}

			// For a `*`, we try to match the rest of the pattern (pattern[pi:])
			// against every possible suffix of the string (s[si:]).
			for si <= slen {
				if matched, err := matchRecursive(pattern, s, pi, si); err != nil {
					return false, err
				} else if matched {
					return true, nil
				}
				si++
			}
			return false, nil

		case '?':
			// Special rule: `?` followed by `.` must consume one character, as `.` cannot be optional.
			if pi+1 < plen && rune(pattern[pi+1]) == '.' {
				if si < slen {
					pi++
					si++
				} else {
					return false, nil // Not enough characters in string to satisfy `?.`.
				}
			} else {
				// Standard `?` behavior: try to match zero characters first (by advancing pattern),
				// then try to match one character (by advancing both pattern and string).
				if matched, err := matchRecursive(pattern, s, pi+1, si); err != nil {
					return false, err
				} else if matched {
					return true, nil
				}
				if si < slen {
					return matchRecursive(pattern, s, pi+1, si+1)
				}
				return false, nil
			}

		case '.':
			// `.` must match exactly one character.
			if si >= slen {
				return false, nil
			}
			pi++
			si++

		case '[':
			// Character class matching
			if si >= slen {
				return false, nil
			}
			matched, newPi, err := matchCharClass(pattern, rune(s[si]), pi)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil
			}
			pi = newPi
			si++

		case '\\':
			// Escape sequence handling
			if pi+1 >= plen {
				return false, ErrBadPattern // Trailing backslash
			}
			pi++                   // Skip the backslash
			pc = rune(pattern[pi]) // Get the escaped character
			// Fall through to default case for literal match
			fallthrough

		default:
			// Standard character match.
			if si >= slen || rune(s[si]) != pc {
				return false, nil
			}
			pi++
			si++
		}
	}

	// If we have consumed the entire pattern, the match is successful only if
	// we have also consumed the entire string.
	return si == slen, nil
}

// matchRecursiveRunes is the core backtracking algorithm for rune-based matching.
// It is structurally similar to matchRecursive but operates on slices of runes
// to correctly handle multi-byte Unicode characters.
func matchRecursiveRunes(pattern, s []rune, pi, si int) (bool, error) {
	plen, slen := len(pattern), len(s)

	for pi < plen {
		pc := pattern[pi]

		switch pc {
		case '*':
			// Coalesce consecutive stars into one.
			remaining := pattern[pi:]
			idx := slices.IndexFunc(remaining, func(r rune) bool { return r != '*' })
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

		case '?':
			// Context-aware `?` matching for runes.
			if pi+1 < plen {
				nextChar := pattern[pi+1]
				if nextChar == '.' || nextChar == '*' {
					// If the next pattern char needs a character (`.` or `*`), have `?` be greedy
					// and try to match one character first.
					if si < slen {
						if matched, err := matchRecursiveRunes(pattern, s, pi+1, si+1); err != nil {
							return false, err
						} else if matched {
							return true, nil
						}
					}
					// If the greedy match fails, try a lazy match (zero characters).
					return matchRecursiveRunes(pattern, s, pi+1, si)
				}
			}
			// Default `?` behavior: be lazy and try matching zero characters first.
			if matched, err := matchRecursiveRunes(pattern, s, pi+1, si); err != nil {
				return false, err
			} else if matched {
				return true, nil
			}
			if si < slen {
				return matchRecursiveRunes(pattern, s, pi+1, si+1)
			}
			return false, nil

		case '.':
			// `.` must match exactly one rune.
			if si >= slen {
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

// MatchFold provides case-insensitive matching by converting both pattern and
// string to lower case. It contains a fast path for patterns without wildcards.
func MatchFold[T ~string | ~[]byte | ~[]rune](pattern, s T) (bool, error) {
	switch p := any(pattern).(type) {
	case string:
		str := any(s).(string)
		// Fast path: if pattern has no wildcards, use EqualFold (zero allocs).
		if !strings.ContainsAny(p, "*?.") {
			return strings.EqualFold(p, str), nil
		}
		// For complex patterns, fall back to lowercase conversion.
		return Match(strings.ToLower(p), strings.ToLower(str))

	case []byte:
		bytesData := any(s).([]byte)
		// Fast path: if pattern has no wildcards, use EqualFold (zero allocs).
		if !bytes.ContainsAny(p, "*?.") {
			return bytes.EqualFold(p, bytesData), nil
		}
		// For complex patterns, fall back to lowercase conversion.
		return Match(bytes.ToLower(p), bytes.ToLower(bytesData))

	case []rune:
		runes := any(s).([]rune)
		// Fast path: if pattern has no wildcards, use EqualFunc (zero allocs).
		if !slices.ContainsFunc(p, func(r rune) bool {
			return r == '*' || r == '?' || r == '.'
		}) {
			matched := slices.EqualFunc(p, runes, func(a, b rune) bool {
				return unicode.ToLower(a) == unicode.ToLower(b)
			})
			return matched, nil
		}
		// For complex patterns, fall back to lowercase conversion.
		return Match(toLowerRunes(p), toLowerRunes(runes))
	}
	return false, nil
}

// toLowerRunes converts a rune slice to its lower-case equivalent.
func toLowerRunes(r []rune) []rune {
	// Note: This modifies the slice in-place.
	for i, v := range r {
		r[i] = unicode.ToLower(v)
	}
	return r
}
