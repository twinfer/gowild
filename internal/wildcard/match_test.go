package wildcard

import (
	"strings"
	"testing"
)

// TestMatch validates the logic of wild card matching for string input,
// it supports '*', '?' and '.' wildcards with various test cases.
func TestMatch(t *testing.T) {
	cases := []struct {
		s       string
		pattern string
		result  bool
	}{
		// --- Empty String cases ---
		{"", "", true},
		{"", "*", true},
		{"", "**", true},
		{"", "?", false},  // ? requires exactly one character
		{"", "??", false}, // ?? requires exactly two characters
		{"", "?*", false}, // ?* requires at least one character
		{"", "*?", false}, // *? requires at least one character
		{"", ".", false},
		{"", ".?", false},
		{"", "?.", false},
		{"", ".*", false},
		{"", "*.", false},
		{"", "*.?", false},
		{"", "?.*", false},

		// --- Single Character cases ---
		{"a", "", false},
		{"a", "a", true},
		{"a", "*", true},
		{"a", "**", true},
		{"a", "?", true},   // ? matches exactly one character
		{"a", "??", false}, // ?? requires exactly two characters
		{"a", ".", true},
		{"a", ".?", false}, // .? requires exactly two characters
		{"a", "?.", false}, // ?. requires exactly two characters
		{"a", ".*", true},
		{"a", "*.", true},
		{"a", "*.?", false}, // *.? requires at least two characters
		{"ax", "?.*", true}, // ? matches 'a', . matches 'x', * matches empty

		// --- Basic Functionality Tests ---
		{"hello world", "hello world", true},
		{"hello", "world", false},
		{"HELLO WORLD", "HELLO WORLD", true},
		{"HELLO", "hello", false},
		{"😊", "😊", true},
		{"😊", "👍", false},
		{"a long string with many unicode chars 👨‍👩‍👧‍👦", "a long string with many unicode chars 👨‍👩‍👧‍👦", true},
		{"a", "b", false},

		// --- Star Wildcard Tests ---
		{"file.txt", "file.*", true},
		{"file.txt", "*.txt", true},
		{"file.txt", "*.*", true},
		{"file.txt", "f*.t", true},

		// --- Question Mark Wildcard Tests ---
		{"cat", "c?t", true},       // ? matches 'a'
		{"caat", "c?t", false},     // caat has 4 chars, c?t expects 3
		{"cats", "cat?", true},     // ? matches 's'
		{"cuts", "c?ts", true},     // ? matches 'u'
		{"cts", "c?ts", false},     // cts has 3 chars, c?ts expects 4
		{"caats", "c??ts", true},   // ?? matches 'aa'
		{"cuts", "c??ts", false},   // cuts has 4 chars, c??ts expects 5
		{"cabats", "c???ts", true}, // ??? matches 'aba'
		{"caats", "c???ts", false}, // caats has 5 chars, c???ts expects 6
		{"caats", "c?t?s", false},  // would need 'c[a]t[a]s' structure
		{"ca ts", "c?t?s", false},  // space doesn't match structure

		// --- Dot Wildcard Tests ---
		{"cat", "c.t", true},
		{"caat", "c..t", true},
		{"ct", "c.t", false},
		{"cats", ".ats", true},

		// --- Complex Combination Test ---
		{"The quick brown 🦊 is named 'Fred'.", "The quick?brown 🦊 is named '*.'.", true},
		{"The quick brown 🦊 is named 'George'.", "The quick?brown 🦊 is named '*.'.", true},

		// --- Advanced wildcard combinations ---
		{"axc", "a.c", true},
		{"abc", "a?c", true}, // ? must match exactly one character
		{"ac", "a?c", false}, // ac has 2 chars, a?c expects 3
		{"abbc", "a*c", true},
		{"axbyc", "a*b*c", true},
		{"axbyc", "a.b.c", true},
		{"axbyc", "a?b?c", true},
		{"axbyc", "a*b?c", true},
		{"axbyc", "a?b*c", true},
		{"axbyc", "a.b*c", true},
		{"axbyc", "a*b.c", true},
		{"axbyc", "a.b?c", true},
		{"axbyc", "a?b.c", true},

		// --- Consecutive and redundant wildcards ---
		{"longstring", "long**string", true},
		{"longstring", "long***string", true},

		// --- Character class tests ---
		// Basic character sets
		{"a", "[abc]", true},
		{"b", "[abc]", true},
		{"c", "[abc]", true},
		{"d", "[abc]", false},

		// Negated character sets
		{"a", "[!abc]", false},
		{"d", "[!abc]", true},
		{"a", "[^abc]", false},
		{"d", "[^abc]", true},

		// Character ranges
		{"b", "[a-z]", true},
		{"A", "[a-z]", false},
		{"1", "[0-9]", true},
		{"a", "[0-9]", false},
		{"5", "[0-9a-f]", true},
		{"b", "[0-9a-f]", true},
		{"g", "[0-9a-f]", false},

		// Special characters in character classes
		{"]", "[]]", true},
		{"-", "[-]", true},
		{"a", "[-az]", true},
		{"-", "[-az]", true},
		{"z", "[-az]", true},
		{"b", "[-az]", false},

		// Complex character classes
		{"A", "[A-Za-z]", true},
		{"z", "[A-Za-z]", true},
		{"5", "[A-Za-z]", false},
		{"F", "[0-9A-Fa-f]", true},
		{"g", "[0-9A-Fa-f]", false},
		{"_", "[a-zA-Z0-9_]", true},
		{"!", "[a-zA-Z0-9_]", false},

		// Character classes with special positions
		{"a]", "[a]]", true}, // [a]] = character class [a] + literal ']'
		{"a", "[a]]", false}, // [a]] does not match just 'a' (needs 'a]')
		{"]", "[a]]", false}, // [a]] does not match just ']' (needs 'a]')
		{"-", "[a-]", true},
		{"a", "[a-]", true},
		{"b", "[a-]", false},
		{"-", "[-a]", true},
		{"a", "[-a]", true},
		{"b", "[-a]", false},

		// Multiple ranges in one class
		{"5", "[0-359]", true}, // Fixed: 0-3 and 5-9 ranges
		{"4", "[0-359]", false},
		{"A", "[A-CF-H]", true},
		{"D", "[A-CF-H]", false},

		// Negated complex classes
		{"4", "[!0-359]", true},
		{"5", "[!0-359]", false},
		{"D", "[^A-CF-H]", true},
		{"A", "[^A-CF-H]", false},

		// Character classes in patterns with wildcards
		{"abc", "[a-z]*", true},
		{"123", "[a-z]*", false},
		{"a1b", "[a-z]*[0-9]*[a-z]", true},
		{"ab1", "[a-z]*[0-9]*[a-z]", false},
		{"test123", "*[0-9]", true},
		{"testABC", "*[0-9]", false},

		// Empty matches with character classes
		{"", "[a-z]*", false}, // [a-z]* requires at least one char from [a-z]
		{"", "[a-z]", false},
		{"", "[a-z]?", false}, // [a-z]? requires at least one char from [a-z]

		// --- Escape sequence tests ---
		{"a*", "a\\*", true},         // Literal asterisk
		{"a?", "a\\?", true},         // Literal question mark
		{"a.", "a\\.", true},         // Literal dot
		{"a[", "a\\[", true},         // Literal opening bracket
		{"a*b", "a\\*b", true},       // Literal asterisk in middle
		{"*start", "\\*start", true}, // Literal asterisk at start
		{"end*", "end\\*", true},     // Literal asterisk at end
		{"a?b", "a\\?b", true},       // Literal question mark in middle
		{"?start", "\\?start", true}, // Literal question mark at start
		{"end?", "end\\?", true},     // Literal question mark at end
		{"a.b", "a\\.b", true},       // Literal dot in middle
		{".start", "\\.start", true}, // Literal dot at start
		{"end.", "end\\.", true},     // Literal dot at end
		{"a[b", "a\\[b", true},       // Literal bracket in middle
		{"[start", "\\[start", true}, // Literal bracket at start
		{"end[", "end\\[", true},     // Literal bracket at end
		{"a\\", "a\\\\", true},       // Pattern a\\ matches string a\
		{"\\test", "\\\\test", true}, // Pattern \\test matches string \test
		{"test\\", "test\\\\", true}, // Pattern test\\ matches string test\
		// Mixed escape sequences
		{"*?.[", "\\*\\?\\.\\[", true},
		{"test*file?.txt[0]", "test\\*file\\?\\.txt\\[0]", true},
		// Escape sequences that don't match
		{"ab", "a\\*", false}, // Literal * doesn't match b
		{"ab", "a\\?", false}, // Literal ? doesn't match b
		{"ab", "a\\.", false}, // Literal . doesn't match b
		{"ab", "a\\[", false}, // Literal [ doesn't match b

		// --- Dot wildcard tests (matches non-whitespace characters) ---
		{"a", ".", true},             // . matches single non-whitespace char
		{"1", ".", true},             // . matches digit
		{"_", ".", true},             // . matches underscore
		{"-", ".", true},             // . matches hyphen
		{"@", ".", true},             // . matches symbol
		{"hello", "hell.", true},     // . at end matches 'o'
		{"hello", ".ello", true},     // . at start matches 'h'
		{"hello", "he.lo", true},     // . in middle matches 'l'
		{"test123", "test...", true}, // Multiple . match multiple non-whitespace
		// Dot should NOT match whitespace
		{" ", ".", false},                     // . does not match space
		{"\t", ".", false},                    // . does not match tab
		{"\n", ".", false},                    // . does not match newline
		{"\r", ".", false},                    // . does not match carriage return
		{"hello world", "hello.world", false}, // . does not match space between words
		{"a\tb", "a.b", false},                // . does not match tab
		{"", ".", false},                      // . does not match empty (no char to match)
		{"ab", ".", false},                    // . matches exactly one char, not two
		// Mixed patterns with dot
		{"file1.txt", "file.", false},     // . doesn't match '1' because string is longer
		{"file1.txt", "file.*txt", true},  // . matches '1', * matches '.', txt matches 'txt'
		{"user_name", "user.name", true},  // . matches '_'
		{"user name", "user.name", false}, // . doesn't match ' '

		// --- Greediness and backtracking cases ---
		{"ababa", "a*a", true},           // * should match "bab"
		{"abab", "a*b", true},            // * should match "ba"
		{"aaab", "*ab", true},            // * should match "aa"
		{"mississippi", "m*i*i", true},   // First * is "ississ", second * is ""
		{"mississippi", "m*iss*i", true}, // First * is "", second * is "iss"
		{"ab", "a*b", true},
		{"aab", "a*b", true},
		{"aaab", "a*b", true},

		// --- Patterns ending in wildcards ---
		{"abc", "abc*", true},
		{"abcd", "abc?", true}, // ? requires exactly one more character
		{"abc", "abc?", false}, // abc has 3 chars, abc? expects 4
		{"abc", "abc.", false},
		{"abc", "ab.", true},

		// --- More failing cases ---
		{"axbyc", "a.b-c", false},
		{"axbyc", "a?b-c", false},
		{"ab", "a.b", false},
		{"a", "a.", false},
	}

	for i, c := range cases {
		result, err := Match(c.pattern, c.s)
		if err != nil {
			t.Errorf("Test %d: Unexpected error: %v; With Pattern: `%s` and String: `%s`", i+1, err, c.pattern, c.s)
			continue
		}
		if c.result != result {
			t.Errorf("Test %d: Expected `%v`, found `%v`; With Pattern: `%s` and String: `%s`", i+1, c.result, result, c.pattern, c.s)
		}
	}
}

