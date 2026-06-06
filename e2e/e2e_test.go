//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/AndriyKalashnykov/gqlgen-gorm/graph/generated"
	resolvers "github.com/AndriyKalashnykov/gqlgen-gorm/graph/resolvers"
	"github.com/AndriyKalashnykov/gqlgen-gorm/internal/common"
)

// newServer boots the same handler chain as server.go over an ephemeral
// httptest port, backed by an isolated temp SQLite database.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("DB_DSN", filepath.Join(t.TempDir(), "e2e.db"))

	db, err := common.InitDb()
	if err != nil {
		t.Fatalf("init db: %v", err)
	}

	srv := handler.NewDefaultServer(
		generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers.Resolver{}}),
	)
	mux := http.NewServeMux()
	mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
	mux.Handle("/query", common.CreateContext(&common.CustomContext{Database: db}, srv))

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func post(t *testing.T, url, query string, data any) {
	t.Helper()
	reqBody, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody)) //nolint:noctx // test client
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(env.Errors) > 0 {
		t.Fatalf("graphql errors: %+v", env.Errors)
	}
	if data != nil {
		if err := json.Unmarshal(env.Data, data); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
	}
}

func TestPlaygroundServed(t *testing.T) {
	ts := newServer(t)

	resp, err := http.Get(ts.URL + "/") //nolint:noctx // test client
	if err != nil {
		t.Fatalf("get playground: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("playground status = %d, want 200", resp.StatusCode)
	}
}

func TestCreateThenListOverHTTP(t *testing.T) {
	ts := newServer(t)
	url := ts.URL + "/query"

	var created struct {
		CreateTodo struct {
			ID   int    `json:"id"`
			Text string `json:"text"`
			Done bool   `json:"done"`
		} `json:"createTodo"`
	}
	post(t, url, `mutation { createTodo(text: "e2e") { id text done } }`, &created)
	if created.CreateTodo.ID != 1 || created.CreateTodo.Text != "e2e" {
		t.Fatalf("createTodo = %+v, want {1 e2e false}", created.CreateTodo)
	}

	var listed struct {
		GetTodos []struct {
			Text string `json:"text"`
		} `json:"getTodos"`
	}
	post(t, url, `{ getTodos { id text done } }`, &listed)
	if len(listed.GetTodos) != 1 || listed.GetTodos[0].Text != "e2e" {
		t.Fatalf("getTodos = %+v, want one todo 'e2e'", listed.GetTodos)
	}
}
