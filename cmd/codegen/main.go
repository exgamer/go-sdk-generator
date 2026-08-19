// Command codegen scaffolds project files for services built on
// go-sdk-rest-template.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/exgamer/go-sdk-generator/internal/gomod"
	"github.com/exgamer/go-sdk-generator/internal/scaffold"
)

var docsStubPath = filepath.Join("docs", "docs.go")

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	case "init":
		err = runInit(os.Args[2:])
	case "kernel":
		err = runKernel(os.Args[2:])
	case "domain":
		err = runDomain(os.Args[2:])
	case "infra":
		err = runInfra(os.Args[2:])
	case "entrypoints":
		err = runEntrypoints(os.Args[2:])
	case "bootstrap":
		err = runBootstrap(os.Args[2:])
	case "app-name":
		err = runAppName(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "codegen:", err)
		os.Exit(1)
	}
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	appName := fs.String("app-name", "", "app title used in the main.go swagger annotation (default: last segment of the module path)")
	kernelsFlag := fs.String("kernels", "", "comma-separated kernels to register in app.go (postgres,http,rabbit,redis); empty = no kernels")
	force := fs.Bool("force", false, "overwrite main.go / internal/app/app.go if they already exist")

	if err := fs.Parse(args); err != nil {
		return err
	}

	kernels := splitKernels(*kernelsFlag)

	result, err := scaffold.Run(scaffold.Options{
		AppTitle: *appName,
		Kernels:  kernels,
		Force:    *force,
	})

	if err != nil {
		return err
	}

	var anyWritten bool

	for _, f := range result.Files {
		switch {
		case f.Written:
			anyWritten = true
			fmt.Printf("created  %s\n", f.Path)
		case f.Path == docsStubPath:
			fmt.Printf("skipped  %s (already exists — real `swag init` output, never overwritten)\n", f.Path)
		default:
			fmt.Printf("skipped  %s (already exists, use --force to overwrite)\n", f.Path)
		}
	}

	if anyWritten {
		if err := getAndTidy(scaffold.RequiredModules(kernels)); err != nil {
			return err
		}
	}

	return nil
}

func runKernel(args []string) error {
	if len(args) == 0 || args[0] != "add" {
		return fmt.Errorf("usage: codegen kernel add <kernel>[,<kernel>...]")
	}

	fs := flag.NewFlagSet("kernel add", flag.ExitOnError)

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	rest := fs.Args()

	if len(rest) != 1 {
		return fmt.Errorf("usage: codegen kernel add <kernel>[,<kernel>...]")
	}

	kernels := splitKernels(rest[0])

	if len(kernels) == 0 {
		return fmt.Errorf("no kernels given")
	}

	results, err := scaffold.AddKernel(".", kernels)

	if err != nil {
		return err
	}

	var added []string

	for _, r := range results {
		if r.Added {
			fmt.Printf("added    %s\n", r.Kernel)
			added = append(added, r.Kernel)
		} else {
			fmt.Printf("skipped  %s (already registered)\n", r.Kernel)
		}
	}

	if len(added) > 0 {
		if err := getAndTidy(scaffold.KernelModules(added)); err != nil {
			return err
		}
	}

	return nil
}

func runDomain(args []string) error {
	if len(args) == 0 || args[0] != "add" {
		return fmt.Errorf("usage: codegen domain add --fields=Name:type[,Name:type...] [--methods=...] [--force] <domain>/<module>")
	}

	fs := flag.NewFlagSet("domain add", flag.ExitOnError)
	fieldsFlag := fs.String("fields", "", "entity fields as Name:type,Name2:type2 (allowed types: string,bool,int,int64,uint,uint64,float32,float64)")
	methodsFlag := fs.String("methods", "", "comma-separated Repository/Service methods to generate (paginated,getbyid,create,update,delete,activate,deactivate); empty = all")
	force := fs.Bool("force", false, "overwrite entity.go/dto.go/repository.go/service.go if they already exist")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	rest := fs.Args()

	if len(rest) != 1 {
		return fmt.Errorf("usage: codegen domain add --fields=Name:type[,Name:type...] [--methods=...] [--force] <domain>/<module>")
	}

	domain, module, err := splitDomainModule(rest[0])

	if err != nil {
		return err
	}

	fields, err := scaffold.ParseFields(*fieldsFlag)

	if err != nil {
		return err
	}

	methods, err := scaffold.ParseMethods(*methodsFlag)

	if err != nil {
		return err
	}

	result, err := scaffold.AddDomain(scaffold.DomainOptions{
		Domain:  domain,
		Module:  module,
		Fields:  fields,
		Methods: methods,
		Force:   *force,
	})

	if err != nil {
		return err
	}

	var anyWritten bool

	for _, f := range result.Files {
		if f.Written {
			anyWritten = true
			fmt.Printf("created  %s\n", f.Path)
		} else {
			fmt.Printf("skipped  %s (already exists, use --force to overwrite)\n", f.Path)
		}
	}

	if anyWritten {
		if modules := scaffold.DomainModules(methods); len(modules) > 0 {
			if err := getAndTidy(modules); err != nil {
				return err
			}
		}
	}

	return nil
}

