// Package gowild provides highly optimized functions for matching strings against
// patterns containing wildcards. It is designed for performance-critical
// applications that require fast pattern matching on ASCII, Unicode, or binary data.
//
// # Core Functions:
//
// The package provides two main functions with unified internal implementation:
//   - Match: Case-sensitive wildcard matching
//   - MatchFold: Case-insensitive wildcard matching with Unicode folding
//
// Both functions use the same optimized matching algorithm internally, differing
// only in character comparison logic.
//
// # Supported Wildcards:
//
//   - `*`: Matches any sequence of characters (including zero characters)
//   - `?`: Matches zero or one character (any character)
//   - `.`: Matches exactly one non-whitespace character
//   - `[abc]`: Matches any character in the set (a, b, or c)
//   - `[!abc]` or `[^abc]`: Matches any character not in the set
//   - `[a-z]`: Matches any character in the range a to z
//   - `\*`, `\?`, `\.`, `\[`: Matches the literal character
//
// # Type Support:
//
// The package supports two input types through Go generics:
//   - `string`: Optimized UTF-8 aware matching for strings
//   - `[]byte`: Zero-allocation matching for byte slices
//
// The functions automatically choose the optimal matching strategy based on the input type.
//
// # Character Classes:
//
// Character classes ([abc], [a-z], [!xyz]) are always case-sensitive, even when
// using MatchFold. This maintains compatibility with standard glob behavior.
package gowild

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/twinfer/gowild/internal/wildcard"
)

// ErrBadPattern indicates a pattern was malformed.
var ErrBadPattern = wildcard.ErrBadPattern

// Match returns true if the pattern matches the input data using case-sensitive comparison.
// It supports two types:
//
//   - string: UTF-8 aware string matching with automatic optimization
//   - []byte: Zero-allocation matching for byte slices
//
// The function uses a unified matching algorithm that handles both ASCII and Unicode
// characters efficiently with proper UTF-8 decoding.
//
// Examples:
//
//	Match("hello*", "hello world")           // string matching
//	Match([]byte("*.txt"), []byte("file.txt")) // byte slice matching
//	Match("café*", "café au lait")           // Unicode matching
//	Match("file?.txt", "file.txt")           // ? matches zero characters
//	Match("file?.txt", "fileX.txt")          // ? matches one character
func Match[T ~string | ~[]byte](pattern, s T) (bool, error) {
	return matchWithOptions(pattern, s, false)
}

// MatchFold returns true if the pattern matches the input data using case-insensitive
// comparison with Unicode folding. It supports two types:
//
//   - string: UTF-8 aware case-insensitive matching with Unicode folding
//   - []byte: Zero-allocation case-insensitive matching for byte slices
//
// The function uses the same unified matching algorithm as Match, but applies
// Unicode simple folding for character comparison. Character classes remain
// case-sensitive to maintain standard glob behavior.
//
// Examples:
//
//	MatchFold("HELLO*", "hello world")           // ASCII case-insensitive
//	MatchFold("CAFÉ*", "café au lait")           // Unicode case-insensitive
//	MatchFold("FILE?.TXT", "file.txt")           // ? matches zero characters
//	MatchFold("FILE?.TXT", "fileX.txt")          // ? matches one character
func MatchFold[T ~string | ~[]byte](pattern, s T) (bool, error) {
	return matchWithOptions(pattern, s, true)
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

// matchWithOptions is the unified internal helper that handles both case-sensitive and case-insensitive matching
func matchWithOptions[T ~string | ~[]byte](pattern, s T, fold bool) (bool, error) {
	// Handle empty pattern case
	if len(pattern) == 0 {
		return len(s) == 0, nil
	}

	if pStr, ok := any(pattern).(string); ok {
		str := any(s).(string)

		// single "*" wildcard
		if pStr == "*" {
			return true, nil
		}

		// if there are no wildcards, do a direct comparison
		if !strings.ContainsFunc(pStr, wildcard.IsWildcard) {
			if fold {
				return strings.EqualFold(pStr, str), nil
			} else {
				return pStr == str, nil
			}
		}

		// Use the unified internal matching algorithm
		return wildcard.MatchInternal(pStr, str, fold)
	}

	if pBytes, ok := any(pattern).([]byte); ok {
		sBytes := any(s).([]byte)

		// single "*" wildcard
		if len(pBytes) == 1 && pBytes[0] == '*' {
			return true, nil
		}

		// if there are no wildcards, do a direct comparison
		if !bytes.ContainsFunc(pBytes, wildcard.IsWildcard) {
			if fold {
				return bytes.EqualFold(pBytes, sBytes), nil
			} else {
				return bytes.Equal(pBytes, sBytes), nil
			}
		}

		// Use the unified internal matching algorithm
		return wildcard.MatchInternal(pBytes, sBytes, fold)
	}

	// This should not be reachable due to type constraints
	return false, fmt.Errorf("unsupported type: %T", pattern)
}
