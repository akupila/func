package source

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"
)

func TestFileList_Copy(t *testing.T) {
	files := `
-- foo.txt --
Foo
-- bar/bar.txt --
Bar
`
	src := tempdir(t)
	writeTxtar(t, src, files)
	before := FileList{
		Root:  src,
		Files: []string{"foo.txt", "bar/bar.txt"},
	}

	dst := tempdir(t)
	err := before.Copy(dst)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := exec.LookPath("diff"); err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	out, err := exec.Command("diff", "-q", src, dst).CombinedOutput()
	if err != nil {
		msg := string(out)
		msg = strings.ReplaceAll(msg, src, "<src>")
		msg = strings.ReplaceAll(msg, dst, "<dst>")
		t.Fatalf("Diff: %v\n%s", err, msg)
	}
}

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
