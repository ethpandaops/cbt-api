package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// CORS returns a middleware that handles CORS headers
func CORS() func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "HEAD", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		MaxAge:         86400, // 24 hours
	})
	return c.Handler
}
