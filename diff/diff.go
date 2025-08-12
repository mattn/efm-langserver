// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.package langserver
// Source:
// https://github.com/golang/tools/blob/78b158585360beccadc3faac6e35759f491831f3/internal/lsp/diff/myers/diff.go

package diff

import (
	"strings"

	"github.com/konradmalik/efm-langserver/types"
)

// opKind is used to denote the type of operation a line represents.
type opKind int

const (
	// delete is the operation kind for a line that is present in the input
	// but not in the output.
	delete opKind = iota
	// insert is the operation kind for a line that is new in the output.
	insert
)

// Sources:
// https://blog.jcoglan.com/2017/02/17/the-myers-diff-algorithm-part-3/
// https://www.codeproject.com/Articles/42279/%2FArticles%2F42279%2FInvestigating-Myers-diff-algorithm-Part-1-of-2

// ComputeEdits computes diff edits from 2 string inputs
func ComputeEdits(_ types.DocumentURI, before, after string) []types.TextEdit {
	ops := operations(splitLines(before), splitLines(after))
	edits := make([]types.TextEdit, 0, len(ops))
	for _, op := range ops {
		switch op.Kind {
		case delete:
			// Delete: unformatted[i1:i2] is deleted.
			edits = append(edits, types.TextEdit{Range: types.Range{
				Start: types.Position{Line: op.I1, Character: 0},
				End:   types.Position{Line: op.I2, Character: 0},
			}})
		case insert:
			// Insert: formatted[j1:j2] is inserted at unformatted[i1:i1].
			if content := strings.Join(op.Content, ""); content != "" {
				edits = append(edits, types.TextEdit{
					Range: types.Range{
						Start: types.Position{Line: op.I1, Character: 0},
						End:   types.Position{Line: op.I2, Character: 0},
					},
					NewText: content,
				})
			}
		}
	}
	return edits
}

type operation struct {
	Kind    opKind
	Content []string // content from b
	I1, I2  int      // indices of the line in a
	J1      int      // indices of the line in b, J2 implied by len(Content)
}

// operations returns the list of operations to convert a into b, consolidating
// operations for multiple lines and not including equal lines.
func operations(a, b []string) []*operation {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}

	trace, offset := shortestEditSequence(a, b)
	snakes := backtrack(trace, len(a), len(b), offset)

	M, N := len(a), len(b)

	var i int
	solution := make([]*operation, len(a)+len(b))

	add := func(op *operation, i2, j2 int) {
		if op == nil {
			return
		}
		op.I2 = i2
		if op.Kind == insert {
			op.Content = b[op.J1:j2]
		}
		solution[i] = op
		i++
	}
	x, y := 0, 0
	for _, snake := range snakes {
		if len(snake) < 2 {
			continue
		}
		var op *operation
		// delete (horizontal)
		for snake[0]-snake[1] > x-y {
			if op == nil {
				op = &operation{
					Kind: delete,
					I1:   x,
					J1:   y,
				}
			}
			x++
			if x == M {
				break
			}
		}
		add(op, x, y)
		op = nil
		// insert (vertical)
		for snake[0]-snake[1] < x-y {
			if op == nil {
				op = &operation{
					Kind: insert,
					I1:   x,
					J1:   y,
				}
			}
			y++
		}
		add(op, x, y)
		op = nil
		// equal (diagonal)
		for x < snake[0] {
			x++
			y++
		}
		if x >= M && y >= N {
			break
		}
	}
	return solution[:i]
}

// backtrack uses the trace for the edit sequence computation and returns the
// "snakes" that make up the solution. A "snake" is a single deletion or
// insertion followed by zero or diagonals.
func backtrack(trace [][]int, x, y, offset int) [][]int {
	snakes := make([][]int, len(trace))
	d := len(trace) - 1
	for ; x > 0 && y > 0 && d > 0; d-- {
		V := trace[d]
		if len(V) == 0 {
			continue
		}
		snakes[d] = []int{x, y}

		k := x - y

		var kPrev int
		if k == -d || (k != d && V[k-1+offset] < V[k+1+offset]) {
			kPrev = k + 1
		} else {
			kPrev = k - 1
		}

		x = V[kPrev+offset]
		y = x - kPrev
	}
	if x < 0 || y < 0 {
		return snakes
	}
	snakes[d] = []int{x, y}
	return snakes
}

// shortestEditSequence returns the shortest edit sequence that converts a into b.
func shortestEditSequence(a, b []string) ([][]int, int) {
	M, N := len(a), len(b)
	V := make([]int, 2*(N+M)+1)
	offset := N + M
	trace := make([][]int, N+M+1)

	// Iterate through the maximum possible length of the SES (N+M).
	for d := 0; d <= N+M; d++ {
		copyV := make([]int, len(V))
		// k lines are represented by the equation y = x - k. We move in
		// increments of 2 because end points for even d are on even k lines.
		for k := -d; k <= d; k += 2 {
			// At each point, we either go down or to the right. We go down if
			// k == -d, and we go to the right if k == d. We also prioritize
			// the maximum x value, because we prefer deletions to insertions.
			var x int
			if k == -d || (k != d && V[k-1+offset] < V[k+1+offset]) {
				x = V[k+1+offset] // down
			} else {
				x = V[k-1+offset] + 1 // right
			}

			y := x - k

			// Diagonal moves while we have equal contents.
			for x < M && y < N && a[x] == b[y] {
				x++
				y++
			}

			V[k+offset] = x

			// Return if we've exceeded the maximum values.
			if x == M && y == N {
				// Makes sure to save the state of the array before returning.
				copy(copyV, V)
				trace[d] = copyV
				return trace, offset
			}
		}

		// Save the state of the array.
		copy(copyV, V)
		trace[d] = copyV
	}
	return nil, 0
}

func splitLines(text string) []string {
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
