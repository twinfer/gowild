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
		return parseCharClassBytes(p, pi)
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

// MatchesFold performs case-insensitive matching against this character class.
func (cc *CharClass) MatchesFold(char rune) bool {
	// Check individual characters with case folding
	matched := slices.ContainsFunc(cc.Chars, func(c rune) bool {
		return equalFoldRune(c, char)
	})

	// Check ranges with case folding if not matched yet
	if !matched {
		matched = slices.ContainsFunc(cc.Ranges, func(r CharRange) bool {
			return charInRangeFold(char, r.Start, r.End)
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

			// Add range
			cc.Ranges = append(cc.Ranges, CharRange{Start: c1, End: c2})
		} else {
			// Add single character
			cc.Chars = append(cc.Chars, c1)
		}
	}

	return cc, pi, nil
}

// parseCharClassBytes parses a character class from a byte slice pattern.
func parseCharClassBytes(pattern []byte, pi int) (*CharClass, int, error) {
	// Convert to string for consistent parsing logic
	return parseCharClassString(string(pattern), pi)
}

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
	// Enhanced fast paths based on Go stdlib patterns

	// 1. Handle "*word*" (contains pattern)
	if len(pattern) >= 2 && pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		middle := pattern[1 : len(pattern)-1]
		if !strings.ContainsAny(middle, "*?.[\\") {
			return strings.Contains(s, middle), true
		}
	}

	// 2. Handle "prefix*" if the prefix contains no other wildcards or character classes.
	if prefix, found := strings.CutSuffix(pattern, "*"); found {
		if !strings.ContainsAny(prefix, "*?.[\\") {
			return strings.HasPrefix(s, prefix), true
		}
	}

	// 3. Handle "*suffix" if the suffix contains no other wildcards or character classes.
	if suffix, found := strings.CutPrefix(pattern, "*"); found {
		if !strings.ContainsAny(suffix, "*?.[\\") {
			return strings.HasSuffix(s, suffix), true
		}
	}

	// 4. Handle "prefix*suffix" if the prefix and suffix contain no other wildcards or character classes.
	if prefix, suffix, found := strings.Cut(pattern, "*"); found && prefix != "" && suffix != "" {
		if !strings.ContainsAny(prefix, "*?.[\\") && !strings.ContainsAny(suffix, "*?.[\\") {
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
		if !bytes.ContainsAny(middle, "*?.[\\") {
			return bytes.Contains(s, middle), true
		}
	}

	// 2. Handle "prefix*" if the prefix contains no other wildcards or character classes.
	if prefix, found := bytes.CutSuffix(pattern, []byte("*")); found {
		if !bytes.ContainsAny(prefix, "*?.[\\") {
			return bytes.HasPrefix(s, prefix), true
		}
	}

	// 3. Handle "*suffix" if the suffix contains no other wildcards or character classes.
	if suffix, found := bytes.CutPrefix(pattern, []byte("*")); found {
		if !bytes.ContainsAny(suffix, "*?.[\\") {
			return bytes.HasSuffix(s, suffix), true
		}
	}

	// 4. Handle "prefix*suffix" if the prefix and suffix contain no other wildcards or character classes.
	if prefix, suffix, found := bytes.Cut(pattern, []byte("*")); found && len(prefix) > 0 && len(suffix) > 0 {
		if !bytes.ContainsAny(prefix, "*?.[\\") && !bytes.ContainsAny(suffix, "*?.[\\") {
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

// matchGenericRecursive dispatches to the appropriate iterative or recursive implementation
// based on the type of the pattern and string. Uses iterative algorithm for simple patterns,
// falls back to recursive for complex multi-wildcard patterns.
func matchGenericRecursive[T ~string | ~[]byte | ~[]rune](pattern, s T) (bool, error) {
	switch p := any(pattern).(type) {
	case string:
		// Use iterative algorithm for better performance
		return iterativeMatch(p, any(s).(string))
	case []byte:
		// Use iterative algorithm for better performance
		return iterativeMatch(p, any(s).([]byte))
	case []rune:
		// Keep recursive implementation for runes (Unicode correctness)
		return matchRecursiveRunes(p, any(s).([]rune), 0, 0)
	}
	// Should never be reached due to generic type constraints.
	return false, nil
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

// MatchFold provides case-insensitive matching without allocating new strings.
// It contains a fast path for patterns without wildcards using EqualFold, and
// implements zero-allocation case-insensitive matching for wildcard patterns.
func MatchFold[T ~string | ~[]byte | ~[]rune](pattern, s T) (bool, error) {
	switch p := any(pattern).(type) {
	case string:
		str := any(s).(string)
		// Fast path: if pattern has no wildcards, use EqualFold (zero allocs).
		if !strings.ContainsAny(p, "*?.[\\") {
			return strings.EqualFold(p, str), nil
		}
		// Use iterative algorithm for better performance
		return iterativeMatchFold(p, str)

	case []byte:
		bytesData := any(s).([]byte)
		// Fast path: if pattern has no wildcards, use EqualFold (zero allocs).
		if !bytes.ContainsAny(p, "*?.[\\") {
			return bytes.EqualFold(p, bytesData), nil
		}
		// Use iterative algorithm for better performance
		return iterativeMatchFold(p, bytesData)

	case []rune:
		runes := any(s).([]rune)
		// Fast path: if pattern has no wildcards, use EqualFunc (zero allocs).
		if !slices.ContainsFunc(p, func(r rune) bool {
			return r == '*' || r == '?' || r == '.' || r == '[' || r == '\\'
		}) {
			matched := slices.EqualFunc(p, runes, func(a, b rune) bool {
				return unicode.ToLower(a) == unicode.ToLower(b)
			})
			return matched, nil
		}
		// Zero-allocation case-insensitive wildcard matching.
		return matchFoldRecursiveRunes(p, runes, 0, 0)
	}
	return false, nil
}

// matchFoldRecursiveRunes implements case-insensitive wildcard matching for rune slices.
func matchFoldRecursiveRunes(pattern, s []rune, pi, si int) (bool, error) {
	plen, slen := len(pattern), len(s)

	for pi < plen {
		pc := pattern[pi]

		switch pc {
		case '*':
			remaining := pattern[pi:]
			idx := slices.IndexFunc(remaining, func(r rune) bool { return r != '*' })
			if idx == -1 {
				return true, nil
			}
			pi = pi + idx

			for si <= slen {
				if matched, err := matchFoldRecursiveRunes(pattern, s, pi, si); err != nil {
					return false, err
				} else if matched {
					return true, nil
				}
				si++
			}
			return false, nil

		case '?':
			if pi+1 < plen {
				nextChar := pattern[pi+1]
				if nextChar == '.' || nextChar == '*' {
					if si < slen {
						if matched, err := matchFoldRecursiveRunes(pattern, s, pi+1, si+1); err != nil {
							return false, err
						} else if matched {
							return true, nil
						}
					}
					return matchFoldRecursiveRunes(pattern, s, pi+1, si)
				}
			}
			if matched, err := matchFoldRecursiveRunes(pattern, s, pi+1, si); err != nil {
				return false, err
			} else if matched {
				return true, nil
			}
			if si < slen {
				return matchFoldRecursiveRunes(pattern, s, pi+1, si+1)
			}
			return false, nil

		case '.':
			if si >= slen {
				return false, nil
			}
			pi++
			si++

		default:
			if si >= slen {
				return false, nil
			}
			if !equalFoldRune(pc, s[si]) {
				return false, nil
			}
			pi++
			si++
		}
	}

	return si == slen, nil
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

// charInRangeFold checks if a character falls within a case-insensitive range.
func charInRangeFold(char, start, end rune) bool {
	// Simple case: direct range check
	if char >= start && char <= end {
		return true
	}

	// Check case variants of the character
	for f := unicode.SimpleFold(char); f != char; f = unicode.SimpleFold(f) {
		if f >= start && f <= end {
			return true
		}
	}

	return false
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
		// Case 1: `*` wildcard. Save its state and continue.
		if pIdx < pLen && pattern[pIdx] == '*' {
			starIdx = pIdx
			sTmpIdx = sIdx
			pIdx++
			continue
		}

		// Case 2: `?` wildcard optimization - handle consecutive `?` as a group
		if pIdx < pLen && pattern[pIdx] == '?' {
			// Count consecutive `?` wildcards
			qStart := pIdx
			for pIdx < pLen && pattern[pIdx] == '?' {
				pIdx++
			}
			qCount := pIdx - qStart

			// Try all possibilities from consuming 0 to min(qCount, remaining chars) characters
			// Push states in reverse order so we try the "consume more" options first
			maxConsume := min(sLen-sIdx, qCount)

			for consume := maxConsume; consume >= 1; consume-- {
				backtrackStack = append(backtrackStack, backtrackState{pIdx: pIdx, sIdx: sIdx + consume})
			}
			// The current path tries consuming 0 characters (continue immediately)
			continue
		}

		// Case 3: We have a potential match (literal, `.`, or end of pattern).
		// If we're at the end of the string, we might still have a match if the rest of the pattern is optional.
		if sIdx == sLen {
			// Consume trailing optional wildcards
			for pIdx < pLen && (pattern[pIdx] == '?' || pattern[pIdx] == '*') {
				pIdx++
			}
			if pIdx == pLen {
				return true, nil // Matched successfully
			}
			// Mismatch, fall through to backtrack
		} else if pIdx < pLen && (pattern[pIdx] == '.' || pattern[pIdx] == s[sIdx]) {
			// Standard character match.
			pIdx++
			sIdx++
			continue
		} else if pIdx < pLen && pattern[pIdx] == '[' {
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
		} else if pIdx < pLen && pattern[pIdx] == '\\' {
			// Escape sequence handling
			if pIdx+1 >= pLen {
				return false, ErrBadPattern // Trailing backslash
			}
			// Check if escaped character matches
			if pattern[pIdx+1] == s[sIdx] {
				pIdx += 2 // Skip backslash and escaped character
				sIdx++
				continue
			}
			// Escaped character doesn't match, fall through to backtrack
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

// iterativeMatchFold case-insensitive version of the iterative matching algorithm.
// It handles backtracking for both `*` and `?`, with Unicode case folding.
func iterativeMatchFold[T ~string | ~[]byte](pattern, s T) (bool, error) {
	pLen, sLen := len(pattern), len(s)
	pIdx, sIdx := 0, 0

	type backtrackState struct {
		pIdx int
		sIdx int
	}
	backtrackStack := []backtrackState{}

	starIdx, sTmpIdx := -1, -1

	for { // The loop continues as long as there are characters to match or states to backtrack to.
		// Case 1: `*` wildcard. Save its state and continue.
		if pIdx < pLen && pattern[pIdx] == '*' {
			starIdx = pIdx
			sTmpIdx = sIdx
			pIdx++
			continue
		}

		// Case 2: `?` wildcard optimization - handle consecutive `?` as a group
		if pIdx < pLen && pattern[pIdx] == '?' {
			// Count consecutive `?` wildcards
			qStart := pIdx
			for pIdx < pLen && pattern[pIdx] == '?' {
				pIdx++
			}
			qCount := pIdx - qStart

			// Try all possibilities from consuming 0 to min(qCount, remaining chars) characters
			// Push states in reverse order so we try the "consume more" options first
			maxConsume := min(sLen-sIdx, qCount)

			for consume := maxConsume; consume >= 1; consume-- {
				backtrackStack = append(backtrackStack, backtrackState{pIdx: pIdx, sIdx: sIdx + consume})
			}
			// The current path tries consuming 0 characters (continue immediately)
			continue
		}

		// Case 3: We have a potential match (literal, `.`, or end of pattern).
		// If we're at the end of the string, we might still have a match if the rest of the pattern is optional.
		if sIdx == sLen {
			// Consume trailing optional wildcards
			for pIdx < pLen && (pattern[pIdx] == '?' || pattern[pIdx] == '*') {
				pIdx++
			}
			if pIdx == pLen {
				return true, nil // Matched successfully
			}
			// Mismatch, fall through to backtrack
		} else if pIdx < pLen && (pattern[pIdx] == '.' || equalFoldRune(rune(pattern[pIdx]), rune(s[sIdx]))) {
			// Case-insensitive character match.
			pIdx++
			sIdx++
			continue
		} else if pIdx < pLen && pattern[pIdx] == '[' {
			// Character class matching (case-insensitive)
			cc, newPIdx, err := NewCharClass(pattern, pIdx)
			if err != nil {
				return false, err
			}
			if cc.MatchesFold(rune(s[sIdx])) {
				pIdx = newPIdx
				sIdx++
				continue
			}
			// Character class doesn't match, fall through to backtrack
		} else if pIdx < pLen && pattern[pIdx] == '\\' {
			// Escape sequence handling (case-insensitive)
			if pIdx+1 >= pLen {
				return false, ErrBadPattern // Trailing backslash
			}
			// Check if escaped character matches (case-insensitive)
			if equalFoldRune(rune(pattern[pIdx+1]), rune(s[sIdx])) {
				pIdx += 2 // Skip backslash and escaped character
				sIdx++
				continue
			}
			// Escaped character doesn't match, fall through to backtrack
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
