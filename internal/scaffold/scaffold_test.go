package scaffold

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGoMod(t *testing.T, dir, modulePath string) {
	t.Helper()
	content := "module " + modulePath + "\n\ngo 1.25.5\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustParse(t *testing.T, path string) {
	t.Helper()
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, path, nil, parser.AllErrors); err != nil {
		t.Errorf("%s: does not parse as valid Go: %v", path, err)
	}
}

func TestRun_GeneratesFilesInEmptyProject(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "example.com/myservice")

	result, err := Run(Options{RootDir: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result.Files))
	}
	for _, f := range result.Files {
		if !f.Written {
			t.Errorf("expected %s to be written on empty project", f.Path)
		}
	}

	mainPath := filepath.Join(dir, "main.go")
	appPath := filepath.Join(dir, "internal", "app", "app.go")

	mustParse(t, mainPath)
	mustParse(t, appPath)

	mainSrc, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainSrc), `"example.com/myservice/internal/app"`) {
		t.Errorf("main.go does not import the detected module path:\n%s", mainSrc)
	}

	appSrc, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, notWant := range []string{"PostgresKernel", "HttpKernel", "NewRabbitKernel"} {
		if strings.Contains(string(appSrc), notWant) {
			t.Errorf("did not expect %q with no kernels specified:\n%s", notWant, appSrc)
		}
	}
}

func TestRun_SkipsExistingFilesWithoutForce(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "example.com/myservice")

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Run(Options{RootDir: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	var mainResult, appResult FileResult
	for _, f := range result.Files {
		switch f.Path {
		case "main.go":
			mainResult = f
		case filepath.Join("internal", "app", "app.go"):
			appResult = f
		}
	}

	if mainResult.Written {
		t.Error("expected existing main.go to be skipped")
	}
	if !appResult.Written {
		t.Error("expected missing internal/app/app.go to be written")
	}

	src, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(src) != "package main\n" {
		t.Error("existing main.go content was overwritten without --force")
	}
}

func TestRun_ForceOverwritesExistingFiles(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "example.com/myservice")

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Run(Options{RootDir: dir, Force: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, f := range result.Files {
		if !f.Written {
			t.Errorf("expected %s to be (re)written with Force", f.Path)
		}
	}
}

func TestRun_KernelSubset(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "example.com/myservice")

	if _, err := Run(Options{RootDir: dir, Kernels: []string{"http"}}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	appSrc, err := os.ReadFile(filepath.Join(dir, "internal", "app", "app.go"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(appSrc), "HttpKernel") {
		t.Error("expected HttpKernel to be present")
	}
	for _, notWant := range []string{"PostgresKernel", "RabbitKernel"} {
		if strings.Contains(string(appSrc), notWant) {
			t.Errorf("did not expect %q with kernels=[http]:\n%s", notWant, appSrc)
		}
	}

	mustParse(t, filepath.Join(dir, "internal", "app", "app.go"))
}

func TestRun_RejectsUnknownKernel(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "example.com/myservice")

	if _, err := Run(Options{RootDir: dir, Kernels: []string{"bogus"}}); err == nil {
		t.Error("expected an error for an unknown kernel")
	}
}

func TestRun_RequiresGoMod(t *testing.T) {
	dir := t.TempDir()

	if _, err := Run(Options{RootDir: dir}); err == nil {
		t.Error("expected an error when go.mod is missing")
	}
}
