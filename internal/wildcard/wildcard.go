// Package wildcard contains the core implementation of the wildcard matching logic.
// It is intended for internal use by the parent gowild package.
package wildcard

import (
	"bytes"
	"slices"
	"strings"
	"unicode"
)

// PatternSegment represents a parsed segment of the pattern
type PatternSegment struct {
	Type     SegmentType
	Literal  []byte
	Position int
}

type SegmentType int

const (
	SegmentLiteral  SegmentType = iota
	SegmentStar                 // *
	SegmentQuestion             // ?
	SegmentDot                  // .
)

// Match is the internal, generic core matching function.
// It acts as a dispatcher, attempting several fast-path optimizations before
// falling back to a full recursive match for complex patterns.
func Match[T ~string | ~[]byte | ~[]rune](pattern, s T) bool {
	if len(pattern) == 0 {
		return len(s) == 0
	}

	// Fast path for the most common case: a universal wildcard.
	switch p := any(pattern).(type) {
	case string:
		if p == "*" {
			return true
		}
	case []byte:
		if len(p) == 1 && p[0] == '*' {
			return true
		}
	case []rune:
		if len(p) == 1 && p[0] == '*' {
			return true
		}
	}

	// Fast path for patterns without any wildcards.
	if isExactMatch(pattern, s) {
		return true
	}

	// Fast path for simple patterns like "prefix*", "*suffix", or "prefix*suffix".
	if fastPatternMatch(pattern, s) {
		return true
	}

	// Fallback to the full, recursive implementation for complex patterns.
	return matchGenericRecursive(pattern, s)
}

// fastPatternMatch handles common simple patterns (e.g., "prefix*") to avoid
// the overhead of the recursive matcher. It only supports string and []byte types.
func fastPatternMatch[T ~string | ~[]byte | ~[]rune](pattern, s T) bool {
	// This optimization is only implemented for byte-oriented types.
	switch p := any(pattern).(type) {
	case string:
		str := any(s).(string)
		return fastPatternMatchString(p, str)
	case []byte:
		bytes := any(s).([]byte)
		return fastPatternMatchBytes(p, bytes)
	}
	return false
}

// fastPatternMatchString implements the fast path logic for strings.
func fastPatternMatchString(pattern, s string) bool {
	// Handles "prefix*" if the prefix contains no other wildcards.
	if prefix, found := strings.CutSuffix(pattern, "*"); found {
		if !strings.ContainsAny(prefix, "*?.") {
			return strings.HasPrefix(s, prefix)
		}
	}

	// Handles "*suffix" if the suffix contains no other wildcards.
	if suffix, found := strings.CutPrefix(pattern, "*"); found {
		if !strings.ContainsAny(suffix, "*?.") {
			return strings.HasSuffix(s, suffix)
		}
	}

	// Handles "prefix*suffix" if the prefix and suffix contain no other wildcards.
	if prefix, suffix, found := strings.Cut(pattern, "*"); found && prefix != "" && suffix != "" {
		if !strings.ContainsAny(prefix, "*?.") && !strings.ContainsAny(suffix, "*?.") {
			return len(s) >= len(prefix)+len(suffix) &&
				strings.HasPrefix(s, prefix) &&
				strings.HasSuffix(s, suffix)
		}
	}

	return false
}

// fastPatternMatchBytes implements the fast path logic for byte slices.
func fastPatternMatchBytes(pattern, s []byte) bool {
	// Handles "prefix*" if the prefix contains no other wildcards.
	if prefix, found := bytes.CutSuffix(pattern, []byte("*")); found {
		if !bytes.ContainsAny(prefix, "*?.") {
			return bytes.HasPrefix(s, prefix)
		}
	}

	// Handles "*suffix" if the suffix contains no other wildcards.
	if suffix, found := bytes.CutPrefix(pattern, []byte("*")); found {
		if !bytes.ContainsAny(suffix, "*?.") {
			return bytes.HasSuffix(s, suffix)
		}
	}

	// Handles "prefix*suffix" if the prefix and suffix contain no other wildcards.
	if prefix, suffix, found := bytes.Cut(pattern, []byte("*")); found && len(prefix) > 0 && len(suffix) > 0 {
		if !bytes.ContainsAny(prefix, "*?.") && !bytes.ContainsAny(suffix, "*?.") {
			return len(s) >= len(prefix)+len(suffix) &&
				bytes.HasPrefix(s, prefix) &&
				bytes.HasSuffix(s, suffix)
		}
	}

	return false
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
		if strings.ContainsAny(p, "*?.") {
			return false
		}
	case []byte:
		if bytes.ContainsAny(p, "*?.") {
			return false
		}
	case []rune:
		if slices.ContainsFunc(p, func(r rune) bool {
			return r == '*' || r == '?' || r == '.'
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
func matchGenericRecursive[T ~string | ~[]byte | ~[]rune](pattern, s T) bool {
	switch p := any(pattern).(type) {
	case string:
		return matchRecursive(p, any(s).(string), 0, 0)
	case []byte:
		return matchRecursive(p, any(s).([]byte), 0, 0)
	case []rune:
		return matchRecursiveRunes(p, any(s).([]rune), 0, 0)
	}
	// Should never be reached due to generic type constraints.
	return false
}

// matchRecursive is the core backtracking algorithm for byte-based types (string, []byte).
// It iterates through the pattern and string, handling wildcards as follows:
// - `.` matches any single byte.
// - `?` matches zero or one byte.
// - `*` matches zero or more bytes through recursion.
func matchRecursive[T ~string | ~[]byte](pattern, s T, pi, si int) bool {
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
				if idx == -1 { return true } // Pattern ends with stars.
				pi = pi + idx
			case []byte:
				remaining := p[pi:]
				idx := bytes.IndexFunc(remaining, func(r rune) bool { return r != '*' })
				if idx == -1 { return true } // Pattern ends with stars.
				pi = pi + idx
			}

			// If the star is at the end of the pattern, it's an automatic match.
			if pi == plen {
				return true
			}

			// For a `*`, we try to match the rest of the pattern (pattern[pi:])
			// against every possible suffix of the string (s[si:]).
			for si <= slen {
				if matchRecursive(pattern, s, pi, si) {
					return true
				}
				si++
			}
			return false

		case '?':
			// Special rule: `?` followed by `.` must consume one character, as `.` cannot be optional.
			if pi+1 < plen && rune(pattern[pi+1]) == '.' {
				if si < slen {
					pi++
					si++
				} else {
					return false // Not enough characters in string to satisfy `?.`.
				}
			} else {
				// Standard `?` behavior: try to match zero characters first (by advancing pattern),
				// then try to match one character (by advancing both pattern and string).
				if matchRecursive(pattern, s, pi+1, si) {
					return true
				}
				if si < slen {
					return matchRecursive(pattern, s, pi+1, si+1)
				}
				return false
			}

		case '.':
			// `.` must match exactly one character.
			if si >= slen {
				return false
			}
			pi++
			si++

		default:
			// Standard character match.
			if si >= slen || rune(s[si]) != pc {
				return false
			}
			pi++
			si++
		}
	}

	// If we have consumed the entire pattern, the match is successful only if
	// we have also consumed the entire string.
	return si == slen
}