// TestMatchErrors validates error handling for malformed patterns
func TestMatchErrors(t *testing.T) {
	cases := []struct {
		pattern string
		s       string
		desc    string
	}{
		// Invalid character class ranges (these should actually error)
		{"[z-a]", "b", "invalid range (z-a) should error"},
		{"[9-0]", "5", "invalid numeric range (9-0) should error"},
	}

	for i, c := range cases {
		_, err := Match(c.pattern, c.s)
		if err == nil {
			t.Errorf("Test %d: Expected error for pattern '%s', but got none. %s", i+1, c.pattern, c.desc)
		}
		if err != nil && err != ErrBadPattern {
			t.Errorf("Test %d: Expected ErrBadPattern, got %v for pattern '%s'", i+1, err, c.pattern)
		}
	}
}

// TestMatchFromByte validates byte slice matching with the same test cases
func TestMatchFromByte(t *testing.T) {
	cases := []struct {
		s       []byte
		pattern []byte
		result  bool
	}{
		{[]byte(""), []byte(""), true},
		{[]byte(""), []byte("*"), true},
		{[]byte(""), []byte("**"), true},
		{[]byte(""), []byte("?"), false}, // ? requires exactly one character
		{[]byte(""), []byte("."), false},

		{[]byte("a"), []byte(""), false},
		{[]byte("a"), []byte("a"), true},
		{[]byte("a"), []byte("*"), true},
		{[]byte("a"), []byte("?"), true},
		{[]byte("a"), []byte("."), true},

		{[]byte("match the exact string"), []byte("match the exact string"), true},
		{[]byte("do not match a different string"), []byte("this is a different string"), false},
		{[]byte("match an emoji 😃"), []byte("match an emoji 😃"), true},

		{[]byte("match a string with a *"), []byte("match a string *"), true},
		{[]byte("match a string with a * at the beginning"), []byte("* at the beginning"), true},
		{[]byte("match a string with two *"), []byte("match * with *"), true},

		{[]byte("match a string with a ?"), []byte("match ? string with a ?"), true},
		{[]byte("match a string with a ? at the beginning"), []byte("?atch a string with a ? at the beginning"), true},

		{[]byte("match a string with a ."), []byte("match . string with a ."), true},
		{[]byte("match a string with a . at the beginning"), []byte(".atch a string with a . at the beginning"), true},
	}

	for i, c := range cases {
		result, err := Match(c.pattern, c.s)
		if err != nil {
			t.Errorf("Test %d: Unexpected error: %v; With Pattern: `%s` and String: `%s`", i+1, err, c.pattern, c.s)
			continue
		}
		if c.result != result {
			t.Errorf("Test %d: Expected `%v`, found `%v`; With Pattern: `%s` and String: `%s`", i+1, c.result, result, c.pattern, c.s)
		}
	}
}

