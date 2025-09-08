package wildcard_bench

import (
	"fmt"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/IGLOU-EU/go-wildcard/v2"
	"github.com/twinfer/gowild"
)

var TestSet = []struct {
	pattern string
	input   string
}{
	{"", "These aren't the wildcard you're looking for"},
	{"These aren't the wildcard you're looking for", ""},
	{"*", "These aren't the wildcard you're looking for"},
	{"These aren't the wildcard you're looking for", "These aren't the wildcard you're looking for"},
	{"Th.e * the wildcard you?re looking fo?", "These aren't the wildcard you're looking for"},
	{"Th.e * the wi??.ldcard you?re looking fo?", "These aren't the wildcard you're looking for These aren't the wildcard you're looking for either"},
	{"*🤷🏾‍♂️*", "T🥵🤷🏾‍♂️🥓"},
}

func BenchmarkRegex(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for b.Loop() {

				regexp.MatchString(t.pattern, t.input)
			}
		})
	}
}

func BenchmarkFilepath(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for b.Loop() {

				filepath.Match(t.pattern, t.input)
			}
		})
	}
}

func BenchmarkGoWildcardMatch(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for b.Loop() {

				wildcard.MatchByRune(t.pattern, t.input)
			}
		})
	}
}

func BenchmarkMatch(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for b.Loop() {

				gowild.Match(t.pattern, t.input)
			}
		})
	}
}

func BenchmarkMatchFromByte(b *testing.B) {
	for i, t := range TestSet {
		pattern := []byte(t.pattern)
		input := []byte(t.input)

		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for b.Loop() {

				gowild.Match(pattern, input)
			}
		})
	}
}
