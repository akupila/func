package source

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFileList(t *testing.T) {
	files := map[string][]byte{
		"foo.txt":     []byte("foo"),
		"bar/baz.txt": []byte("barbaz"),
	}
	dir, done := writeTestFiles(t, files)
	defer done()

	fl := NewFileList(dir)
	fl.Add("foo.txt")
	fl.Add("bar/baz.txt")

	if diff := cmp.Diff(fl.Root, dir); diff != "" {
		t.Errorf("Root does not match (-got +want)\n%s", diff)
	}

	wantFiles := []string{"foo.txt", "bar/baz.txt"}
	if diff := cmp.Diff(fl.Files, wantFiles); diff != "" {
		t.Errorf("Files do not match (-got +want)\n%s", diff)
	}

	var buf bytes.Buffer
	err := fl.Write(&buf)
	if err != nil {
		t.Fatalf("Compute checksum: %v", err)
	}
	gotContent := buf.String()
	wantContent := "barbazfoo"
	if gotContent != wantContent {
		t.Errorf("Written contenet does not match\nGot  %q\nWant %q", gotContent, wantContent)
	}

	zip := &bytes.Buffer{}
	if err := fl.Zip(zip); err != nil {
		t.Fatalf("zip: %v", err)
	}
	checkZip(t, zip.Bytes(), files)
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

func checkZip(t *testing.T, data []byte, want map[string][]byte) {
	t.Helper()
	zf, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
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
