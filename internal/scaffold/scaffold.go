// Package scaffold generates the starter main.go and internal/app/app.go
// files for an empty project built on go-sdk-rest-template.
package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/exgamer/go-sdk-generator/internal/gomod"
)

//go:embed templates
var templatesFS embed.FS

// Options controls what Run generates.
type Options struct {
	// RootDir is the project root. Defaults to "." when empty.
	RootDir string
	// AppTitle is used in the main.go swagger annotation. Defaults to the
	// last path segment of the module path.
	AppTitle string
	// Kernels is the subset of {"postgres", "http", "rabbit"} to register
	// in app.go. Empty means no kernels are registered.
	Kernels []string
	// Force overwrites main.go / internal/app/app.go even if they already
	// exist. docs/docs.go is never overwritten by Force (see Run).
	Force bool
}

// FileResult describes the outcome for a single generated file.
type FileResult struct {
	Path    string
	Written bool // false when skipped because the file already existed
}

// Result is the outcome of Run.
type Result struct {
	Files []FileResult
}

// Run generates main.go, internal/app/app.go, and a docs/docs.go placeholder
// (for the swag-generated docs package main.go imports) when they are
// missing. main.go/app.go are also regenerated when opts.Force is set;
// docs/docs.go never is, so a real `swag init` output is never clobbered.
func Run(opts Options) (Result, error) {
	root := opts.RootDir

	if root == "" {
		root = "."
	}

	modulePath, err := gomod.ModulePath(root)

	if err != nil {
		return Result{}, fmt.Errorf("determine module path (run `go mod init` first): %w", err)
	}

	kernels := opts.Kernels

	if err := validateKernels(kernels); err != nil {
		return Result{}, err
	}

	appTitle := opts.AppTitle

	if appTitle == "" {
		appTitle = lastSegment(modulePath)
	}

	data := struct {
		ModulePath  string
		AppTitle    string
		HasPostgres bool
		HasHttp     bool
		HasRabbit   bool
		HasRedis    bool
	}{
		ModulePath:  modulePath,
		AppTitle:    appTitle,
		HasPostgres: contains(kernels, "postgres"),
		HasHttp:     contains(kernels, "http"),
		HasRabbit:   contains(kernels, "rabbit"),
		HasRedis:    contains(kernels, "redis"),
	}

	targets := []struct {
		relPath  string
		template string
		force    bool
	}{
		{"main.go", "templates/main.go.tmpl", opts.Force},
		{filepath.Join("internal", "app", "app.go"), "templates/app.go.tmpl", opts.Force},
		// docs/docs.go is a placeholder for `swag init` (main.go imports it
		// unconditionally). Never overwritten by --force: once `swag init`
		// has generated the real file, `codegen init --force` must not
		// clobber it back to the stub.
		{filepath.Join("docs", "docs.go"), "templates/docs.go.tmpl", false},
	}

	result := Result{}

	for _, t := range targets {
		fr, err := generateFile(root, t.relPath, t.template, data, t.force)

		if err != nil {
			return Result{}, err
		}

		result.Files = append(result.Files, fr)
	}

	return result, nil
}

// generateFile renders tmplName with data and writes it to <root>/<relPath>,
// skipping (without error) when the file already exists and force is false.
func generateFile(root, relPath, tmplName string, data any, force bool) (FileResult, error) {
	absPath := filepath.Join(root, relPath)

	if !force {
		if _, err := os.Stat(absPath); err == nil {
			return FileResult{Path: relPath, Written: false}, nil
		} else if !os.IsNotExist(err) {
			return FileResult{}, fmt.Errorf("stat %s: %w", absPath, err)
		}
	}

	src, err := render(tmplName, data)

	if err != nil {
		return FileResult{}, fmt.Errorf("render %s: %w", relPath, err)
	}

	if err = os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return FileResult{}, fmt.Errorf("create directory for %s: %w", relPath, err)
	}

	if err = os.WriteFile(absPath, src, 0o644); err != nil {
		return FileResult{}, fmt.Errorf("write %s: %w", relPath, err)
	}

	return FileResult{Path: relPath, Written: true}, nil
}

func render(name string, data any) ([]byte, error) {
	tmpl, err := template.ParseFS(templatesFS, name)

	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	formatted, err := format.Source(buf.Bytes())

	if err != nil {
		return nil, fmt.Errorf("gofmt: %w (source:\n%s)", err, buf.String())
	}

	return formatted, nil
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}

	return false
}

func lastSegment(modulePath string) string {
	parts := strings.Split(modulePath, "/")

	return parts[len(parts)-1]
}
