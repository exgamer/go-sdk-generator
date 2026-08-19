package scaffold

import (
	"fmt"
	"strings"
)

// domainMethodOrder is the canonical order methods are rendered in — mirrors
// the Repository interface in the go-sdk-rest-template "city" example.
var domainMethodOrder = []string{
	"paginated",
	"getbyid",
	"create",
	"update",
	"delete",
	"activate",
	"deactivate",
}

var domainMethodSet = func() map[string]bool {
	set := make(map[string]bool, len(domainMethodOrder))

	for _, m := range domainMethodOrder {
		set[m] = true
	}

	return set
}()

// ParseMethods parses a "paginated,getbyid,..." flag value (case-insensitive)
// into a canonical, deduplicated, ordered subset of the Repository/Service
// methods to generate. An empty raw value means "all methods".
func ParseMethods(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return append([]string(nil), domainMethodOrder...), nil
	}

	selected := make(map[string]bool)

	for _, part := range strings.Split(raw, ",") {
		m := strings.ToLower(strings.TrimSpace(part))

		if m == "" {
			continue
		}

		if !domainMethodSet[m] {
			return nil, fmt.Errorf("unknown method %q (allowed: %s)", m, strings.Join(domainMethodOrder, ", "))
		}

		selected[m] = true
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no methods given")
	}

	out := make([]string, 0, len(selected))

	for _, m := range domainMethodOrder {
		if selected[m] {
			out = append(out, m)
		}
	}

	return out, nil
}

// methodFlags is the set of per-method booleans the domain templates branch
// on (Go template field names, so they must be exported).
type methodFlags struct {
	HasPaginated  bool
	HasGetById    bool
	HasCreate     bool
	HasUpdate     bool
	HasDelete     bool
	HasActivate   bool
	HasDeactivate bool
}

func buildMethodFlags(methods []string) methodFlags {
	set := make(map[string]bool, len(methods))

	for _, m := range methods {
		set[m] = true
	}

	return methodFlags{
		HasPaginated:  set["paginated"],
		HasGetById:    set["getbyid"],
		HasCreate:     set["create"],
		HasUpdate:     set["update"],
		HasDelete:     set["delete"],
		HasActivate:   set["activate"],
		HasDeactivate: set["deactivate"],
	}
}
