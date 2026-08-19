package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/exgamer/go-sdk-generator/internal/gomod"
)

// BootstrapOptions controls what AddBootstrap generates.
type BootstrapOptions struct {
	// RootDir is the project root. Defaults to "." when empty.
	RootDir string
	// Domain and Module identify the module to bootstrap — every layer
	// already generated for <domain>/<module> is auto-detected from the
	// filesystem and wired together; nothing else needs to be specified.
	Domain string
	Module string
	// Force overwrites factory/module.go files that already exist. Never
	// affects whether the module gets (re-)registered in app.go — that's
	// always idempotent regardless of Force (see RegisterModule).
	Force bool
}

type bootstrapTarget struct {
	relName string
	tmpl    string
}

// AddBootstrap generates the bootstrap layer for a module — factories
// (repositories/services/handlers, plus consumers if a rabbit consumer
// exists) and module.go — wiring together whichever domain/infra/
// entrypoints layers were already generated for <domain>/<module>, then
// registers the Module in internal/app/app.go. Follows the pattern of
// internal/app/bootstrap/city/ in go-sdk-rest-template.
//
// Everything is auto-detected from the filesystem rather than re-specified:
//   - postgres infra is required (it's the only thing that implements
//     domain.Repository; error if internal/infrastructure/postgres/... is
//     missing, pointing at `infra add postgres`)
//   - redis / http-client infra are wired into repositoriesFactory if
//     present, otherwise skipped
//   - at least one of the admin/client HTTP entrypoints is required —
//     without one, servicesFactory/repositoriesFactory would be dead code,
//     since the generated rabbit consumer (like the real city one) never
//     calls into the domain service on its own
//   - the rabbit consumer, if present, is additionally wired in
//
// Bootstrap path is always internal/app/bootstrap/{domain}/{module}/ (not
// go-sdk-rest-template's flat bootstrap/{module}/ default) — this generator
// handles many domains, and a flat path would collide if two domains ever
// have a same-named module.
func AddBootstrap(opts BootstrapOptions) (InfraResult, error) {
	root := opts.RootDir

	if root == "" {
		root = "."
	}

	if err := validateDomainModule(opts.Domain, opts.Module); err != nil {
		return InfraResult{}, err
	}

	parsed, err := ParseDomain(root, opts.Domain, opts.Module)

	if err != nil {
		return InfraResult{}, err
	}

	if !dirExists(filepath.Join(root, "internal", "infrastructure", "postgres", opts.Domain, opts.Module)) {
		return InfraResult{}, fmt.Errorf("internal/infrastructure/postgres/%s/%s not found — run `codegen infra add postgres %s/%s` first", opts.Domain, opts.Module, opts.Domain, opts.Module)
	}

	hasRedis := dirExists(filepath.Join(root, "internal", "infrastructure", "redis", opts.Domain, opts.Module))
	hasHttpClient := dirExists(filepath.Join(root, "internal", "infrastructure", "http", opts.Domain, opts.Module))
	hasAdmin := dirExists(filepath.Join(root, "internal", "entrypoints", "admin", "http", opts.Domain, opts.Module))
	hasClient := dirExists(filepath.Join(root, "internal", "entrypoints", "client", "http", opts.Domain, opts.Module))
	hasConsumer := dirExists(filepath.Join(root, "internal", "entrypoints", "consumers", "rabbit", opts.Domain, opts.Module))

	if !hasAdmin && !hasClient {
		return InfraResult{}, fmt.Errorf(
			"no HTTP entrypoint found for %s/%s — run `codegen entrypoints add http admin %s/%s` (or client) first: "+
				"bootstrap needs at least one to wire the generated service into",
			opts.Domain, opts.Module, opts.Domain, opts.Module,
		)
	}

	modulePath, err := gomod.ModulePath(root)

	if err != nil {
		return InfraResult{}, fmt.Errorf("determine module path (run `go mod init` first): %w", err)
	}

	base := modulePath + "/internal"

	data := struct {
		Module           string
		EntityName       string
		DomainImport     string
		PostgresImport   string
		RedisImport      string
		HttpClientImport string
		ConfigsImport    string
		AdminImport      string
		ClientImport     string
		ConsumerImport   string
		HasRedis         bool
		HasHttpClient    bool
		HasAdmin         bool
		HasClient        bool
		HasConsumer      bool
	}{
		Module:           parsed.Module,
		EntityName:       parsed.EntityName,
		DomainImport:     base + "/domains/" + opts.Domain + "/" + opts.Module,
		PostgresImport:   base + "/infrastructure/postgres/" + opts.Domain + "/" + opts.Module,
		RedisImport:      base + "/infrastructure/redis/" + opts.Domain + "/" + opts.Module,
		HttpClientImport: base + "/infrastructure/http/" + opts.Domain + "/" + opts.Module,
		ConfigsImport:    base + "/configs",
		AdminImport:      base + "/entrypoints/admin/http/" + opts.Domain + "/" + opts.Module,
		ClientImport:     base + "/entrypoints/client/http/" + opts.Domain + "/" + opts.Module,
		ConsumerImport:   base + "/entrypoints/consumers/rabbit/" + opts.Domain + "/" + opts.Module,
		HasRedis:         hasRedis,
		HasHttpClient:    hasHttpClient,
		HasAdmin:         hasAdmin,
		HasClient:        hasClient,
		HasConsumer:      hasConsumer,
	}

	bootstrapDir := filepath.Join("internal", "app", "bootstrap", opts.Domain, opts.Module)

	targets := []bootstrapTarget{
		{"repositories_factory.go", "templates/bootstrap/repositories_factory.go.tmpl"},
		{"services_factory.go", "templates/bootstrap/services_factory.go.tmpl"},
		{"handlers_factory.go", "templates/bootstrap/handlers_factory.go.tmpl"},
		{"module.go", "templates/bootstrap/module.go.tmpl"},
	}

	if hasConsumer {
		targets = append(targets, bootstrapTarget{"consumers_factory.go", "templates/bootstrap/consumers_factory.go.tmpl"})
	}

	result := InfraResult{Modules: bootstrapModules(hasRedis, hasConsumer, hasHttpClient)}

	for _, t := range targets {
		fr, err := generateFile(root, filepath.Join(bootstrapDir, t.relName), t.tmpl, data, opts.Force)

		if err != nil {
			return InfraResult{}, err
		}

		result.Files = append(result.Files, fr)
	}

	registered, err := RegisterModule(root, opts.Domain, opts.Module)

	if err != nil {
		return InfraResult{}, err
	}

	result.ModuleRegistered = registered

	return result, nil
}

// bootstrapModules returns the module paths module.go always/conditionally
// needs: gosdk-core (app.App) and gosdk-postgres-core (postgres DI helper)
// unconditionally — infra add postgres itself doesn't require the postgres
// kernel package, only `kernel add postgres`/`init --kernels=postgres` does,
// so this is the first place that's guaranteed to need it — plus
// gosdk-redis-core/gosdk-rabbit-core/viper when those layers are wired in.
func bootstrapModules(hasRedis, hasConsumer, hasHttpClient bool) []string {
	modules := []string{modulesBase + "gosdk-core", modulesBase + "gosdk-postgres-core"}

	if hasRedis {
		modules = append(modules, modulesBase+"gosdk-redis-core")
	}

	if hasConsumer {
		modules = append(modules, modulesBase+"gosdk-rabbit-core")
	}

	if hasHttpClient {
		modules = append(modules, "github.com/spf13/viper")
	}

	return modules
}

func dirExists(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}