// TestMatchByRune validates Unicode-aware rune matching
func TestMatchByRune(t *testing.T) {
	cases := []struct {
		s       string
		pattern string
		result  bool
	}{
		{"", "", true},
		{"", "*", true},
		{"", "?", false}, // ? requires exactly one character
		{"", ".", false},

		{"a", "", false},
		{"a", "a", true},
		{"a", "*", true},
		{"a", "?", true},
		{"a", ".", true},

		// Unicode and emoji test cases - important for rune matching
		{"café", "café", true},
		{"café", "caf?", true}, // ? matches 'é'
		{"café", "ca*", true},
		{"café", "c.f.", true},
		{"🌟", "🌟", true},
		{"🌟", "?", true},
		{"🌟", "*", true},
		{"🌟", ".", true},
		{"🌟hello", "?hello", true},
		{"🌟hello", "*hello", true},
		{"🌟hello", ".hello", true},

		// Complex Unicode sequences
		{"🌅☕️📰", "🌅☕️📰", true},
		{"🌅☕️📰", "🌅*", true},
		{"🌅☕️📰", "*📰", true},
		{"🌅☕️📰", "?☕️?", true},
		{"🌅☕️📰", "....", true}, // 4 Unicode characters should need 4 dots

		{"match an emoji 😃", "match an emoji ?", true},
		{"match an emoji 😃", "match * emoji ?", true},
		{"do not match because of different emoji 😃", "do not match because of different emoji 😄", false},

		// --- Unicode characters mixed with wildcards ---
		{"café", "caf?", true}, // ? matches 'é'
		{"café", "caf.", true},
		{"café", "c.f.", true},
		{"你好世界", "你好*", true},
		{"你好世界", "你好.界", true},
		{"你好世界", "你好?界", true}, // ? matches '世'
		{"你好世界", "*世界", true},
		{"你好世界X", "你好世界?", true}, // ? matches 'X'
		{"你好世界", "你好世界?", false}, // Missing character for ?
		{"你好世界", "你好世界.", false},
	}

	for i, c := range cases {
		result, err := Match([]rune(c.pattern), []rune(c.s))
		if err != nil {
			t.Errorf("Test %d: Unexpected error: %v; With Pattern: `%s` and String: `%s`", i+1, err, c.pattern, c.s)
			continue
		}
		if c.result != result {
			t.Errorf("Test %d: Expected `%v`, found `%v`; With Pattern: `%s` and String: `%s`", i+1, c.result, result, c.pattern, c.s)
		}
	}
}