// matchRecursiveRunes is the core backtracking algorithm for rune-based matching.
// It is structurally similar to matchRecursive but operates on slices of runes
// to correctly handle multi-byte Unicode characters.
func matchRecursiveRunes(pattern, s []rune, pi, si int) bool {
	plen, slen := len(pattern), len(s)

	for pi < plen {
		pc := pattern[pi]

		switch pc {
		case '*':
			// Coalesce consecutive stars into one.
			remaining := pattern[pi:]
			idx := slices.IndexFunc(remaining, func(r rune) bool { return r != '*' })
			if idx == -1 { return true } // Pattern ends with stars.
			pi = pi + idx

			// For a `*`, try to match the rest of the pattern against every suffix.
			for si <= slen {
				if matchRecursiveRunes(pattern, s, pi, si) {
					return true
				}
				si++
			}
			return false

		case '?':
			// Context-aware `?` matching for runes.
			if pi+1 < plen {
				nextChar := pattern[pi+1]
				if nextChar == '.' || nextChar == '*' {
					// If the next pattern char needs a character (`.` or `*`), have `?` be greedy
					// and try to match one character first.
					if si < slen {
						if matchRecursiveRunes(pattern, s, pi+1, si+1) {
							return true
						}
					}
					// If the greedy match fails, try a lazy match (zero characters).
					return matchRecursiveRunes(pattern, s, pi+1, si)
				}
			}
			// Default `?` behavior: be lazy and try matching zero characters first.
			if matchRecursiveRunes(pattern, s, pi+1, si) {
				return true
			}
			if si < slen {
				return matchRecursiveRunes(pattern, s, pi+1, si+1)
			}
			return false

		case '.':
			// `.` must match exactly one rune.
			if si >= slen {
				return false
			}
			pi++
			si++

		default:
			// Standard rune match.
			if si >= slen || s[si] != pc {
				return false
			}
			pi++
			si++
		}
	}

	return si == slen
}

// MatchFold provides case-insensitive matching by converting both pattern and
// string to lower case. It contains a fast path for patterns without wildcards.
func MatchFold[T ~string | ~[]byte | ~[]rune](pattern, s T) bool {
	switch p := any(pattern).(type) {
	case string:
		str := any(s).(string)
		// Fast path: if pattern has no wildcards, use EqualFold (zero allocs).
		if !strings.ContainsAny(p, "*?.") {
			return strings.EqualFold(p, str)
		}
		// For complex patterns, fall back to lowercase conversion.
		return Match(strings.ToLower(p), strings.ToLower(str))

	case []byte:
		bytesData := any(s).([]byte)
		// Fast path: if pattern has no wildcards, use EqualFold (zero allocs).
		if !bytes.ContainsAny(p, "*?.") {
			return bytes.EqualFold(p, bytesData)
		}
		// For complex patterns, fall back to lowercase conversion.
		return Match(bytes.ToLower(p), bytes.ToLower(bytesData))

	case []rune:
		runes := any(s).([]rune)
		// Fast path: if pattern has no wildcards, use EqualFunc (zero allocs).
		if !slices.ContainsFunc(p, func(r rune) bool {
			return r == '*' || r == '?' || r == '.'
		}) {
			return slices.EqualFunc(p, runes, func(a, b rune) bool {
				return unicode.ToLower(a) == unicode.ToLower(b)
			})
		}
		// For complex patterns, fall back to lowercase conversion.
		return Match(toLowerRunes(p), toLowerRunes(runes))
	}
	return false
}

// toLowerRunes converts a rune slice to its lower-case equivalent.
func toLowerRunes(r []rune) []rune {
	// Note: This modifies the slice in-place.
	for i, v := range r {
		r[i] = unicode.ToLower(v)
	}
	return r
}