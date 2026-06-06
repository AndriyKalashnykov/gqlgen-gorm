//go:build integration

package graph_test

import (
	"path/filepath"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/AndriyKalashnykov/gqlgen-gorm/graph/customTypes"
	"github.com/AndriyKalashnykov/gqlgen-gorm/graph/generated"
	resolvers "github.com/AndriyKalashnykov/gqlgen-gorm/graph/resolvers"
	"github.com/AndriyKalashnykov/gqlgen-gorm/internal/common"
)

type todo struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// newTestClient wires the real gqlgen executable schema and resolvers over an
// isolated temp-file SQLite database, exercising the same handler chain as the
// production server (CreateContext middleware + GetContext in the resolvers).
func newTestClient(t *testing.T) (*client.Client, *gorm.DB) {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&customTypes.Todo{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	srv := handler.NewDefaultServer(
		generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers.Resolver{}}),
	)
	h := common.CreateContext(&common.CustomContext{Database: db}, srv)
	return client.New(h), db
}

func TestTodoLifecycle(t *testing.T) {
	c, db := newTestClient(t)

	countRows := func() int64 {
		var n int64
		if err := db.Model(&customTypes.Todo{}).Count(&n).Error; err != nil {
			t.Fatalf("count rows: %v", err)
		}
		return n
	}

	// createTodo
	var created struct{ CreateTodo todo }
	c.MustPost(`mutation { createTodo(text: "todo1") { id text done } }`, &created)
	if created.CreateTodo.ID != 1 || created.CreateTodo.Text != "todo1" || created.CreateTodo.Done {
		t.Fatalf("createTodo = %+v, want {1 todo1 false}", created.CreateTodo)
	}
	if n := countRows(); n != 1 {
		t.Fatalf("row count after create = %d, want 1", n)
	}

	// getTodos
	var listed struct{ GetTodos []todo }
	c.MustPost(`{ getTodos { id text done } }`, &listed)
	if len(listed.GetTodos) != 1 {
		t.Fatalf("getTodos len = %d, want 1", len(listed.GetTodos))
	}

	// getTodo
	var fetched struct{ GetTodo todo }
	c.MustPost(`{ getTodo(todoId: 1) { id text done } }`, &fetched)
	if fetched.GetTodo.Text != "todo1" {
		t.Fatalf("getTodo text = %q, want todo1", fetched.GetTodo.Text)
	}

	// updateTodo
	var updated struct{ UpdateTodo todo }
	c.MustPost(`mutation { updateTodo(input: {id: 1, text: "todo", done: true}) { id text done } }`, &updated)
	if !updated.UpdateTodo.Done || updated.UpdateTodo.Text != "todo" {
		t.Fatalf("updateTodo = %+v, want text=todo done=true", updated.UpdateTodo)
	}

	// deleteTodo — regression guard: must return the deleted row (non-null
	// Todo!), not nil, and must remove it from the database.
	var deleted struct{ DeleteTodo todo }
	c.MustPost(`mutation { deleteTodo(todoId: 1) { id text done } }`, &deleted)
	if deleted.DeleteTodo.ID != 1 || deleted.DeleteTodo.Text != "todo" {
		t.Fatalf("deleteTodo = %+v, want the deleted row {1 todo true}", deleted.DeleteTodo)
	}
	if n := countRows(); n != 0 {
		t.Fatalf("row count after delete = %d, want 0", n)
	}
}

func TestDeleteTodoNotFound(t *testing.T) {
	c, _ := newTestClient(t)

	var resp struct{ DeleteTodo *todo }
	err := c.Post(`mutation { deleteTodo(todoId: 999) { id text done } }`, &resp)
	if err == nil {
		t.Fatal("expected an error deleting a non-existent todo, got nil")
	}
}
