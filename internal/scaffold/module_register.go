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
	"strings"

	"github.com/exgamer/go-sdk-generator/internal/gomod"
)

// RegisterModule adds &{alias}.Module{} to internal/app/app.go's
// RegisterAndInitModules(...) call, plus the matching import, the same way
// AddKernel patches RegisterAndInitKernels(...) — leaves the rest of the
// file (including manual edits) untouched. Reports added=false without
// error if the module is already registered.
func RegisterModule(rootDir, domain, module string) (added bool, err error) {
	appPath := filepath.Join(rootDir, "internal", "app", "app.go")

	src, err := os.ReadFile(appPath)

	if err != nil {
		return false, fmt.Errorf("read %s (run `codegen init` first): %w", appPath, err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, appPath, src, parser.ParseComments)

	if err != nil {
		return false, fmt.Errorf("parse %s: %w", appPath, err)
	}

	importDecl := findImportDecl(file)

	if importDecl == nil {
		return false, fmt.Errorf("%s: no import declaration found", appPath)
	}

	call := findCallBySelector(file, "RegisterAndInitModules")

	if call == nil {
		return false, fmt.Errorf("%s: RegisterAndInitModules(...) call not found", appPath)
	}

	modulePath, err := gomod.ModulePath(rootDir)

	if err != nil {
		return false, fmt.Errorf("determine module path (run `go mod init` first): %w", err)
	}

	importPath := modulePath + "/internal/app/bootstrap/" + domain + "/" + module

	if importedPaths(importDecl)[importPath] {
		return false, nil
	}

	alias := domain + strings.ToUpper(module[:1]) + module[1:]

	importDecl.Specs = append(importDecl.Specs, &ast.ImportSpec{
		Name: ast.NewIdent(alias),
		Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(importPath)},
	})

	// Built manually (not via parser.ParseExpr) so every position is the
	// AST zero value — splicing a node with positions from a fresh,
	// separately-parsed file can otherwise confuse the printer's line-break
	// decisions for the surrounding call (observed: it would break the
	// selector itself across lines, e.g. "&catalogProduct.\n\tModule{}").
	call.Args = append(call.Args, &ast.UnaryExpr{
		Op: token.AND,
		X: &ast.CompositeLit{
			Type: &ast.SelectorExpr{
				X:   ast.NewIdent(alias),
				Sel: ast.NewIdent("Module"),
			},
		},
	})

	var buf bytes.Buffer

	if err := format.Node(&buf, fset, file); err != nil {
		return false, fmt.Errorf("format %s: %w", appPath, err)
	}

	formatted, err := format.Source(buf.Bytes())

	if err != nil {
		return false, fmt.Errorf("gofmt %s: %w (source:\n%s)", appPath, err, buf.Bytes())
	}

	if err := os.WriteFile(appPath, formatted, 0o644); err != nil {
		return false, fmt.Errorf("write %s: %w", appPath, err)
	}

	return true, nil
}
