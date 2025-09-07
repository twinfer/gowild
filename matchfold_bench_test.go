package gowild

import (
	"testing"
)

// Benchmark data for MatchFold performance analysis
var matchFoldCases = []struct {
	pattern string
	input   string
	name    string
}{
	{"hello", "HELLO", "simple_exact"},
	{"Hello*World", "HELLO BEAUTIFUL WORLD", "prefix_suffix"},
	{"*test*", "THIS IS A TEST STRING", "contains"},
	{"file*.txt", "FILE_NAME.TXT", "prefix_wildcard"},
	{"Hello?.txt", "HELLOx.TXT", "question_mark"},
	{"H*l*o", "HELLO", "multiple_wildcards"},
	{"verylongpatternwithmanychars*", "VERYLONGPATTERNWITHMANYCHARSANDMORE", "long_pattern"},
}

func BenchmarkMatchFoldCurrent(b *testing.B) {
	for _, tc := range matchFoldCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				MatchFold(tc.pattern, tc.input)
			}
		})
	}
}

func BenchmarkMatchFoldCurrentWithAllocs(b *testing.B) {
	for _, tc := range matchFoldCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				MatchFold(tc.pattern, tc.input)
			}
		})
	}
}

func BenchmarkMatchFoldOptimized(b *testing.B) {
	for _, tc := range matchFoldCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				MatchFold(tc.pattern, tc.input)
			}
		})
	}
}