// TestMatchEdgeCases validates patterns that could cause  issues
func TestMatchEdgeCases(t *testing.T) {
	cases := []struct {
		s       string
		pattern string
		result  bool
		desc    string
	}{
		// Many consecutive wildcards (should be optimized)
		{"tes", "???", true, "three ? wildcards match three chars"},
		{"test", "????", true, "four ? wildcards match four chars"},
		{"tests", "?????", true, "five ? wildcards match five chars"},
		{"", "???", false, "three ? wildcards cannot match empty string"},
		{"a", "???", false, "three ? wildcards cannot match single char"},

		// Complex backtracking scenarios
		{"aaaaab", "a*a*a*b", true, "multiple * with repeating chars"},
		{"aaaaaab", "a*a*a*a*b", true, "many * with repeating chars"},
		{"abcdefg", "a*b*c*d*e*f*g", true, "alternating chars and *"},

		// Patterns that could cause exponential complexity (if not optimized)
		{"axbxaxbxaxbxaxbx", "a?b?a?b?a?b?a?b?", true, "alternating ? patterns"},
		{"", "?????????", false, "nine ? wildcards cannot match empty string"},
		{"x", "?????????", false, "nine ? wildcards cannot match single char"},
		{"123456789", "?????????", true, "nine ? wildcards match nine chars"},

		// Deep nesting with character classes
		{"a1b2c3", "[a-z][0-9][a-z][0-9][a-z][0-9]", true, "alternating char classes"},
		{"abcdef", "[a-z]*[a-z]*[a-z]*", true, "multiple char class wildcards"},

		// Long literal strings with wildcards
		{"verylongstringwithmanychars", "very*string*many*", true, "long string with wildcards"},
		{"verylongstringwithmanychars", "*very*string*many*chars", true, "long string with leading wildcard"},

		// Edge cases with dots
		{"abcdef", "......", true, "exact length with dots"},
		{"abc", "......", false, "insufficient length with dots"},

		// Mixed wildcard stress test
		{"complex_test_string_123", "*test*string*[0-9]*", true, "mixed wildcards stress test"},
		{"", "*?*?*?*", false, "empty string cannot match patterns with ? wildcards"},
		{"abcd", "*?*?*?*", true, "four chars can match *?*?*?* pattern"},

		// Patterns that previously caused issues
		{"mississippi", "m*i*s*s*i*p*p*i", true, "complex pattern with many *"},
	}

	for i, c := range cases {
		result, err := Match(c.pattern, c.s)
		if err != nil {
			t.Errorf("Test %d (%s): Unexpected error: %v; With Pattern: `%s` and String: `%s`", i+1, c.desc, err, c.pattern, c.s)
			continue
		}
		if c.result != result {
			t.Errorf("Test %d (%s): Expected `%v`, found `%v`; With Pattern: `%s` and String: `%s`", i+1, c.desc, c.result, result, c.pattern, c.s)
		}
	}
}

