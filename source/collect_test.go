package source

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExcludeFile(t *testing.T) {
	filter := ExcludeFile("/foo/bar/baz.txt")
	pass := map[string]bool{
		"/foo/bar/baz.txt": false,
		".git/HEAD":        true,
		"/foo/bar":         true,
		"baz.txt":          true,
	}
	for input, want := range pass {
		t.Run(input, func(t *testing.T) {
			got := filter(input)
			if got != want {
				t.Errorf("Filter %q; got = %t, want = %t", input, got, want)
			}
		})
	}
}

func TestExcludeHidden(t *testing.T) {
	filter := ExcludeHidden()
	pass := map[string]bool{
		"foo.txt":    true,
		".git":       false,
		"src/.build": false,
	}
	for input, want := range pass {
		t.Run(input, func(t *testing.T) {
			got := filter(input)
			if got != want {
				t.Errorf("Filter %q; got = %t, want = %t", input, got, want)
			}
		})
	}
}

func TestCollect(t *testing.T) {
	tests := []struct {
		name      string
		files     string
		filters   []Filter
		wantFiles []string
		wantErr   bool
	}{
		{
			name:      "Empty",
			files:     "",
			wantFiles: nil,
		},
		{
			name: "Files",
			files: `
-- foo.txt --
Foo
-- bar.txt --
Bar
`,
			wantFiles: []string{"bar.txt", "foo.txt"},
		},
		{
			name: "Filter",
			files: `
-- foo.txt --
Foo
-- bar.txt --
Bar
-- baz/qux.txt --
Qux
`,
			filters: []Filter{
				ExcludeFile("foo.txt"),
				ExcludeFile("baz/qux.txt"),
			},
			wantFiles: []string{"bar.txt"},
		},
		{
			name: "HiddenDir",
			files: `
-- .git/HEAD --
ref
-- source/.build/cache --
xyz
-- source/main.go --
package main
`,
			filters: []Filter{
				ExcludeHidden(),
			},
			wantFiles: []string{
				"source/main.go",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := tempdir(t)
			writeTxtar(t, dir, tc.files)
			got, err := Collect(dir, tc.filters...)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Error = %v, want err = %t", err, tc.wantErr)
			}
			want := &FileList{Root: dir, Files: tc.wantFiles}
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}
