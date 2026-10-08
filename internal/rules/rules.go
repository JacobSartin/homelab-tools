// Package rules describes key-order and blank-line rules for YAML mappings.
package rules

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// Spacing sets the blank lines between the entries of a mapping.
type Spacing int

const (
	// Keep leaves blank lines as they are.
	Keep Spacing = iota
	// Compact removes blank lines between entries.
	Compact
	// Separate requires a blank line between entries.
	Separate
)

// Rule applies to the block mappings at a key path.
type Rule struct {
	// Path is dotted; "*" matches any one key or sequence index, and "" is the
	// document root.
	Path string
	// Order lists keys that move to the front, in this order.
	Order []string
	// SortRest sorts keys not named in Order naturally; otherwise they keep
	// their relative order.
	SortRest bool
	Spacing  Spacing
	// BlankAfter lists keys followed by a blank line when another entry comes after them.
	BlankAfter []string
	// Source names where the rule was declared, for explanations.
	Source string
}

// Orders reports whether the rule rearranges keys.
func (r Rule) Orders() bool { return len(r.Order) > 0 || r.SortRest }

// String summarizes what the rule does.
func (r Rule) String() string {
	var parts []string
	if len(r.Order) > 0 {
		parts = append(parts, "order "+strings.Join(r.Order, ", "))
	}
	if r.SortRest {
		parts = append(parts, "sort other keys")
	}
	switch r.Spacing {
	case Compact:
		parts = append(parts, "compact")
	case Separate:
		parts = append(parts, "separate entries")
	}
	if len(r.BlankAfter) > 0 {
		parts = append(parts, "blank line after "+strings.Join(r.BlankAfter, ", "))
	}
	return fmt.Sprintf("%s (%s)", strings.Join(parts, "; "), r.Source)
}

// Match reports whether pattern matches the key path.
func Match(pattern string, path []string) bool {
	var parts []string
	if pattern != "" {
		parts = strings.Split(pattern, ".")
	}
	if len(parts) != len(path) {
		return false
	}
	for i, part := range parts {
		if part != "*" && part != path[i] {
			return false
		}
	}
	return true
}

// First returns the first rule that matches the key path.
func First(set []Rule, path []string) (Rule, bool) {
	i := slices.IndexFunc(set, func(r Rule) bool { return Match(r.Path, path) })
	if i < 0 {
		return Rule{}, false
	}
	return set[i], true
}

// Sorted returns keys in the order the rule requires.
func (r Rule) Sorted(keys []string) []string {
	rank := func(key string) int {
		if i := slices.Index(r.Order, key); i >= 0 {
			return i
		}
		return len(r.Order)
	}
	sorted := slices.Clone(keys)
	slices.SortStableFunc(sorted, func(a, b string) int {
		if d := rank(a) - rank(b); d != 0 || !r.SortRest || rank(a) < len(r.Order) {
			return d
		}
		return NaturalCompare(a, b)
	})
	return sorted
}

// NaturalCompare orders strings by code point, comparing digit runs numerically.
func NaturalCompare(a, b string) int {
	x, y := chunks(a), chunks(b)
	for i := 0; i < min(len(x), len(y)); i++ {
		if x[i] == y[i] {
			continue
		}
		if isDigits(x[i]) && isDigits(y[i]) {
			m, _ := strconv.ParseUint(x[i], 10, 64)
			n, _ := strconv.ParseUint(y[i], 10, 64)
			if m != n {
				if m < n {
					return -1
				}
				return 1
			}
		}
		return strings.Compare(x[i], y[i])
	}
	return len(x) - len(y)
}

func chunks(s string) []string {
	var out []string
	for len(s) > 0 {
		digit := unicode.IsDigit(rune(s[0]))
		i := strings.IndexFunc(s, func(r rune) bool { return unicode.IsDigit(r) != digit })
		if i < 0 {
			i = len(s)
		}
		out = append(out, s[:i])
		s = s[i:]
	}
	return out
}

func isDigits(s string) bool { return s != "" && unicode.IsDigit(rune(s[0])) }
