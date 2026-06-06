// Package common provides shared database and HTTP request-context plumbing.
package common

import (
	"context"
	"net/http"

	"gorm.io/gorm"
)

// CustomContext carries per-request dependencies (the database handle)
// through the request context to the GraphQL resolvers.
type CustomContext struct {
	Database *gorm.DB
}

// contextKey is a private type for context keys to avoid collisions with
// keys defined in other packages.
type contextKey string

const customContextKey contextKey = "CUSTOM_CONTEXT"

// CreateContext returns middleware that injects args into each request's
// context so resolvers can retrieve it via GetContext.
func CreateContext(args *CustomContext, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customContext := &CustomContext{
			Database: args.Database,
		}
		requestWithCtx := r.WithContext(context.WithValue(r.Context(), customContextKey, customContext))
		next.ServeHTTP(w, requestWithCtx)
	})
}

// GetContext returns the CustomContext stored in ctx, or nil if none was set.
func GetContext(ctx context.Context) *CustomContext {
	customContext, ok := ctx.Value(customContextKey).(*CustomContext)
	if !ok {
		return nil
	}
	return customContext
}