// FuzzMatch provides fuzz testing for string matching robustness
func FuzzMatchM(f *testing.F) {
	// Add seed corpus with known wildcard patterns
	f.Add("*")
	f.Add("?")
	f.Add(".")
	f.Add("*.txt")
	f.Add("test*")
	f.Add("he?lo")
	f.Add("file[0-9]")
	f.Add("[a-z]")
	f.Add("[!0-9]")
	f.Add("literal*")
	f.Add("prefix*suffix")
	f.Add("a.b")
	f.Add("???")
	f.Add("*.*")

	f.Fuzz(func(t *testing.T, pattern string) {
		// Test 1: Self-matching (original test)
		matched, err := Match(pattern, pattern)
		if err != nil {
			// Some strings are not valid patterns (e.g., trailing backslash)
			// Skip these cases as they're expected to fail
			t.Skipf("Invalid pattern %q: %v", pattern, err)
		}
		if !matched {
			t.Fatalf("Pattern %q does not match itself", pattern)
		}

		// Test 2: Property-based testing for wildcard behavior
		if len(pattern) > 0 {
			// Test star wildcard expansion property
			if pattern == "*" {
				// * should match any string
				testStrings := []string{"", "hello", "test123", "🌟"}
				for _, s := range testStrings {
					if matched, err := Match(pattern, s); err != nil || !matched {
						t.Errorf("Pattern '*' should match %q, got %v, err: %v", s, matched, err)
					}
				}
			}

			// Test question mark behavior
			if pattern == "?" {
				// ? should match any single character
				if matched, err := Match(pattern, "a"); err != nil || !matched {
					t.Errorf("Pattern '?' should match single char 'a', got %v, err: %v", matched, err)
				}
				if matched, err := Match(pattern, "ab"); err != nil || matched {
					t.Errorf("Pattern '?' should not match 'ab', got %v, err: %v", matched, err)
				}
			}

			// Test dot wildcard (non-whitespace only)
			if pattern == "." {
				// . should match non-whitespace characters
				if matched, err := Match(pattern, "a"); err != nil || !matched {
					t.Errorf("Pattern '.' should match 'a', got %v, err: %v", matched, err)
				}
				if matched, err := Match(pattern, " "); err != nil || matched {
					t.Errorf("Pattern '.' should not match space, got %v, err: %v", matched, err)
				}
			}
		}

		// Test 3: Cross-type consistency
		if !strings.ContainsAny(pattern, "\\") { // Skip patterns with escapes for byte/rune tests
			patternBytes := []byte(pattern)
			patternRunes := []rune(pattern)

			testString := "test"
			testBytes := []byte(testString)
			testRunes := []rune(testString)

			stringResult, stringErr := Match(pattern, testString)
			byteResult, byteErr := Match(patternBytes, testBytes)
			runeResult, runeErr := Match(patternRunes, testRunes)

			if (stringErr == nil) != (byteErr == nil) || (stringErr == nil) != (runeErr == nil) {
				t.Errorf("Error consistency failed for pattern %q: string err=%v, byte err=%v, rune err=%v",
					pattern, stringErr, byteErr, runeErr)
			}

			if stringErr == nil && stringResult != byteResult {
				t.Errorf("String/byte result mismatch for pattern %q: string=%v, byte=%v",
					pattern, stringResult, byteResult)
			}

			if stringErr == nil && stringResult != runeResult {
				t.Errorf("String/rune result mismatch for pattern %q: string=%v, rune=%v",
					pattern, stringResult, runeResult)
			}
		}
	})
}

