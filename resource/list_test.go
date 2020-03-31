package resource

import (
	"testing"

	"github.com/func/func/source"
	"github.com/google/go-cmp/cmp"
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
	bar := &Resource{Name: "Bar", SourceCode: &source.Code{}}
	baz := &Resource{Name: "Bar", SourceCode: &source.Code{}}

	list := List{foo, bar, baz}

	got := list.WithSource()
	want := List{bar, baz}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}
