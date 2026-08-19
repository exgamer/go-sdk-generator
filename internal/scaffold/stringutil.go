package scaffold

import (
	"regexp"
	"strings"
)

var snakeBoundaryRe = regexp.MustCompile("([a-z0-9])([A-Z])")

// toSnakeCase converts a Go identifier (e.g. "CategoryID") to a snake_case
// SQL column name (e.g. "category_id").
func toSnakeCase(s string) string {
	return strings.ToLower(snakeBoundaryRe.ReplaceAllString(s, "${1}_${2}"))
}

// pluralize is a small English pluralizer, good enough for module names
// (city, product, tag, ...) used as default table names.
func pluralize(s string) string {
	if s == "" {
		return s
	}

	switch {
	case strings.HasSuffix(s, "y") && len(s) > 1 && !isVowel(s[len(s)-2]):
		return s[:len(s)-1] + "ies"
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "z"),
		strings.HasSuffix(s, "ch"), strings.HasSuffix(s, "sh"):
		return s + "es"
	default:
		return s + "s"
	}
}

func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return true
	default:
		return false
	}
}
