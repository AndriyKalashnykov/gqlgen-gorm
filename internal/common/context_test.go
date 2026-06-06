package common

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/gorm"
)

// CreateContext must inject the CustomContext (with its DB handle) into the
// request context so downstream resolvers can retrieve it via GetContext.
func TestCreateContextInjectsDatabase(t *testing.T) {
	db := &gorm.DB{}
	var got *CustomContext

	probe := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = GetContext(r.Context())
	})

	handler := CreateContext(&CustomContext{Database: db}, probe)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/query", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got == nil {
		t.Fatal("GetContext returned nil inside the handler")
	}
	if got.Database != db {
		t.Fatalf("injected DB handle not propagated: got %p, want %p", got.Database, db)
	}
}

// GetContext must return nil (not panic) when no CustomContext was injected.
func TestGetContextMissingReturnsNil(t *testing.T) {
	if cc := GetContext(context.Background()); cc != nil {
		t.Fatalf("expected nil for a bare context, got %+v", cc)
	}
}
