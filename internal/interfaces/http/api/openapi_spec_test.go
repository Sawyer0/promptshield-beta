package api

import (
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/testutil/mocks"
)

// TestOpenAPISpecMatchesRouter keeps docs/api/openapi.yaml parseable and in
// step with the /v1 routes NewMux registers. Routes the spec says it does not
// cover (debug, tenant, assignment, audit, system admin) are skipped.
func TestOpenAPISpecMatchesRouter(t *testing.T) {
	data, err := os.ReadFile("../../../../docs/api/openapi.yaml")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var spec struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("spec is not valid YAML: %v", err)
	}

	documented := map[string]bool{}
	for path, ops := range spec.Paths {
		if !strings.HasPrefix(path, "/v1/") {
			continue
		}
		for method := range ops {
			if method == "parameters" {
				continue
			}
			documented[strings.ToUpper(method)+" "+path] = true
		}
	}

	mux := NewMux(Options{RulepackService: services.RulepackServiceCstor(&mocks.MockRulepackRepository{}, nil)})
	routes, ok := mux.(chi.Routes)
	if !ok {
		t.Fatalf("NewMux returned %T, want chi.Routes", mux)
	}
	registered := map[string]bool{}
	err = chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = strings.TrimSuffix(route, "/")
		switch {
		case strings.HasPrefix(route, "/v1/admin/"), strings.HasPrefix(route, "/debug/"), route == "/scan:async":
			return nil
		}
		registered[method+" /v1"+route] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	if len(registered) == 0 || len(documented) == 0 {
		t.Fatalf("found %d routes and %d documented operations", len(registered), len(documented))
	}

	var missing, stale []string
	for r := range registered {
		if !documented[r] {
			missing = append(missing, r)
		}
	}
	for d := range documented {
		if !registered[d] {
			stale = append(stale, d)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	if len(missing) > 0 {
		t.Errorf("routes missing from openapi.yaml: %v", missing)
	}
	if len(stale) > 0 {
		t.Errorf("openapi.yaml documents routes that do not exist: %v", stale)
	}
}
