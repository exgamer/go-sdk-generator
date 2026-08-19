package scaffold

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var domainNameRe = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// DomainOptions controls what AddDomain generates.
type DomainOptions struct {
	// RootDir is the project root. Defaults to "." when empty.
	RootDir string
	// Domain is the domain name, e.g. "handbook".
	Domain string
	// Module is the module name, e.g. "city". Also used as the Go package
	// name and (capitalized) as the entity type name.
	Module string
	// Fields are the entity fields beyond the always-present ID/Status.
	Fields []Field
	// Methods is the subset of {paginated, getbyid, create, update, delete,
	// activate, deactivate} to generate on Repository/Service. Empty means
	// all of them (use ParseMethods("") to get the default explicitly).
	Methods []string
	// Force overwrites files that already exist.
	Force bool
}

// AddDomain generates the domain layer for a new module — entity.go,
// dto.go (Search), repository.go (Repository interface) and service.go —
// under internal/domains/{domain}/{module}/, following the
// go-sdk-rest-template convention (see the real "city" module).
func AddDomain(opts DomainOptions) (Result, error) {
	root := opts.RootDir
	if root == "" {
		root = "."
	}

	if !domainNameRe.MatchString(opts.Domain) {
		return Result{}, fmt.Errorf("invalid domain %q: lowercase letters/digits only, starting with a letter", opts.Domain)
	}
	if !domainNameRe.MatchString(opts.Module) {
		return Result{}, fmt.Errorf("invalid module %q: lowercase letters/digits only, starting with a letter", opts.Module)
	}
	if len(opts.Fields) == 0 {
		return Result{}, fmt.Errorf("no fields given")
	}
	if len(opts.Methods) == 0 {
		return Result{}, fmt.Errorf("no methods given")
	}

	data := struct {
		Module       string
		EntityName   string
		Fields       []Field
		SearchFields []Field
		methodFlags
	}{
		Module:       opts.Module,
		EntityName:   strings.ToUpper(opts.Module[:1]) + opts.Module[1:],
		Fields:       opts.Fields,
		SearchFields: searchableFields(opts.Fields),
		methodFlags:  buildMethodFlags(opts.Methods),
	}

	domainDir := filepath.Join("internal", "domains", opts.Domain, opts.Module)

	targets := []struct {
		relName string
		tmpl    string
	}{
		{"entity.go", "templates/domain/entity.go.tmpl"},
		{"dto.go", "templates/domain/dto.go.tmpl"},
		{"repository.go", "templates/domain/repository.go.tmpl"},
		{"service.go", "templates/domain/service.go.tmpl"},
	}

	result := Result{}
	for _, t := range targets {
		fr, err := generateFile(root, filepath.Join(domainDir, t.relName), t.tmpl, data, opts.Force)
		if err != nil {
			return Result{}, err
		}
		result.Files = append(result.Files, fr)
	}

	return result, nil
}