// FuzzMatchFromByte provides fuzz testing for byte slice matching robustness
func FuzzMatchFromByte(f *testing.F) {
	// Add seed corpus for byte patterns
	f.Add("*.bin")
	f.Add("file?")
	f.Add("[0-9]")
	f.Add("data.*")

	f.Fuzz(func(t *testing.T, s string) {
		b := []byte(s)

		// Test 1: Self-matching
		matched, err := Match(b, b)
		if err != nil {
			// Skip invalid patterns
			t.Skipf("Invalid pattern %q: %v", s, err)
		}
		if !matched {
			t.Fatalf("Byte pattern %q does not match itself", s)
		}

		// Test 2: Consistency with string version
		if len(s) > 0 && !strings.ContainsAny(s, "\\") {
			stringMatched, stringErr := Match(s, s)
			if (err == nil) != (stringErr == nil) {
				t.Errorf("Error consistency failed between byte and string for %q", s)
			}
			if err == nil && matched != stringMatched {
				t.Errorf("Result mismatch between byte (%v) and string (%v) for %q", matched, stringMatched, s)
			}
		}

		// Test 3: Negative cases for specific patterns
		if s == "*" {
			// Test that * matches various byte sequences
			testCases := [][]byte{nil, {}, []byte("hello"), {0, 1, 2, 255}}
			for _, testBytes := range testCases {
				if matched, err := Match(b, testBytes); err != nil || !matched {
					t.Errorf("Pattern '*' should match byte sequence %v, got %v, err: %v", testBytes, matched, err)
				}
			}
		}
	})
}

