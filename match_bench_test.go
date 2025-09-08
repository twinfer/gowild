package gowild

import (
	"path/filepath"
	"regexp"
	"testing"
)

// Global test cases for fair comparison across all implementations
var commonTestCases = []struct {
	name    string
	pattern string
	text    string
	regex   string // Equivalent regex pattern
}{
	{
		name:    "Simple Suffix",
		pattern: "*.txt",
		text:    "document.txt",
		regex:   `.*\.txt$`,
	},
	{
		name:    "Simple Prefix",
		pattern: "test*",
		text:    "test_file.go",
		regex:   `^test.*`,
	},
	{
		name:    "Contains Pattern",
		pattern: "*user*",
		text:    "get_user_data",
		regex:   `.*user.*`,
	},
	{
		name:    "Complex Multi-Wildcard",
		pattern: "*test*file*",
		text:    "my_test_config_file.json",
		regex:   `.*test.*file.*`,
	},
	{
		name:    "Question Mark Pattern",
		pattern: "file?.txt",
		text:    "file1.txt",
		regex:   `^file.?\.txt$`,
	},
	{
		name:    "Character Class",
		pattern: "[a-z]*.log",
		text:    "server.log",
		regex:   `^[a-z].*\.log$`,
	},
	{
		name:    "Needle in Haystack",
		pattern: "*important*file[0-9]?.log",
		text:    "this is a very long log entry with lots of text and data before we find the important_config_file3.log entry that we are searching for in this haystack of information",
		regex:   `.*important.*file[0-9].?\.log`,
	},
}

// Pre-compiled regex patterns for performance comparison
var compiledRegexes = make([]*regexp.Regexp, len(commonTestCases))

func init() {
	// Pre-compile all regex patterns
	for i, tc := range commonTestCases {
		compiledRegexes[i] = regexp.MustCompile(tc.regex)
	}
}

// BenchmarkGoWild tests gowild performance on common patterns
func BenchmarkGoWild(b *testing.B) {
	for _, tc := range commonTestCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				Match(tc.pattern, tc.text) // Ignoring error for benchmark
			}
		})
	}
}

// BenchmarkFilepath tests path/filepath.Match performance on common patterns
func BenchmarkFilepath(b *testing.B) {
	for _, tc := range commonTestCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				filepath.Match(tc.pattern, tc.text) // Ignoring error for benchmark
			}
		})
	}
}

// BenchmarkRegexCompiled tests pre-compiled regex performance on common patterns
func BenchmarkRegexCompiled(b *testing.B) {
	for i, tc := range commonTestCases {
		b.Run(tc.name, func(b *testing.B) {
			regex := compiledRegexes[i]
			for b.Loop() {
				regex.MatchString(tc.text)
			}
		})
	}
}

// BenchmarkRegexNotCompiled tests regex compilation + matching performance on common patterns
func BenchmarkRegexNotCompiled(b *testing.B) {
	for _, tc := range commonTestCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				regexp.MatchString(tc.regex, tc.text) // Compiles regex each time
			}
		})
	}
}
