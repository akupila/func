package source

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/txtar"
)

func tempdir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Helper()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func writeTxtar(t *testing.T, dir, input string) {
	archive := txtar.Parse([]byte(input))
	for _, f := range archive.Files {
		filename := filepath.Join(dir, f.Name)
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.Fatal(err)
		}
		if err := ioutil.WriteFile(filename, f.Data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
