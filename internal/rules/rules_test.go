package rules

import (
	"slices"
	"testing"
)

func TestMatch(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		path    []string
		want    bool
	}{
		{"", nil, true},
		{"", []string{"a"}, false},
		{"a.*.c", []string{"a", "b", "c"}, true},
		{"a.*.c", []string{"a", "c"}, false},
	} {
		if got := Match(tc.pattern, tc.path); got != tc.want {
			t.Errorf("Match(%q, %q) = %v", tc.pattern, tc.path, got)
		}
	}
}

func TestNaturalCompare(t *testing.T) {
	got := []string{"a10", "a2", "B", "a1"}
	slices.SortFunc(got, NaturalCompare)
	if want := []string{"B", "a1", "a2", "a10"}; !slices.Equal(got, want) {
		t.Errorf("got %q", got)
	}
}

func TestSorted(t *testing.T) {
	keys := []string{"z", "b", "a", "c"}
	kept := Rule{Order: []string{"c"}}.Sorted(keys)
	if want := []string{"c", "z", "b", "a"}; !slices.Equal(kept, want) {
		t.Errorf("kept: got %q", kept)
	}
	sorted := Rule{Order: []string{"c"}, SortRest: true}.Sorted(keys)
	if want := []string{"c", "a", "b", "z"}; !slices.Equal(sorted, want) {
		t.Errorf("sorted: got %q", sorted)
	}
}
