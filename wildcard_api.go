// Package gowild provides highly optimized functions for matching strings against
// patterns containing wildcards. It is designed for performance-critical
// applications that require fast pattern matching on ASCII, Unicode, or binary data.
//
// # Supported Wildcards:
//
//   - `*`: Matches any sequence of characters (including zero characters).
//   - `?`: Matches exactly one character (any character).
//   - `.`: Matches exactly one non-whitespace character.
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
	"sync"

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
	// Handle  "empty Pattren" case
	if len(pattern) == 0 {
		if len(s) == 0 {
			return true, nil
		} else if len(s) > 0 {
			return false, nil
		}
	}
	// Delegate everything else to internal function
	return wildcard.Match(pattern, s)
}

// MatchFold returns true if the pattern matches the input data with Case-insensitive
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

	// Handle  "empty Pattren" case
	if len(pattern) == 0 {
		if len(s) == 0 {
			return true, nil
		} else if len(s) > 0 {
			return false, nil
		}
	}
	// Delegate everything else to internal function
	return wildcard.MatchFold(pattern, s)
}

// MatchMultiple concurrently matches a single input against multiple patterns(case ensitive).
// It returns a slice of booleans where each element corresponds to the pattern
// at the same index.
//
// If any pattern is malformed, it returns an error. The order of results corresponds to
// the order of input patterns.
//
// Example:
//
//	patterns := []string{"foo*", "Foo*", "baz[0-9]"}
//	matches, err := MatchMultiple(patterns, "foobar")
//	// matches will be [true, false, false]
func MatchMultiple[S ~string | ~[]byte](patterns []S, s S) ([]bool, error) {
	results := make([]bool, len(patterns))
	// Use an error channel to capture an error from any goroutine.
	errChan := make(chan error, 1)

	var wg sync.WaitGroup

	for i, p := range patterns {
		wg.Add(1)
		go func(i int, p S) {
			defer wg.Done()
			match, err := Match(p, s)
			if err != nil {
				// Try to send the error. If the channel is full, that's fine.
				select {
				case errChan <- err:
				default:
				}
				return
			}
			results[i] = match
		}(i, p)
	}

	wg.Wait()
	close(errChan)

	// Check if any of the goroutines reported an error.
	if err, ok := <-errChan; ok {
		return nil, err
	}

	return results, nil
}

// MatchFoldMultiple concurrently matches a single input against multiple patterns(case-insensitive).
// It returns a slice of booleans where each element corresponds to the pattern
// at the same index.
//
// If any pattern is malformed, it returns an error. The order of results corresponds to
// the order of input patterns.
//
// Example:
//
//	patterns := []string{"Foo*", "foo*", "baz[0-9]"}
//	matches, err := MatchMultiple(patterns, "foobar")
//	// matches will be [true, true, false]
func MatchFoldMultiple[S ~string | ~[]byte](patterns []S, s S) ([]bool, error) {
	results := make([]bool, len(patterns))
	// Use an error channel to capture an error from any goroutine.
	errChan := make(chan error, 1)

	var wg sync.WaitGroup

	for i, p := range patterns {
		wg.Add(1)
		go func(i int, p S) {
			defer wg.Done()
			match, err := MatchFold(p, s)
			if err != nil {
				// Try to send the error. If the channel is full, that's fine.
				select {
				case errChan <- err:
				default:
				}
				return
			}
			results[i] = match
		}(i, p)
	}

	wg.Wait()
	close(errChan)

	// Check if any of the goroutines reported an error.
	if err, ok := <-errChan; ok {
		return nil, err
	}

	return results, nil
}
