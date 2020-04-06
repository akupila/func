package cli

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPrefixLineWriter(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		input  string
		want   string
	}{
		{
			name:   "NoInput",
			prefix: "",
			input:  "",
			want:   "",
		},
		{
			name:   "NoPrefix",
			prefix: "",
			input:  "foo",
			want:   "foo",
		},
		{
			name:   "Prefix",
			prefix: ">",
			input:  "foo",
			want:   ">foo",
		},
		{
			name:   "NoTrailingNewline",
			prefix: ">",
			input:  "foo\nbar",
			want:   ">foo\n>bar",
		},
		{
			name:   "TrailingNewline",
			prefix: ">",
			input:  "foo\nbar\n",
			want:   ">foo\n>bar\n",
		},
		{
			name:   "EmptyLines",
			prefix: ">",
			input:  "foo\n\nbar\n\nbaz\n\n",
			want:   ">foo\n>\n>bar\n>\n>baz\n>\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			pw := &PrefixWriter{Output: &buf, Prefix: []byte(tc.prefix)}
			n, err := pw.Write([]byte(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			if n != len(tc.input) {
				t.Fatalf("Write returned %d, want %d", n, len(tc.input))
			}
			if diff := cmp.Diff(buf.String(), tc.want); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}
