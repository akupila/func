package source

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"testing"
)

func TestFileList_Write(t *testing.T) {
	files := `
-- foo.txt --
Foo
-- bar/baz.txt --
Qux
`
	dir := tempdir(t)
	writeTxtar(t, dir, files)
	l := &FileList{
		Root:  dir,
		Files: []string{"foo.txt", "bar/baz.txt"},
	}
	var buf bytes.Buffer
	if err := l.Write(&buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := "Qux\nFoo\n"
	if got != want {
		t.Errorf("Got %q, want %q", got, want)
	}
}

func TestFileList_Zip(t *testing.T) {
	files := `
-- foo.txt --
Foo
-- bar/bar.txt --
Bar
`
	dir := tempdir(t)
	writeTxtar(t, dir, files)
	l := &FileList{
		Root:  dir,
		Files: []string{"foo.txt", "bar/bar.txt"},
	}

	var buf bytes.Buffer
	if err := l.Zip(&buf); err != nil {
		t.Fatal(err)
	}

	want := map[string][]byte{
		"foo.txt":     []byte("Foo\n"),
		"bar/bar.txt": []byte("Bar\n"),
	}

	zf, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(zf.File) != len(want) {
		t.Errorf("Number of files does not match; got %d, want %d", len(zf.File), len(want))
	}
	for _, f := range zf.File {
		wantData, ok := want[f.Name]
		if !ok {
			t.Errorf("Unexpected file %q", f.Name)
			continue
		}
		b, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		gotData, err := ioutil.ReadAll(b)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(gotData, wantData) {
			t.Errorf("Data does not match for %s\nGot\n%s\nWant\n%s", f.Name, hex.Dump(gotData), hex.Dump(wantData))
		}
	}
}
