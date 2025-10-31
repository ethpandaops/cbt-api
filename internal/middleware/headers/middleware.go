package headers

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

// Middleware returns an HTTP middleware function that applies headers based on path matching.
// The middleware matches the request path to a policy and sets all configured headers.
func (m *Manager) Middleware(log logrus.FieldLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Match path to policy and get headers
			policyName, headers := m.Match(r.URL.Path)

			if len(headers) > 0 {
				// Set all headers from policy
				for key, value := range headers {
					w.Header().Set(key, value)
				}

				log.WithFields(logrus.Fields{
					"path":   r.URL.Path,
					"policy": policyName,
				}).Debug("applied header policy")
			}

			next.ServeHTTP(w, r)
		})
	}
}
