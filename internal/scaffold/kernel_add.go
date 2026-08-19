package scaffold

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
)

// KernelResult reports whether a kernel was newly registered.
type KernelResult struct {
	Kernel string
	Added  bool // false when the kernel was already registered
}

// AddKernel registers the given kernels in an existing internal/app/app.go:
// it adds the missing import and the matching RegisterAndInitKernels
// argument, leaving everything else in the file (including manual edits)
// untouched. Kernels already present are reported with Added=false and
// skipped.
func AddKernel(rootDir string, kernels []string) ([]KernelResult, error) {
	if err := validateKernels(kernels); err != nil {
		return nil, err
	}

	appPath := filepath.Join(rootDir, "internal", "app", "app.go")

	src, err := os.ReadFile(appPath)

	if err != nil {
		return nil, fmt.Errorf("read %s (run `codegen init` first): %w", appPath, err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, appPath, src, parser.ParseComments)

	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", appPath, err)
	}

	importDecl := findImportDecl(file)

	if importDecl == nil {
		return nil, fmt.Errorf("%s: no import declaration found", appPath)
	}

	call := findRegisterCall(file)

	if call == nil {
		return nil, fmt.Errorf("%s: RegisterAndInitKernels(...) call not found", appPath)
	}

	existing := importedPaths(importDecl)

	results := make([]KernelResult, 0, len(kernels))
	changed := false

	for _, k := range kernels {
		spec := kernelSpecs[k]

		if existing[spec.importPath()] {
			results = append(results, KernelResult{Kernel: k, Added: false})

			continue
		}

		importDecl.Specs = append(importDecl.Specs, &ast.ImportSpec{
			Name: ast.NewIdent(spec.alias),
			Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(spec.importPath())},
		})

		argExpr, err := parser.ParseExpr(spec.argSrc)

		if err != nil {
			return nil, fmt.Errorf("internal error: parse kernel arg for %q: %w", k, err)
		}

		call.Args = append(call.Args, argExpr)

		existing[spec.importPath()] = true
		results = append(results, KernelResult{Kernel: k, Added: true})
		changed = true
	}

	if !changed {
		return results, nil
	}

	var buf bytes.Buffer

	if err := format.Node(&buf, fset, file); err != nil {
		return nil, fmt.Errorf("format %s: %w", appPath, err)
	}

	formatted, err := format.Source(buf.Bytes())

	if err != nil {
		return nil, fmt.Errorf("gofmt %s: %w (source:\n%s)", appPath, err, buf.Bytes())
	}

	if err := os.WriteFile(appPath, formatted, 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", appPath, err)
	}

	return results, nil
}

func findImportDecl(file *ast.File) *ast.GenDecl {
	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			return gd
		}
	}

	return nil
}

func findRegisterCall(file *ast.File) *ast.CallExpr {
	return findCallBySelector(file, "RegisterAndInitKernels")
}

// findCallBySelector finds the first call expression anywhere in file whose
// selector (method/function name) matches selName, e.g. a call shaped like
// `x.RegisterAndInitKernels(...)` or `x.RegisterAndInitModules(...)`.
func findCallBySelector(file *ast.File, selName string) *ast.CallExpr {
	var call *ast.CallExpr

	ast.Inspect(file, func(n ast.Node) bool {
		if call != nil {
			return false
		}

		ce, ok := n.(*ast.CallExpr)

		if !ok {
			return true
		}

		sel, ok := ce.Fun.(*ast.SelectorExpr)

		if !ok || sel.Sel.Name != selName {
			return true
		}

		call = ce

		return false
	})

	return call
}

func importedPaths(decl *ast.GenDecl) map[string]bool {
	paths := make(map[string]bool, len(decl.Specs))

	for _, spec := range decl.Specs {
		imp, ok := spec.(*ast.ImportSpec)

		if !ok {
			continue
		}

		if path, err := strconv.Unquote(imp.Path.Value); err == nil {
			paths[path] = true
		}
	}

	return paths
}
