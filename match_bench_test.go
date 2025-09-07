package gowild

import "testing"

// BenchmarkPatterns tests the performance of  pattern matching
func BenchmarkPatterns(b *testing.B) {
	testCases := []struct {
		name    string
		pattern string
		text    string
	}{
		// Single character patterns  with IndexByte
		{"Single char *x", "*x", "this is a test with x at the end"},
		{"Single char x*", "x*", "x marks the spot for treasure hunting"},

		// Suffix patterns  with LastIndex
		{"Star suffix short", "*test", "this is a test"},
		{"Star suffix long", "*optimization", "this is a much longer string that ends with optimization"},

		// Contains patterns  with Contains
		{"Contains short", "*test*", "this test is good"},
		{"Contains long", "*optimization*", "the performance optimization here is excellent"},

		// Multi-segment patterns using bidirectional search
		{"Two segments", "hello*world", "hello beautiful world"},
		{"Three segments", "start*middle*end", "start of the middle section leads to end"},
		{"Four segments", "a*b*c*d", "a very long string with b in the middle and c near the d"},

		// Complex patterns that should benefit from reverse matching
		{"Reverse terminal", "*suffix.txt", "a very long filename that ends with suffix.txt"},
		{"Complex reverse", "*final*result", "this is the final answer and result"},

		// Case-insensitive
		{"Case fold prefix", "HELLO*", "hello world"},
		{"Case fold suffix", "*WORLD", "hello world"},
		{"Case fold contains", "*TEST*", "this test works"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				Match(tc.pattern, tc.text) // Ignoring error for benchmark
			}
		})
	}
}

// BenchmarkBytes tests  specific to byte slice operations
func BenchmarkBytes(b *testing.B) {
	testCases := []struct {
		name    string
		pattern string
		text    string
	}{
		// IndexByte vs IndexFunc comparison
		{"Bytes single char *x", "*x", "this is a test with x at the end"},
		{"Bytes star suffix", "*bytes", " for bytes"},
		{"Bytes contains", "*optimize*", "bytes optimize performance"},
		{"Bytes multi-segment", "start*middle*end", "start of middle leads to end"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			pattern := []byte(tc.pattern)
			text := []byte(tc.text)
			for b.Loop() {
				Match(pattern, text) // Ignoring error for benchmark
			}
		})
	}
}

// BenchmarkCaseFold tests case-insensitive
func BenchmarkCaseFold(b *testing.B) {
	testCases := []struct {
		name    string
		pattern string
		text    string
	}{
		// Direct EqualFold vs ToLower+Match
		{"Fold no wildcards", "HELLO", "hello"},
		{"Fold prefix star", "HELLO*", "hello world"},
		{"Fold star suffix", "*WORLD", "hello world"},
		{"Fold contains", "*TEST*", "this test works"},
		{"Fold complex", "START*middle*END", "start of middle leads to end"},
	}

	for _, tc := range testCases {
		b.Run(tc.name+" String", func(b *testing.B) {
			for b.Loop() {
				MatchFold(tc.pattern, tc.text) // Ignoring error for benchmark
			}
		})

		b.Run(tc.name+" Bytes", func(b *testing.B) {
			pattern := []byte(tc.pattern)
			text := []byte(tc.text)
			for b.Loop() {
				MatchFold(pattern, text) // Ignoring error for benchmark
			}
		})
	}
}

// BenchmarkReverseMatching specifically tests reverse matching
func BenchmarkReverseMatching(b *testing.B) {
	// Create test strings of different lengths to show reverse matching benefits
	longString := "this is a very long string that should benefit from reverse matching when the pattern ends in a specific suffix"

	testCases := []struct {
		name    string
		pattern string
		text    string
	}{
		{"Reverse short", "*suffix", "test suffix"},
		{"Reverse medium", "*matching", "this requires reverse matching"},
		{"Reverse long", "*suffix", longString},
		{"Reverse very long", "*optimization", longString + " optimization"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				Match(tc.pattern, tc.text) // Ignoring error for benchmark
			}
		})
	}
}
