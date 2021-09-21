package tuiutil

import (
	"strings"
)

// Indent a string with the given prefix at the start of either the first, or all lines.
//
//  input     - The input string to indent.
//  prefix    - The prefix to add.
//  prefixAll - If true, prefix all lines with the given prefix.
//
// Example usage:
//
//  indented := wordwrap.Indent("Hello\nWorld", "-", true)
func Indent(input string, prefix string, prefixAll bool) string {
	lines := strings.Split(input, "\n")
	prefixLen := len(prefix)
	result := make([]string, len(lines))

	for i, line := range lines {
		if prefixAll || i == 0 {
			result[i] = prefix + line
		} else {
			result[i] = strings.Repeat(" ", prefixLen) + line
		}
	}

	return strings.Join(result, "\n")
}
