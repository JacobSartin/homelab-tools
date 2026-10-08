package format

import (
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// entry is one pair of a block mapping as a range of whole lines: the comment
// lines directly above the key, the key, and everything indented under it.
type entry struct {
	key        string
	start, end int // inclusive line indexes
}

// mapping is a block mapping laid out as line ranges. The lines between two
// entries (blank lines and detached comments) are its gaps; they stay in
// place when entries move.
type mapping struct {
	entries []entry
	indent  int
	// prefix is the text before the first key on its line, such as "- " for a
	// mapping that is a sequence item.
	prefix string
}

var itemPrefix = regexp.MustCompile(`^[ -]*$`)

func indentOf(line string) int {
	if strings.TrimSpace(line) == "" {
		return -1
	}
	return len(line) - len(strings.TrimLeft(line, " "))
}

func isComment(line string) bool { return strings.HasPrefix(strings.TrimSpace(line), "#") }

// layout finds the line ranges of a block mapping's entries. It reports false
// for layouts it cannot rearrange line by line, which are then left alone.
func layout(lines []string, node *yaml.Node, top bool) (mapping, bool) {
	if node.Style&yaml.FlowStyle != 0 || len(node.Content) < 2 {
		return mapping{}, false
	}
	indent := node.Content[0].Column - 1
	first := lines[node.Content[0].Line-1]
	m := mapping{indent: indent, prefix: first[:min(indent, len(first))]}
	if strings.TrimSpace(m.prefix) == "" {
		m.prefix = ""
	} else if !itemPrefix.MatchString(m.prefix) {
		return mapping{}, false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		line := key.Line - 1
		if key.Kind != yaml.ScalarNode || key.Column-1 != indent ||
			(i > 0 && indentOf(lines[line]) != indent) {
			return mapping{}, false
		}
		start := line
		// The document's opening comment (such as a schema modeline) stays on
		// top, and a sequence dash keeps whatever is above it.
		if !(i == 0 && (top || m.prefix != "")) {
			for start > 0 && isComment(lines[start-1]) && indentOf(lines[start-1]) == indent {
				start--
			}
		}
		end := line
		for j := line + 1; j < len(lines); j++ {
			ind := indentOf(lines[j])
			if ind < 0 {
				continue
			}
			// A sequence may sit at its key's indentation ("key:\n- a").
			indentless := ind == indent && value.Kind == yaml.SequenceNode &&
				value.Column-1 == indent && strings.HasPrefix(lines[j][ind:], "-") &&
				!strings.HasPrefix(lines[j], "---")
			if ind <= indent && !indentless {
				break
			}
			end = j
		}
		if len(m.entries) > 0 && start <= m.entries[len(m.entries)-1].end {
			return mapping{}, false
		}
		m.entries = append(m.entries, entry{key: key.Value, start: start, end: end})
	}
	return m, true
}

func (m mapping) keys() []string {
	keys := make([]string, len(m.entries))
	for i, e := range m.entries {
		keys[i] = e.key
	}
	return keys
}

// gaps returns the lines before each entry after the first.
func (m mapping) gaps(lines []string) [][]string {
	gaps := make([][]string, len(m.entries))
	for i := 1; i < len(m.entries); i++ {
		gaps[i] = slices.Clone(lines[m.entries[i-1].end+1 : m.entries[i].start])
	}
	return gaps
}

// render returns the mapping's lines with entries in the given key order and
// the given gaps before each position.
func (m mapping) render(lines []string, order []string, gaps [][]string) []string {
	var out []string
	for i, key := range order {
		e := m.entries[slices.IndexFunc(m.entries, func(e entry) bool { return e.key == key })]
		block := slices.Clone(lines[e.start : e.end+1])
		if m.prefix != "" && e.start == m.entries[0].start {
			block[0] = strings.Repeat(" ", m.indent) + block[0][m.indent:]
		}
		out = append(append(out, gaps[i]...), block...)
	}
	if m.prefix != "" {
		out[0] = m.prefix + out[0][m.indent:]
	}
	return out
}

func (m mapping) replace(lines []string, with []string) []string {
	first, last := m.entries[0].start, m.entries[len(m.entries)-1].end
	return slices.Concat(lines[:first], with, lines[last+1:])
}
