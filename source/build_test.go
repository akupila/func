package source

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"
)

func TestParseBuildScript(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    BuildScript
		wantErr bool
	}{
		{
			name:  "SingleLine",
			input: "go build .",
			want: BuildScript{
				"go build .",
			},
		},
		{
			name: "MultiLine",
			input: `
                echo $FOO

                go build .
            `,
			want: BuildScript{
				"echo $FOO",
				"go build .",
			},
		},
		{
			name: "Comment",
			input: `
                # build
                go build .
            `,
			want: BuildScript{
				"go build .",
			},
		},
		{
			name:    "NonexistingBinary",
			input:   "thisfiledoesnotexist --xx --yy",
			wantErr: true,
		},
		{
			name:  "Empty",
			input: "",
			want:  nil,
		},
	}

	for _, tc := range tests {
		got, err := ParseBuildScript(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("Error = %v, want err = %t", err, tc.wantErr)
		}
		if diff := cmp.Diff(got, tc.want); diff != "" {
			t.Errorf("Diff (-got +want)\n%s", diff)
		}
	}
}

func TestBuildStep_Exec(t *testing.T) {
	tests := []struct {
		name    string
		step    BuildStep
		env     map[string]string
		stdout  string
		stderr  string
		wantErr bool
	}{
		{
			name: "Exec",
			step: "true",
		},
		{
			name:   "Stdout",
			step:   "echo hello",
			stdout: "hello\n",
		},
		{
			name:   "Stderr",
			step:   ">&2 echo err",
			stderr: "err\n",
		},
		{
			name:   "Multiple",
			step:   "echo foo && echo bar",
			stdout: "foo\nbar\n",
		},
		{
			name:   "Env",
			env:    map[string]string{"FOO": "bar"},
			step:   "echo $FOO",
			stdout: "bar\n",
		},
		{
			name:    "NonZeroExit",
			step:    "exit 123",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			buildContext := &BuildContext{
				Dir:    tempDir(t),
				Env:    tc.env,
				Stdout: &stdout,
				Stderr: &stderr,
			}

			err := tc.step.Exec(context.Background(), buildContext)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Error = %v, want err = %t", err, tc.wantErr)
			}

			if diff := cmp.Diff(stdout.String(), tc.stdout); diff != "" {
				t.Errorf("Stdout (-got +want)\n%s", diff)
			}
			if diff := cmp.Diff(stderr.String(), tc.stderr); diff != "" {
				t.Errorf("Stderr (-got +want)\n%s", diff)
			}
		})
	}
}

func TestBuildStep_Exec_nonexisting(t *testing.T) {
	step := BuildStep("nonexistingbinary --foo --bar")

	buildContext := &BuildContext{}

	err := step.Exec(context.Background(), buildContext)
	if err == nil {
		t.Fatal("Error is nil")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Errorf("Got %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() == 0 {
		t.Errorf("Got exit code 0, want non-zero")
	}
}

func TestBuildScript_Exec(t *testing.T) {
	tests := []struct {
		name    string
		script  BuildScript
		env     map[string]string
		output  string
		wantErr bool
	}{
		{
			name:   "Empty",
			script: nil,
		},
		{
			name: "One",
			script: BuildScript{
				"echo hello",
			},
			output: `
[1/1] echo hello
hello
`,
		},
		{
			name: "Multiple",
			script: BuildScript{
				"echo foo > testfile",
				"ls",
				"cat testfile",
			},
			output: `
[1/3] echo foo > testfile
[2/3] ls
testfile
[3/3] cat testfile
foo
`,
		},
		{
			name: "Env",
			script: BuildScript{
				"echo $FOO",
			},
			env: map[string]string{
				"FOO": "bar",
			},
			output: `
[1/1] echo $FOO
bar
`,
		},
		{
			name: "Error",
			script: BuildScript{
				"false",
			},
			output: `
[1/1] false
`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer

			buildContext := &BuildContext{
				Dir:    tempDir(t),
				Env:    tc.env,
				Stdout: &out,
				Stderr: &out,
			}

			err := tc.script.Exec(context.Background(), buildContext)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Error = %v, want err = %t", err, tc.wantErr)
			}

			wantOut := strings.TrimLeftFunc(tc.output, unicode.IsSpace)
			if out.String() != wantOut {
				t.Errorf("Combined output does not match\nGot\n%s\nWant\n%s", out.String(), wantOut)
			}
		})
	}
}

// tempDir creates a temporary directory to be used in tests. The directory and
// all its contents are deleted after the test has completed.
func tempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "func-test")
	if err != nil {
		t.Helper()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}
