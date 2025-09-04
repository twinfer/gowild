// Package gowild provides highly optimized functions for matching strings against
// patterns containing wildcards. It is designed for performance-critical
// applications that require fast pattern matching on ASCII or byte-oriented strings.
//
// For Unicode-aware matching, see the 'ByRune' variants of the functions.
//
// # Supported Wildcards:
//
//   - `*`: Matches any sequence of characters (including zero characters).
//   - `?`: Matches any single character or zero characters (an optional character).
//   - `.`: Matches any single character (the character must be present).
package gowild

import (
	"github.com/twinfer/gowild/internal/wildcard"
)

// Match returns true if the pattern matches the string s. It is the fastest
// matching function in this package, optimized for performance by operating on bytes.
//
// This function is ideal for ASCII strings or when byte-wise comparison is
// sufficient. It does NOT correctly handle multi-byte Unicode characters.
// For Unicode-aware matching, use MatchByRune.
func Match(pattern, s string) bool {
	return wildcard.Match(pattern, s)
}

// MatchByRune returns true if the pattern matches the string s, with full
// support for Unicode characters. It operates on runes instead of bytes,
// allowing wildcards to correctly match multi-byte characters (e.g., a `.` can
// match `é`).
//
// This function should be used when the input strings may contain non-ASCII
// characters. Note that this correctness comes with a performance cost compared
// to the byte-wise Match function, as it involves converting strings to rune slices.
func MatchByRune(pattern, s string) bool {
	return wildcard.Match([]rune(pattern), []rune(s))
}

// MatchFromByte returns true if the pattern matches the byte slice s.
// It is functionally equivalent to Match but operates directly on byte slices,
// which can prevent string-to-slice conversion allocations in performance-sensitive code.
func MatchFromByte(pattern, s []byte) bool {
	return wildcard.Match(pattern, s)
}

// MatchFold returns true if the pattern matches the string s in a case-insensitive
// manner. It uses simple case-folding and is optimized for ASCII strings.
//
// Like Match, this function operates on bytes and does not correctly handle
// multi-byte Unicode characters. For case-insensitive Unicode matching, use
// MatchFoldRune.
func MatchFold(pattern, s string) bool {
	return wildcard.MatchFold(pattern, s)
}

// MatchFoldRune returns true if the pattern matches the string s with
// case-insensitivity and full Unicode support.
//
// It combines the case-folding logic of MatchFold with the rune-wise matching
// of MatchByRune. It is the most correct but also the most computationally
// expensive matching function in this package.
func MatchFoldRune(pattern, s string) bool {
	return wildcard.MatchFold([]rune(pattern), []rune(s))
}

// MatchFoldByte returns true if the pattern matches the byte slice s with
// case-insensitivity. It is the byte-slice equivalent of MatchFold.
func MatchFoldByte(pattern, s []byte) bool {
	return wildcard.MatchFold(pattern, s)
}