// FuzzMatchByRune provides fuzz testing for rune matching robustness
func FuzzMatchByRune(f *testing.F) {
	// Add Unicode-aware seed corpus
	f.Add("café*")
	f.Add("test.unicode")
	f.Add("[α-ω]")
	f.Add("🌟*")
	f.Add("файл?")

	f.Fuzz(func(t *testing.T, s string) {
		runes := []rune(s)

		// Test 1: Self-matching
		matched, err := Match(runes, runes)
		if err != nil {
			// Skip invalid patterns
			t.Skipf("Invalid pattern %q: %v", s, err)
		}
		if !matched {
			t.Fatalf("Rune pattern %q does not match itself", s)
		}

		// Test 2: Unicode handling verification
		if len(runes) > 0 {
			// Test that rune matching properly handles multi-byte characters
			for i, r := range runes {
				singleRune := []rune{r}
				questionPattern := []rune("?")

				if r != '*' && r != '?' && r != '.' && r != '[' && r != '\\' {
					// Non-wildcard character should match itself with ?
					if matched, err := Match(questionPattern, singleRune); err != nil || !matched {
						t.Errorf("Pattern '?' should match rune %q at position %d, got %v, err: %v",
							string(r), i, matched, err)
					}

					// Test . wildcard with Unicode spaces
					dotPattern := []rune(".")
					if r == ' ' || r == '\t' || r == '\n' || r == '\u00A0' { // Various Unicode spaces
						if matched, err := Match(dotPattern, singleRune); err != nil || matched {
							t.Errorf("Pattern '.' should not match whitespace rune %q, got %v, err: %v",
								string(r), matched, err)
						}
					} else {
						if matched, err := Match(dotPattern, singleRune); err != nil || !matched {
							t.Errorf("Pattern '.' should match non-whitespace rune %q, got %v, err: %v",
								string(r), matched, err)
						}
					}
				}
			}
		}

		// Test 3: Consistency with string version for valid UTF-8
		if len(s) > 0 && !strings.ContainsAny(s, "\\") && len([]rune(s)) == len(runes) {
			stringMatched, stringErr := Match(s, s)
			if (err == nil) != (stringErr == nil) {
				t.Errorf("Error consistency failed between rune and string for %q", s)
			}
			if err == nil && matched != stringMatched {
				t.Errorf("Result mismatch between rune (%v) and string (%v) for %q", matched, stringMatched, s)
			}
		}
	})
}