func runInfra(args []string) error {
	if len(args) < 2 || args[0] != "add" {
		return fmt.Errorf("usage: codegen infra add <postgres|redis|http> [--force] <domain>/<module>")
	}

	kind := args[1]

	if kind != "postgres" && kind != "redis" && kind != "http" {
		return fmt.Errorf("unknown infra kind %q (allowed: postgres, redis, http)", kind)
	}

	fs := flag.NewFlagSet("infra add "+kind, flag.ExitOnError)
	force := fs.Bool("force", false, "overwrite generated files if they already exist")

	if err := fs.Parse(args[2:]); err != nil {
		return err
	}

	rest := fs.Args()

	if len(rest) != 1 {
		return fmt.Errorf("usage: codegen infra add <postgres|redis|http> [--force] <domain>/<module>")
	}

	domain, module, err := splitDomainModule(rest[0])

	if err != nil {
		return err
	}

	var result scaffold.InfraResult

	switch kind {
	case "postgres":
		result, err = scaffold.AddInfraPostgres(scaffold.InfraPostgresOptions{
			Domain: domain,
			Module: module,
			Force:  *force,
		})
	case "redis":
		result, err = scaffold.AddInfraRedis(scaffold.InfraRedisOptions{
			Domain: domain,
			Module: module,
			Force:  *force,
		})
	case "http":
		result, err = scaffold.AddInfraHttp(scaffold.InfraHttpOptions{
			Domain: domain,
			Module: module,
			Force:  *force,
		})
	}

	if err != nil {
		return err
	}

	return printResultAndTidy(result)
}

func runEntrypoints(args []string) error {
	if len(args) < 2 || args[0] != "add" {
		return fmt.Errorf("usage: codegen entrypoints add <http|rabbit> ... <domain>/<module>")
	}

	switch args[1] {
	case "http":
		return runEntrypointsHttp(args[2:])
	case "rabbit":
		return runEntrypointsRabbit(args[2:])
	default:
		return fmt.Errorf("unknown entrypoints transport %q (allowed: http, rabbit)", args[1])
	}
}

func runEntrypointsHttp(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: codegen entrypoints add http <admin|client> [--prefix=/x] [--force] <domain>/<module>")
	}

	kind := args[0]

	if kind != "admin" && kind != "client" {
		return fmt.Errorf("unknown entrypoint kind %q (allowed: admin, client)", kind)
	}

	fs := flag.NewFlagSet("entrypoints add http "+kind, flag.ExitOnError)
	prefix := fs.String("prefix", "", "route group prefix (default: /<last segment of the module path>)")
	force := fs.Bool("force", false, "overwrite generated files if they already exist")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	rest := fs.Args()

	if len(rest) != 1 {
		return fmt.Errorf("usage: codegen entrypoints add http <admin|client> [--prefix=/x] [--force] <domain>/<module>")
	}

	domain, module, err := splitDomainModule(rest[0])

	if err != nil {
		return err
	}

	result, err := scaffold.AddEntrypointsHttp(scaffold.EntrypointsHttpOptions{
		Kind:   kind,
		Domain: domain,
		Module: module,
		Prefix: *prefix,
		Force:  *force,
	})

	if err != nil {
		return err
	}

	return printResultAndTidy(result)
}

