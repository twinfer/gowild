package wildcard

import (
	"strings"
	"testing"
)

// TestMatchFold validates case-insensitive matching with comprehensive test cases
func TestMatchFold(t *testing.T) {
	cases := []struct {
		s       string
		pattern string
		result  bool
	}{
		// Basic case-insensitive matching
		{"", "", true},
		{"HELLO", "hello", true},
		{"hello", "HELLO", true},
		{"Hello", "hELLo", true},
		{"WORLD", "world", true},

		// Case-insensitive with wildcards
		{"HELLO WORLD", "hello*", true},
		{"HELLO WORLD", "*world", true},
		{"HELLO WORLD", "hello*world", true},
		{"Hello Beautiful World", "hello*world", true},

		// Case-insensitive with ? wildcard
		{"HELLO", "hello", true},
		{"HELLO", "hell?", true},
		{"HELLO", "?ello", true},
		{"Hello", "h?llo", true},

		// Case-insensitive with . wildcard (non-whitespace only)
		{"HELLO", "hell.", true},     // . matches 'O'
		{"HELLO", ".ello", true},     // . matches 'H'
		{"Hello", "he.lo", true},     // . matches 'l' case-insensitive
		{"TEST123", "test...", true}, // Multiple . match multiple non-whitespace chars
		// Dot should NOT match whitespace in case-insensitive mode
		{"HELLO WORLD", "hello.world", false},  // . does not match space
		{"Hello\tWorld", "hello.world", false}, // . does not match tab
		{"Test File", "test.file", false},      // . does not match space

		// Unicode case-insensitive
		{"CAFÉ", "café", true},
		{"café", "CAFÉ", true},
		{"Café", "cAfÉ", true},

		// Complex patterns
		{"TEST FILE NAME", "test*file*name", true},
		{"Test File Name", "TEST*FILE*NAME", true},

		// Should not match
		{"HELLO", "goodbye", false},
		{"hello world", "hello universe", false},
	}

	for i, c := range cases {
		result, err := MatchFold(c.pattern, c.s)
		if err != nil {
			t.Errorf("Test %d: Unexpected error: %v; With Pattern: `%s` and String: `%s`", i+1, err, c.pattern, c.s)
			continue
		}
		if c.result != result {
			t.Errorf("Test %d: Expected `%v`, found `%v`; With Pattern: `%s` and String: `%s`", i+1, c.result, result, c.pattern, c.s)
		}
	}
}

// TestMatchFoldByte validates case-insensitive byte slice matching
func TestMatchFoldByte(t *testing.T) {
	cases := []struct {
		s       []byte
		pattern []byte
		result  bool
	}{
		{[]byte("HELLO"), []byte("hello"), true},
		{[]byte("hello"), []byte("HELLO"), true},
		{[]byte("Hello World"), []byte("hello*world"), true},
		{[]byte("TEST FILE"), []byte("test*file"), true},
		{[]byte("hello"), []byte("goodbye"), false},
	}

	for i, c := range cases {
		result, err := MatchFold(c.pattern, c.s)
		if err != nil {
			t.Errorf("Test %d: Unexpected error: %v; With Pattern: `%s` and String: `%s`", i+1, err, c.pattern, c.s)
			continue
		}
		if c.result != result {
			t.Errorf("Test %d: Expected `%v`, found `%v`; With Pattern: `%s` and String: `%s`", i+1, c.result, result, c.pattern, c.s)
		}
	}
}

// TestMatchFoldRune validates case-insensitive Unicode-aware matching
func TestMatchFoldRune(t *testing.T) {
	cases := []struct {
		s       string
		pattern string
		result  bool
	}{
		{"HELLO", "hello", true},
		{"CAFÉ", "café", true},
		{"café", "CAFÉ", true},
		{"Straße", "strasse", false}, // German ß doesn't match ss in this implementation
		{"HELLO WORLD", "hello*world", true},
		{"Café Au Lait", "café*lait", true},
	}

	for i, c := range cases {
		result, err := MatchFold([]rune(c.pattern), []rune(c.s))
		if err != nil {
			t.Errorf("Test %d: Unexpected error: %v; With Pattern: `%s` and String: `%s`", i+1, err, c.pattern, c.s)
			continue
		}
		if c.result != result {
			t.Errorf("Test %d: Expected `%v`, found `%v`; With Pattern: `%s` and String: `%s`", i+1, c.result, result, c.pattern, c.s)
		}
	}
}