// FuzzMatchNegative tests patterns that should NOT match certain inputs
func FuzzMatchNegative(f *testing.F) {
	// Add seed corpus for negative testing
	f.Add("exact", "different")
	f.Add("?", "")   // ? should not match empty
	f.Add("?", "ab") // ? should not match multiple chars
	f.Add(".", " ")  // . should not match space
	f.Add(".", "\t") // . should not match tab
	f.Add(".", "\n") // . should not match newline
	f.Add("prefix*", "other")
	f.Add("[abc]", "d")
	f.Add("[!xyz]", "x")

	f.Fuzz(func(t *testing.T, pattern, input string) {
		matched, err := Match(pattern, input)

		if err != nil {
			// Skip invalid patterns
			t.Skipf("Invalid pattern %q: %v", pattern, err)
		}

		// Property-based negative testing
		if len(pattern) > 0 && len(input) > 0 {
			// Test that . wildcard does not match whitespace
			if pattern == "." && (input == " " || input == "\t" || input == "\n") {
				if matched {
					t.Errorf("Pattern '.' should not match whitespace %q", input)
				}
			}

			// Test that ? does not match empty or multiple characters
			if pattern == "?" {
				if len(input) != 1 && matched {
					t.Errorf("Pattern '?' should not match input of length %d: %q", len(input), input)
				}
			}

			// Test literal mismatch
			if !strings.ContainsAny(pattern, "*?.\\[") { // Pure literal pattern
				if pattern != input && matched {
					t.Errorf("Literal pattern %q should not match different input %q", pattern, input)
				}
			}

			// Test prefix/suffix constraints
			if strings.HasPrefix(pattern, "prefix") && !strings.HasPrefix(input, "prefix") && matched {
				t.Errorf("Pattern %q starting with 'prefix' should not match %q", pattern, input)
			}
			if strings.HasSuffix(pattern, "suffix") && !strings.HasSuffix(input, "suffix") && matched {
				t.Errorf("Pattern %q ending with 'suffix' should not match %q", pattern, input)
			}
		}

		// Edge case: empty pattern should only match empty input
		if pattern == "" && input != "" && matched {
			t.Errorf("Empty pattern should not match non-empty input %q", input)
		}
	})
}

// FuzzMatchEdgeCases tests edge cases and malformed patterns
func FuzzMatchEdgeCases(f *testing.F) {
	// Add seed corpus for edge cases
	f.Add("\\")    // Trailing backslash
	f.Add("[")     // Unclosed bracket
	f.Add("[]")    // Empty bracket
	f.Add("[a-]")  // Incomplete range
	f.Add("[-z]")  // Range starting with dash
	f.Add("[z-a]") // Invalid range
	f.Add("\\x")   // Escape non-special char
	f.Add("***")   // Multiple stars
	f.Add("???")   // Multiple questions
	f.Add("...")   // Multiple dots
	f.Add("")      // Empty pattern

	f.Fuzz(func(t *testing.T, pattern string) {
		inputs := []string{"", "a", "test", " ", "\t", "\n", "unicode测试", "🌟"}

		for _, input := range inputs {
			matched, err := Match(pattern, input)

			// Test error handling consistency
			if err != nil {
				// Malformed patterns should fail gracefully
				if matched {
					t.Errorf("Malformed pattern %q should not return matched=true with error: %v", pattern, err)
				}
				continue
			}

			// Test basic invariants for valid patterns
			if pattern == "" {
				expected := input == ""
				if matched != expected {
					t.Errorf("Empty pattern with input %q: expected %v, got %v", input, expected, matched)
				}
			}

			if pattern == "*" {
				if !matched {
					t.Errorf("Pattern '*' should always match, failed for input %q", input)
				}
			}

			// Test consecutive wildcard handling
			if strings.Contains(pattern, "***") {
				starPattern := strings.ReplaceAll(pattern, "***", "*")
				starMatched, starErr := Match(starPattern, input)
				if starErr == nil && matched != starMatched {
					t.Errorf("Pattern %q and simplified %q should have same result for %q: %v vs %v",
						pattern, starPattern, input, matched, starMatched)
				}
			}

			// Test Unicode handling
			if strings.Contains(input, "测试") || strings.Contains(input, "🌟") {
				// Verify that byte and rune versions handle Unicode consistently
				if !strings.ContainsAny(pattern, "\\") {
					runeMatched, runeErr := Match([]rune(pattern), []rune(input))
					if (err == nil) != (runeErr == nil) {
						t.Errorf("Unicode consistency: pattern %q, input %q - string err=%v, rune err=%v",
							pattern, input, err, runeErr)
					}
					if err == nil && matched != runeMatched {
						t.Errorf("Unicode consistency: pattern %q, input %q - string=%v, rune=%v",
							pattern, input, matched, runeMatched)
					}
				}
			}
		}
	})
}
