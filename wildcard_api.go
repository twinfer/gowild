/*
Copyright (c) 2025 twinfer.com contact@twinfer.com Copyright (c) 2025 Khalid Daoud mohamed.khalid@gmail.com

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
*/

// Package gowild provides highly optimized functions for matching strings against
// patterns containing wildcards. It features a dual-stream architecture that delivers
// maximum performance for both ASCII-only and Unicode scenarios.
//
// # Performance Architecture:
//
// The package uses two specialized implementations:
//   - ASCII-only stream: Zero UTF-8 overhead, direct byte operations (2-5x faster)
//   - Unicode stream: Full UTF-8 support with case-insensitive matching
//
// # Core Functions:
//
//   - Match: ASCII-optimized case-sensitive wildcard matching
//   - MatchFold: Unicode-aware case-insensitive wildcard matching
//
// The functions automatically route to the appropriate implementation for optimal performance.
//
// # Supported Wildcards:
//
//   - `*`: Matches any sequence of characters (including zero characters)
//   - `?`: Matches zero or one character (any character)
//   - `.`: Matches any single character except newline
//   - `[abc]`: Matches any character in the set (a, b, or c)
//   - `[!abc]` or `[^abc]`: Matches any character not in the set
//   - `[a-z]`: Matches any character in the range a to z
//   - `\*`, `\?`, `\.`, `\[`: Matches the literal character
//
// # Type Support:
//
// The package supports two input types through Go generics:
//   - `string`: Automatic routing to ASCII or Unicode implementation
//   - `[]byte`: Zero-allocation matching for binary data and performance-critical code
//
// # Character Classes:
//
// Character classes ([abc], [a-z], [!xyz]) are always case-sensitive, even when
// using MatchFold. This maintains compatibility with standard glob behavior.
//
// # Performance Guidance:
//
// - Use Match() for ASCII-only patterns when maximum speed is needed
// - Use MatchFold() for Unicode patterns or when case-insensitive matching is required
// - ASCII-only matching provides 2-5x performance improvement over Unicode-aware matching
package gowild

import (
	"sync"

	"github.com/twinfer/gowild/internal/wildcard"
)

// ErrBadPattern indicates a pattern was malformed.
var ErrBadPattern = wildcard.ErrBadPattern

// Match returns true if the pattern matches the input data using case-sensitive comparison.
// It supports two types:
//
//   - string: Optimized ASCII-only string matching for maximum performance
//   - []byte: Zero-allocation matching for byte slices
//
// This function uses an optimized ASCII-only algorithm for maximum performance.
// For Unicode support, use MatchFold instead.
//
// Examples:
//
//	Match("hello*", "hello world")           // ASCII string matching
//	Match([]byte("*.txt"), []byte("file.txt")) // byte slice matching
//	Match("file?.txt", "file.txt")           // ? matches zero characters
//	Match("file?.txt", "fileX.txt")          // ? matches one character
func Match[T ~string | ~[]byte](pattern, s T) (bool, error) {
	return wildcard.MatchInternal(pattern, s)
}

// MatchFold returns true if the pattern matches the input data using case-insensitive
// comparison with Unicode folding. It supports two types:
//
//   - string: UTF-8 aware case-insensitive matching with Unicode folding
//   - []byte: Zero-allocation case-insensitive matching for byte slices
//
// The function uses the Unicode-aware matching algorithm with case folding.
// Character classes remain case-sensitive to maintain standard glob behavior.
//
// Examples:
//
//	MatchFold("HELLO*", "hello world")           // ASCII case-insensitive
//	MatchFold("CAFÉ*", "café au lait")           // Unicode case-insensitive
//	MatchFold("FILE?.TXT", "file.txt")           // ? matches zero characters
//	MatchFold("FILE?.TXT", "fileX.txt")          // ? matches one character
func MatchFold[T ~string | ~[]byte](pattern, s T) (bool, error) {
	return wildcard.MatchInternalFold(pattern, s, true)
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
