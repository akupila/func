package source

import (
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestDiskCache(t *testing.T) {
	cache := &DiskCache{
		Dir: tempdir(t),
	}

	f := cache.Get("nonexisting")
	if f != nil {
		t.Errorf("Expected nil, got %v", f)
	}

	f, err := cache.Create("file.test")
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("hello")
	if _, err := io.Copy(f, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	got, err := ioutil.ReadAll(cache.Get("file.test"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("Read data does not match\nGot\n%s\nWant\n%s", hex.Dump(got), hex.Dump(data))
	}
}

func tempdir(t *testing.T) string {
	t.Helper()
	dir, err := ioutil.TempDir("", "func-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}
