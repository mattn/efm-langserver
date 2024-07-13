package langserver_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/mattn/efm-langserver/langserver"
)

func TestComputeEdits(t *testing.T) {
	cases := []struct {
		name          string
		before, after string
		want          []langserver.TextEdit
	}{
		{
			name:   "Empty",
			before: "", after: "", want: []langserver.TextEdit{},
		},
		{
			name:   "Empty to one line",
			before: "", after: "foo", want: []langserver.TextEdit{
				{NewText: "foo"},
			},
		},
		{
			name:   "One line to another",
			before: "foo", after: "bar", want: []langserver.TextEdit{
				{Range: langserver.Range{End: langserver.Position{Line: 1}}},
				{Range: langserver.Range{Start: langserver.Position{Line: 1}, End: langserver.Position{Line: 1}}, NewText: "bar"},
			},
		},
		{
			name:   "Replace multi lines (issue 281)",
			before: "foo\nbar\nbaz\n", after: "one\ntwo\nthree\n",
			want: []langserver.TextEdit{
				{Range: langserver.Range{End: langserver.Position{Line: 3}}},
				{
					Range:   langserver.Range{Start: langserver.Position{Line: 3}, End: langserver.Position{Line: 3}},
					NewText: "three\n",
				},
				{
					Range:   langserver.Range{Start: langserver.Position{Line: 3}, End: langserver.Position{Line: 3}},
					NewText: "two\n",
				},
				{
					Range:   langserver.Range{Start: langserver.Position{Line: 3}, End: langserver.Position{Line: 3}},
					NewText: "one\n",
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			edits := langserver.ComputeEdits("", c.before, c.after)
			if diff := cmp.Diff(c.want, edits); diff != "" {
				t.Errorf("unexpected edits (-want +got):\n%s", diff)
			}
		})
	}
}
