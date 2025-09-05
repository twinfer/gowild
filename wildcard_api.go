// Package gowild provides highly optimized functions for matching strings against
// patterns containing wildcards. It is designed for performance-critical
// applications that require fast pattern matching on ASCII, Unicode, or binary data.
//
// # Supported Wildcards:
//
//   - `*`: Matches any sequence of characters (including zero characters).
//   - `?`: Matches any single character or zero characters (an optional character).
//   - `.`: Matches any single character (the character must be present).
//   - `[abc]`: Matches any character in the set (a, b, or c)
//   - `[!abc]` or `[^abc]`: Matches any character not in the set
//   - `[a-z]`: Matches any character in the range a to z
//   - `\*`, `\?`, `\.`, `\[`: Matches the literal character
//
// # Type Support:
//
// The package supports three input types through Go generics:
//   - `string`: Optimized byte-wise matching for ASCII strings
//   - `[]byte`: Zero-allocation matching for byte slices
//   - `[]rune`: Unicode-aware matching for multi-byte characters
//
// The functions automatically choose the optimal matching strategy based on the input type.
package gowild

import (
	"github.com/twinfer/gowild/internal/wildcard"
)

// ErrBadPattern indicates a pattern was malformed.
var ErrBadPattern = wildcard.ErrBadPattern

// Match returns true if the pattern matches the input data. It supports three types:
//
//   - string: Fast byte-wise matching optimized for ASCII strings
//   - []byte: Zero-allocation matching for byte slices
//   - []rune: Unicode-aware matching for multi-byte characters
//
// The function automatically selects the optimal matching strategy based on the input type.
// For ASCII strings, it uses fast byte-wise operations. For Unicode strings, use []rune type
// to ensure correct matching of multi-byte characters.
//
// Examples:
//
//	Match("hello*", "hello world")           // string matching
//	Match([]byte("*.txt"), []byte("file.txt")) // byte slice matching
//	Match([]rune("café*"), []rune("café au lait")) // Unicode matching
func Match[T ~string | ~[]byte | ~[]rune](pattern, s T) (bool, error) {
	return wildcard.Match(pattern, s)
}

// MatchFold returns true if the pattern matches the input data with case-insensitive
// comparison. Like Match, it supports three types with automatic optimization:
//
//   - string: Fast case-insensitive matching for ASCII strings
//   - []byte: Zero-allocation case-insensitive matching for byte slices
//   - []rune: Unicode-aware case-insensitive matching
//
// For ASCII strings, it uses optimized case-folding. For proper Unicode case-insensitive
// matching, use []rune type.
//
// Examples:
//
//	MatchFold("HELLO*", "hello world")           // ASCII case-insensitive
//	MatchFold([]rune("CAFÉ*"), []rune("café au lait")) // Unicode case-insensitive
func MatchFold[T ~string | ~[]byte | ~[]rune](pattern, s T) (bool, error) {
	return wildcard.MatchFold(pattern, s)
}