func runEntrypointsRabbit(args []string) error {
	fs := flag.NewFlagSet("entrypoints add rabbit", flag.ExitOnError)
	force := fs.Bool("force", false, "overwrite generated files if they already exist")

	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()

	if len(rest) != 1 {
		return fmt.Errorf("usage: codegen entrypoints add rabbit [--force] <domain>/<module>")
	}

	domain, module, err := splitDomainModule(rest[0])

	if err != nil {
		return err
	}

	result, err := scaffold.AddEntrypointsRabbit(scaffold.EntrypointsRabbitOptions{
		Domain: domain,
		Module: module,
		Force:  *force,
	})

	if err != nil {
		return err
	}

	return printResultAndTidy(result)
}

func runBootstrap(args []string) error {
	if len(args) == 0 || args[0] != "add" {
		return fmt.Errorf("usage: codegen bootstrap add [--force] <domain>/<module>")
	}

	fs := flag.NewFlagSet("bootstrap add", flag.ExitOnError)
	force := fs.Bool("force", false, "overwrite factory/module.go files that already exist")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	rest := fs.Args()

	if len(rest) != 1 {
		return fmt.Errorf("usage: codegen bootstrap add [--force] <domain>/<module>")
	}

	domain, module, err := splitDomainModule(rest[0])

	if err != nil {
		return err
	}

	result, err := scaffold.AddBootstrap(scaffold.BootstrapOptions{
		Domain: domain,
		Module: module,
		Force:  *force,
	})

	if err != nil {
		return err
	}

	if err := printResultAndTidy(result); err != nil {
		return err
	}

	if result.ModuleRegistered {
		fmt.Printf("updated  internal/app/app.go (registered %s/%s)\n", domain, module)
	} else {
		fmt.Println("skipped  internal/app/app.go (module already registered)")
	}

	return nil
}

// printResultAndTidy prints created/skipped for each file and, if anything
// was actually written, runs `go get` + `go mod tidy` for result.Modules.
func printResultAndTidy(result scaffold.InfraResult) error {
	var anyWritten bool

	for _, f := range result.Files {
		if f.Written {
			anyWritten = true
			fmt.Printf("created  %s\n", f.Path)
		} else {
			fmt.Printf("skipped  %s (already exists, use --force to overwrite)\n", f.Path)
		}
	}

	if anyWritten && len(result.Modules) > 0 {
		if err := getAndTidy(result.Modules); err != nil {
			return err
		}
	}

	return nil
}

func splitDomainModule(s string) (domain, module string, err error) {
	parts := strings.Split(s, "/")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid <domain>/<module> %q, expected e.g. handbook/city", s)
	}

	return parts[0], parts[1], nil
}

func runAppName(args []string) error {
	fs := flag.NewFlagSet("app-name", flag.ExitOnError)

	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()

	if len(rest) != 1 {
		return fmt.Errorf("usage: codegen app-name <name>")
	}

	if err := scaffold.SetAppName(".", rest[0]); err != nil {
		return err
	}

	fmt.Printf("updated  main.go (app name: %s)\n", rest[0])

	return nil
}

func splitKernels(flagValue string) []string {
	var kernels []string

	for _, k := range strings.Split(flagValue, ",") {
		k = strings.TrimSpace(k)

		if k != "" {
			kernels = append(kernels, k)
		}
	}

	return kernels
}

func getAndTidy(modules []string) error {
	fmt.Printf("go get   %s\n", strings.Join(modules, " "))

	return gomod.RegisterDependencies(".", modules)
}

func printUsage() {
	fmt.Println(`usage: codegen <command> [flags]

commands:
  init          generate starter main.go and internal/app/app.go in an empty project
  kernel add    register additional kernels (postgres,http,rabbit,redis) in an existing app.go
  domain add    generate the domain layer (entity/dto/repository/service) for a new module
  infra add     generate an infrastructure layer (postgres, redis, http) from an existing domain
  entrypoints add   generate an entrypoints layer (http admin/client, or rabbit consumer) from an existing domain
  bootstrap add     wire up domain+infra+entrypoints into internal/app/bootstrap and register it in app.go
  app-name      change the swagger @title/@description annotations in main.go

run "codegen <command> -h" for flags`)
}
