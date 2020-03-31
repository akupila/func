package resource

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
)

func TestList_ByName(t *testing.T) {
	foo := &Resource{Name: "Foo"}
	bar := &Resource{Name: "Bar"}

	tests := []struct {
		name   string
		list   List
		lookup string
		want   *Resource
	}{
		{
			name:   "Match",
			list:   List{foo, bar},
			lookup: "Foo",
			want:   foo,
		},
		{
			name:   "NoMatch",
			list:   List{foo, bar},
			lookup: "bar",
			want:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.list.ByName(tc.lookup)
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

func TestList_OfType(t *testing.T) {
	foo := &Resource{Name: "Foo", Type: "lambda"}
	bar := &Resource{Name: "Bar", Type: "lambda"}
	baz := &Resource{Name: "Bar", Type: "role"}

	list := List{foo, bar, baz}

	got := list.OfType("lambda")
	want := List{foo, bar}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

func TestList_WithSource(t *testing.T) {
	foo := &Resource{Name: "Foo"}
	bar := &Resource{Name: "Bar", SourceCode: &SourceCode{}}
	baz := &Resource{Name: "Bar", SourceCode: &SourceCode{}}

	list := List{foo, bar, baz}

	got := list.WithSource()
	want := List{bar, baz}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

func TestResource_SourceFiles(t *testing.T) {
	dir, done := writeTestFiles(t, map[string][]byte{
		"main.go":    []byte("package main"),
		"config.hcl": []byte("resource \"test\" {}"), // config file ignored
		".git/HEAD":  []byte("ref: refs/heads/xxx"),  // . dir ignored
	})
	defer done()

	def := hcl.Range{
		Filename: filepath.Join(dir, "config.hcl"),
	}

	tests := []struct {
		name      string
		files     map[string][]byte
		resource  *Resource
		wantFiles []string
	}{
		{
			name: "NoSource",
			resource: &Resource{
				Definition: def,
				SourceCode: nil,
			},
			wantFiles: nil,
		},
		{
			name: "NoSource",
			resource: &Resource{
				Definition: def,
				SourceCode: &SourceCode{
					Dir: dir,
				},
			},
			wantFiles: []string{"main.go"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			list, err := tc.resource.SourceFiles()
			if err != nil {
				t.Fatal(err)
			}

			got := list.Files
			if diff := cmp.Diff(got, tc.wantFiles); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

func writeTestFiles(t *testing.T, files map[string][]byte) (string, func()) {
	t.Helper()
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range files {
		filename := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.Fatal(err)
		}
		if err := ioutil.WriteFile(filename, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir, func() {
		_ = os.RemoveAll(dir)
	}
}
