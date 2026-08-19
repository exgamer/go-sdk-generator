// Package gomod reads the module path declared in a project's go.mod.
package gomod

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ModulePath returns the module path declared by the "module" directive
// in <rootDir>/go.mod.
func ModulePath(rootDir string) (string, error) {
	path := filepath.Join(rootDir, "go.mod")

	f, err := os.Open(path)

	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}

	if err = scanner.Err(); err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	return "", fmt.Errorf("%s: no module directive found", path)
}

// RegisterDependencies runs `go get` for each module path and then
// `go mod tidy` in rootDir, so the generated code's imports are reflected in
// go.mod/go.sum right away. Output is streamed to the current process so the
// caller sees download/auth errors against the private module proxy.
func RegisterDependencies(rootDir string, modules []string) error {
	if len(modules) == 0 {
		return nil
	}

	if err := runGo(rootDir, append([]string{"get"}, modules...)...); err != nil {
		return fmt.Errorf("go get: %w", err)
	}

	if err := runGo(rootDir, "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	return nil
}

func runGo(dir string, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
