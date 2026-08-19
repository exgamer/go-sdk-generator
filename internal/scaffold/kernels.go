package scaffold

import "fmt"

const modulesBase = "github.com/exgamer/"

// kernelSpec describes everything needed to wire one kernel into app.go and
// to fetch its module.
type kernelSpec struct {
	module string // module path, for `go get`
	alias  string // import alias used in app.go
	argSrc string // Go source for the RegisterAndInitKernels argument
}

func (s kernelSpec) importPath() string {
	return s.module + "/pkg/app"
}

var kernelSpecs = map[string]kernelSpec{
	"postgres": {
		module: modulesBase + "gosdk-postgres-core",
		alias:  "postgres",
		argSrc: "&postgres.PostgresKernel{}",
	},
	"http": {
		module: modulesBase + "gosdk-http-core",
		alias:  "http",
		argSrc: "&http.HttpKernel{}",
	},
	"rabbit": {
		module: modulesBase + "gosdk-rabbit-core",
		alias:  "rabbitapp",
		argSrc: "rabbitapp.NewRabbitKernel().EnableConsumer().EnablePublisher()",
	},
	"redis": {
		module: modulesBase + "gosdk-redis-core",
		alias:  "redis",
		argSrc: "&redis.RedisKernel{}",
	},
}

func validateKernels(kernels []string) error {
	for _, k := range kernels {
		if _, ok := kernelSpecs[k]; !ok {
			return fmt.Errorf("unknown kernel %q (allowed: postgres, http, rabbit, redis)", k)
		}
	}

	return nil
}

// RequiredModules returns the module paths needed in go.mod for the given
// kernels, plus gosdk-core which app.go always imports.
func RequiredModules(kernels []string) []string {
	return append([]string{modulesBase + "gosdk-core"}, KernelModules(kernels)...)
}

// KernelModules returns the module paths for the given kernels (without
// gosdk-core).
func KernelModules(kernels []string) []string {
	modules := make([]string, 0, len(kernels))

	for _, k := range kernels {
		if spec, ok := kernelSpecs[k]; ok {
			modules = append(modules, spec.module)
		}
	}

	return modules
}

// DomainModules returns the module paths the generated domain layer needs
// for the given methods: gosdk-db-core (pagination.Paginated), only when
// Paginated is among them.
func DomainModules(methods []string) []string {
	for _, m := range methods {
		if m == "paginated" {
			return []string{modulesBase + "gosdk-db-core"}
		}
	}

	return nil
}
