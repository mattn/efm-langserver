package langserver

import (
	"strings"
)

func convertRowColToIndex(s string, row, col int) int {
	lines := strings.Split(s, "\n")

	if row < 0 {
		row = 0
	} else if row >= len(lines) {
		row = len(lines) - 1
	}

	if col < 0 {
		col = 0
	} else if col > len(lines[row]) {
		col = len(lines[row])
	}

	index := 0
	for i := 0; i < row; i++ {
		// Add the length of each line plus 1 for the newline character
		index += len(lines[i]) + 1
	}
	index += col

	return index
}
