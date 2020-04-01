package source

import (
	"testing"
)

func TestCode_Checksum(t *testing.T) {
	dir := tempdir(t)
	writeTxtar(t, dir, `
-- go.mod --
module github.com/test/test
-- main.go --
package main
func main() {}
`)

	var (
		code1 = &Code{
			Files: &FileList{Root: dir, Files: []string{"go.mod", "main.go"}},
		}
		code2 = &Code{
			Files: &FileList{Root: dir, Files: []string{"main.go"}},
		}
		code3 = &Code{
			Files: &FileList{Root: dir, Files: []string{"go.mod", "main.go"}},
			Build: BuildScript{
				"go build .",
			},
		}
		code4 = &Code{
			Files: &FileList{Root: dir, Files: []string{"go.mod", "main.go"}},
			Build: BuildScript{
				"go build -ldflags \"-w -s\" .",
			},
		}
	)

	tests := []struct {
		name      string
		a, b      *Code
		wantEqual bool
	}{
		{name: "Identical", a: code1, b: code1, wantEqual: true},
		{name: "DiffFiles", a: code1, b: code2, wantEqual: false},
		{name: "WithBuild", a: code1, b: code3, wantEqual: false},
		{name: "DiffBuild", a: code3, b: code4, wantEqual: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sum1, err := tc.a.Checksum()
			if err != nil {
				t.Fatal(err)
			}
			sum2, err := tc.b.Checksum()
			if err != nil {
				t.Fatal(err)
			}
			equal := sum1 == sum2
			if equal != tc.wantEqual {
				t.Errorf("Equal = %t, want equal = %t", equal, tc.wantEqual)
			}
		})
	}
}
