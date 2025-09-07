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

			result := cc.Matches(tt.char)
			if result != tt.match {
				t.Errorf("Expected %v for char '%c' in pattern %s, got %v",
					tt.match, tt.char, tt.pattern, result)
			}
		})
	}
}

func TestCharClassCaseInsensitiveMatching(t *testing.T) {
	tests := []struct {
		pattern string
		char    rune
		match   bool
	}{
		{"[abc]", 'A', true},
		{"[ABC]", 'a', true},
		{"[a-z]", 'A', true},
		{"[A-Z]", 'a', true},
		{"[!abc]", 'A', false},
		{"[!ABC]", 'a', false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			cc, _, err := NewCharClass(tt.pattern, 0)
			if err != nil {
				t.Fatalf("NewCharClass failed: %v", err)
			}

			result := cc.MatchesFold(tt.char)
			if result != tt.match {
				t.Errorf("Expected %v for char '%c' in pattern %s (case insensitive), got %v",
					tt.match, tt.char, tt.pattern, result)
			}
		})
	}
}