// BenchmarkCaseInsensitive tests the EqualFold fast path and general case-insensitive matching
func BenchmarkCaseInsensitive(b *testing.B) {
	// Simple literal cases that should use EqualFold fast path
	testCases := []struct {
		name    string
		pattern string
		text    string
	}{
		{"Simple literal", "HELLO", "hello"},
		{"Unicode", "CAFÉ", "café"},
		{"Complex with wildcards", "HELLO*WORLD", "hello beautiful world"}, // Should use slower path
	}

	for _, tc := range testCases {
		b.Run("MatchFold_"+tc.name, func(b *testing.B) {
			for b.Loop() {
				MatchFold(tc.pattern, tc.text) // Ignoring error for benchmark
			}
		})

		b.Run("MatchFoldByte_"+tc.name, func(b *testing.B) {
			pattern := []byte(tc.pattern)
			text := []byte(tc.text)
			for b.Loop() {
				MatchFold(pattern, text) // Ignoring error for benchmark
			}
		})
	}
}

// FuzzMatchFold provides fuzz testing for case-insensitive matching
func FuzzMatchFold(f *testing.F) {
	// Add seed corpus for case-insensitive patterns
	f.Add("HELLO*")
	f.Add("Test?")
	f.Add("FILE.TXT")
	f.Add("[A-Z]")
	f.Add("café*")
	f.Add("PATTERN.*")

	f.Fuzz(func(t *testing.T, pattern string) {
		// Test 1: Self-matching (case-insensitive)
		matched, err := MatchFold(pattern, pattern)
		if err != nil {
			t.Skipf("Invalid pattern %q: %v", pattern, err)
		}
		if !matched {
			t.Fatalf("Pattern %q does not match itself case-insensitively", pattern)
		}

		// Test 2: Cross-validation with regular Match for case-sensitive patterns
		if !strings.ContainsAny(pattern, "\\") {
			regularMatched, regularErr := Match(pattern, pattern)
			if (err == nil) != (regularErr == nil) {
				t.Errorf("Error consistency failed between Match and MatchFold for %q", pattern)
			}
			if err == nil && matched != regularMatched {
				t.Errorf("Self-match result mismatch between Match (%v) and MatchFold (%v) for %q",
					regularMatched, matched, pattern)
			}
		}

		// Test 3: Case-insensitive property testing
		if len(pattern) > 0 {
			// Test case variations
			upperPattern := strings.ToUpper(pattern)
			lowerPattern := strings.ToLower(pattern)

			// Pattern should match both upper and lower case versions of itself
			if !strings.ContainsAny(pattern, "\\[") { // Skip complex patterns for this test
				if matched, err := MatchFold(pattern, upperPattern); err == nil && !matched {
					t.Errorf("Pattern %q should match its uppercase version %q", pattern, upperPattern)
				}
				if matched, err := MatchFold(pattern, lowerPattern); err == nil && !matched {
					t.Errorf("Pattern %q should match its lowercase version %q", pattern, lowerPattern)
				}
			}

			// Test specific wildcard behaviors case-insensitively
			if pattern == "*" {
				testStrings := []string{"", "HELLO", "hello", "Hello", "测试", "ТЕСТ"}
				for _, s := range testStrings {
					if matched, err := MatchFold(pattern, s); err != nil || !matched {
						t.Errorf("Pattern '*' should match %q case-insensitively, got %v, err: %v", s, matched, err)
					}
				}
			}
		}

		// Test 4: Type consistency for case-insensitive matching
		if !strings.ContainsAny(pattern, "\\") {
			patternBytes := []byte(pattern)
			patternRunes := []rune(pattern)

			testString := "TEST"
			testBytes := []byte(testString)
			testRunes := []rune(testString)

			stringResult, stringErr := MatchFold(pattern, testString)
			byteResult, byteErr := MatchFold(patternBytes, testBytes)
			runeResult, runeErr := MatchFold(patternRunes, testRunes)

			if (stringErr == nil) != (byteErr == nil) || (stringErr == nil) != (runeErr == nil) {
				t.Errorf("Error consistency failed for case-insensitive pattern %q: string err=%v, byte err=%v, rune err=%v",
					pattern, stringErr, byteErr, runeErr)
			}

			if stringErr == nil {
				if stringResult != byteResult {
					t.Errorf("String/byte result mismatch for case-insensitive pattern %q: string=%v, byte=%v",
						pattern, stringResult, byteResult)
				}
				if stringResult != runeResult {
					t.Errorf("String/rune result mismatch for case-insensitive pattern %q: string=%v, rune=%v",
						pattern, stringResult, runeResult)
				}
			}
		}
	})
}
