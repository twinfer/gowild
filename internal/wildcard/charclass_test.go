/*
Copyright (c) 2025 twinfer.com contact@twinfer.com Copyright (c) 2025 Khalid Daoud mohamed.khalid@gmail.com

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
*/
package wildcard

import (
	"testing"
)

func TestCharClassParsing(t *testing.T) {
	tests := []struct {
		pattern  string
		pos      int
		expected string
		negated  bool
		chars    []rune
		ranges   int // number of ranges expected
	}{
		{"[abc]", 0, "[abc]", false, []rune{'a', 'b', 'c'}, 0},
		{"[!abc]", 0, "[!abc]", true, []rune{'a', 'b', 'c'}, 0},
		{"[a-z]", 0, "[a-z]", false, []rune{}, 1},
		{"[a-zA-Z0-9]", 0, "[a-zA-Z0-9]", false, []rune{}, 3},
		{"[!a-z]", 0, "[!a-z]", true, []rune{}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			cc, newPos, err := NewCharClass(tt.pattern, tt.pos)
			if err != nil {
				t.Fatalf("NewCharClass failed: %v", err)
			}

			if cc.Negated != tt.negated {
				t.Errorf("Expected negated=%v, got %v", tt.negated, cc.Negated)
			}

			if len(cc.Chars) != len(tt.chars) {
				t.Errorf("Expected %d chars, got %d", len(tt.chars), len(cc.Chars))
			}

			if len(cc.Ranges) != tt.ranges {
				t.Errorf("Expected %d ranges, got %d", tt.ranges, len(cc.Ranges))
			}

			if newPos != len(tt.pattern) {
				t.Errorf("Expected position %d, got %d", len(tt.pattern), newPos)
			}
		})
	}
}

func TestCharClassMatching(t *testing.T) {
	tests := []struct {
		pattern string
		char    rune
		match   bool
	}{
		{"[abc]", 'a', true},
		{"[abc]", 'd', false},
		{"[!abc]", 'a', false},
		{"[!abc]", 'd', true},
		{"[a-z]", 'a', true},
		{"[a-z]", 'z', true},
		{"[a-z]", 'A', false},
		{"[A-Z]", 'A', true},
		{"[A-Z]", 'a', false},
		{"[0-9]", '5', true},
		{"[0-9]", 'a', false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			cc, _, err := NewCharClass(tt.pattern, 0)
			if err != nil {
				t.Fatalf("NewCharClass failed: %v", err)
			}

			result := cc.MatchesWithFold(tt.char, false)
			if result != tt.match {
				t.Errorf("Expected %v for char '%c' in pattern %s, got %v",
					tt.match, tt.char, tt.pattern, result)
			}
		})
	}
}

func TestCharClassAlwaysCaseSensitive(t *testing.T) {
	tests := []struct {
		pattern string
		char    rune
		match   bool
	}{
		{"[abc]", 'A', false}, // Character classes are always case-sensitive
		{"[ABC]", 'a', false}, // Character classes are always case-sensitive
		{"[a-z]", 'A', false}, // Character classes are always case-sensitive
		{"[A-Z]", 'a', false}, // Character classes are always case-sensitive
		{"[!abc]", 'A', true}, // Negated: 'A' is not in [abc] (case-sensitive)
		{"[!ABC]", 'a', true}, // Negated: 'a' is not in [ABC] (case-sensitive)
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			cc, _, err := NewCharClass(tt.pattern, 0)
			if err != nil {
				t.Fatalf("NewCharClass failed: %v", err)
			}

			result := cc.MatchesWithFold(tt.char, false)
			if result != tt.match {
				t.Errorf("Expected %v for char '%c' in pattern %s (should be case-sensitive), got %v",
					tt.match, tt.char, tt.pattern, result)
			}
		})
	}
}

func TestCharClassErrorCases(t *testing.T) {
	tests := []struct {
		pattern     string
		description string
	}{
		{"[abc", "unclosed character class"},
		{"[a-z", "unclosed character class with range"},
		{"[!abc", "unclosed negated character class"},
		{"[z-a]", "invalid range (z > a)"},
		{"[0-\\x8a-0]", "invalid range with escape sequence"},
		{"[0-\\x8a-0", "unclosed class with invalid range and escape"},
		{"[\\", "incomplete escape sequence"},
		{"[a-\\", "incomplete escape in range"},
		{"[", "empty character class start"},
		{"[]", "empty character class"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			// Test with string
			_, _, err := NewCharClass(tt.pattern, 0)
			if err == nil {
				t.Errorf("Expected error for pattern %q (%s), got nil", tt.pattern, tt.description)
			}

			// Test with []byte to ensure consistency
			_, _, err = NewCharClass([]byte(tt.pattern), 0)
			if err == nil {
				t.Errorf("Expected error for []byte pattern %q (%s), got nil", tt.pattern, tt.description)
			}
		})
	}
}

func TestCharClassConsistency(t *testing.T) {
	// Test patterns that should behave identically for string and []byte
	patterns := []string{
		"[abc]",
		"[a-z]",
		"[!a-z]",
		"[0-9]",
		"[A-Z]",
		"[a-zA-Z0-9]",
		"[abc",        // unclosed
		"[z-a]",       // invalid range
		"[0-\\x8a-0]", // escape with invalid range
		"[0-\xe8-0]",  // invalid UTF-8 byte in pattern
	}

	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			stringClass, stringPos, stringErr := NewCharClass(pattern, 0)
			byteClass, bytePos, byteErr := NewCharClass([]byte(pattern), 0)

			// Errors should be consistent
			if (stringErr == nil) != (byteErr == nil) {
				t.Errorf("Error consistency failed for pattern %q: string err=%v, byte err=%v",
					pattern, stringErr, byteErr)
			}

			// If no errors, positions should match
			if stringErr == nil && byteErr == nil {
				if stringPos != bytePos {
					t.Errorf("Position mismatch for pattern %q: string pos=%d, byte pos=%d",
						pattern, stringPos, bytePos)
				}

				// Character classes should be equivalent
				if stringClass.Negated != byteClass.Negated {
					t.Errorf("Negated mismatch for pattern %q: string=%v, byte=%v",
						pattern, stringClass.Negated, byteClass.Negated)
				}

				if len(stringClass.Chars) != len(byteClass.Chars) {
					t.Errorf("Chars length mismatch for pattern %q: string=%d, byte=%d",
						pattern, len(stringClass.Chars), len(byteClass.Chars))
				}

				if len(stringClass.Ranges) != len(byteClass.Ranges) {
					t.Errorf("Ranges length mismatch for pattern %q: string=%d, byte=%d",
						pattern, len(stringClass.Ranges), len(byteClass.Ranges))
				}
			}
		})
	}
}
