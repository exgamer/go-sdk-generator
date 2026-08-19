package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// SetAppName rewrites the @title/@description swagger annotations in an
// existing main.go to reflect the given app name, leaving the rest of the
// file (including manual edits) untouched.
func SetAppName(rootDir, name string) error {
	mainPath := filepath.Join(rootDir, "main.go")

	src, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("read %s (run `codegen init` first): %w", mainPath, err)
	}

	src, ok := replaceAnnotationLine(src, "@title", name+" (API)")
	if !ok {
		return fmt.Errorf("%s: @title swagger annotation not found", mainPath)
	}

	src, ok = replaceAnnotationLine(src, "@description", name)
	if !ok {
		return fmt.Errorf("%s: @description swagger annotation not found", mainPath)
	}

	return os.WriteFile(mainPath, src, 0o644)
}

// replaceAnnotationLine rewrites the first "// <tag>  <value>" line, keeping
// its original indentation/spacing, and reports whether it found one.
func replaceAnnotationLine(src []byte, tag, value string) ([]byte, bool) {
	re := regexp.MustCompile(`(?m)^(//[ \t]+` + regexp.QuoteMeta(tag) + `[ \t]+).*$`)

	loc := re.FindSubmatchIndex(src)
	if loc == nil {
		return src, false
	}

	var out bytes.Buffer
	out.Write(src[:loc[0]])
	out.Write(src[loc[2]:loc[3]])
	out.WriteString(value)
	out.Write(src[loc[1]:])

	return out.Bytes(), true
}
