package scaffold

import (
	"fmt"
	"regexp"
	"strings"
)

var domainFieldTypes = map[string]bool{
	"string":  true,
	"bool":    true,
	"int":     true,
	"int64":   true,
	"uint":    true,
	"uint64":  true,
	"float32": true,
	"float64": true,
}

var integerFieldTypes = map[string]bool{
	"int":    true,
	"int64":  true,
	"uint":   true,
	"uint64": true,
}

var fieldNameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)

// Field is one user-defined entity field (the always-present ID/Status
// fields are added automatically and must not be listed).
type Field struct {
	Name string
	Type string
}

// ParseFields parses a "Name:type,Name2:type2" flag value into Fields,
// capitalizing field names and validating types.
func ParseFields(raw string) ([]Field, error) {
	var fields []Field

	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		nameType := strings.SplitN(part, ":", 2)
		if len(nameType) != 2 {
			return nil, fmt.Errorf("invalid field %q, expected Name:type", part)
		}

		name := strings.TrimSpace(nameType[0])
		typ := strings.TrimSpace(nameType[1])

		if !fieldNameRe.MatchString(name) {
			return nil, fmt.Errorf("invalid field name %q: must be a Go identifier (letters/digits, starting with a letter)", name)
		}
		name = strings.ToUpper(name[:1]) + name[1:]

		if name == "ID" || name == "Status" {
			return nil, fmt.Errorf("field %q is added automatically, do not specify it in --fields", name)
		}

		if !domainFieldTypes[typ] {
			return nil, fmt.Errorf("unsupported type %q for field %q (allowed: string, bool, int, int64, uint, uint64, float32, float64)", typ, name)
		}

		fields = append(fields, Field{Name: name, Type: typ})
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields given")
	}

	return fields, nil
}

// searchableFields returns the subset of fields exposed on the Search DTO:
// strings (ILIKE filters) and *ID-suffixed integer fields (exact-match
// filters) — mirrors the filtering pattern used by the postgres repository
// in go-sdk-rest-template.
func searchableFields(fields []Field) []Field {
	var out []Field
	for _, f := range fields {
		if f.Type == "string" {
			out = append(out, f)
			continue
		}
		if strings.HasSuffix(f.Name, "ID") && integerFieldTypes[f.Type] {
			out = append(out, f)
		}
	}
	return out
}
