package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// A BuildScript is a list of build instructions.
type BuildScript []BuildStep

// ParseBuildScript parses a shell-script like string into build steps.
//
// Every line is checked to start with a valid executable on $PATH.
// See exec.LookPath() for exact behavior.
//
// Lines starting with # are skipped.
func ParseBuildScript(str string) (BuildScript, error) {
	lines := strings.Split(str, "\n")
	out := make(BuildScript, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, " ")
		for _, part := range parts {
			if strings.Contains(part, "=") {
				continue
			}
			if _, err := exec.LookPath(part); err != nil {
				return nil, fmt.Errorf("line %d: %w", i, err)
			}
			break
		}
		out = append(out, BuildStep(line))
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// BuildContext sets the context for a step in a build.
type BuildContext struct {
	Dir    string
	Env    map[string]string
	Stdout io.Writer
	Stderr io.Writer
}

// BuildStep is a step to execute in a build, typically one command.
type BuildStep string

// Exec executes a build step in a shell.
//
// Cancelling the context will terminate the running command.
func (s BuildStep) Exec(ctx context.Context, buildContext *BuildContext) (err error) {
	env := os.Environ()
	for k, v := range buildContext.Env {
		env = append(env, k+"="+v)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", string(s))
	cmd.Env = env
	cmd.Dir = buildContext.Dir
	cmd.Stdout = buildContext.Stdout
	cmd.Stderr = buildContext.Stderr

	return cmd.Run()
}

func (script BuildScript) String() string {
	if len(script) == 0 {
		return ""
	}
	out := make([]string, len(script))
	for i, s := range script {
		out[i] = string(s)
	}
	return strings.Join(out, ";")
}

// Exec executes all steps in the build script.
func (script BuildScript) Exec(ctx context.Context, buildContext *BuildContext) error {
	n := len(script)
	for i, s := range script {
		if buildContext.Stderr != nil {
			fmt.Fprintf(buildContext.Stderr, "[%d/%d] %s\n", i+1, n, s)
		}
		if err := s.Exec(ctx, buildContext); err != nil {
			return fmt.Errorf("exec step %d: %s: %w", i, s, err)
		}
	}
	return nil
}
