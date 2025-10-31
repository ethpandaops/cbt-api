package headers

import (
	"fmt"
	"regexp"

	"github.com/ethpandaops/cbt-api/internal/config"
)

// Manager handles HTTP header policy matching and application.
type Manager struct {
	policies []compiledPolicy
}

type compiledPolicy struct {
	name    string
	pattern *regexp.Regexp
	headers map[string]string
}

// NewManager creates a new Manager with compiled regex patterns from the provided policies.
// Returns an error if any policy contains an invalid regex pattern.
func NewManager(policies []config.HeaderPolicy) (*Manager, error) {
	compiled := make([]compiledPolicy, 0, len(policies))

	for _, p := range policies {
		pattern, err := regexp.Compile(p.PathPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid path_pattern in policy %q: %w", p.Name, err)
		}

		compiled = append(compiled, compiledPolicy{
			name:    p.Name,
			pattern: pattern,
			headers: p.Headers,
		})
	}

	return &Manager{policies: compiled}, nil
}

// Match finds the first policy that matches the given path and returns its name and headers.
// Returns empty string and nil if no policy matches.
func (m *Manager) Match(path string) (string, map[string]string) {
	for _, p := range m.policies {
		if p.pattern.MatchString(path) {
			return p.name, p.headers
		}
	}

	return "", nil
}
