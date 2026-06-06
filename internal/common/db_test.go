package common

import (
	"path/filepath"
	"testing"

	"github.com/AndriyKalashnykov/gqlgen-gorm/graph/customTypes"
)

// InitDb must honour DB_DSN and apply the Todo migration.
func TestInitDbMigratesTodo(t *testing.T) {
	t.Setenv("DB_DSN", filepath.Join(t.TempDir(), "test.db"))

	db, err := InitDb()
	if err != nil {
		t.Fatalf("InitDb: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if !db.Migrator().HasTable(&customTypes.Todo{}) {
		t.Fatal("todos table was not migrated")
	}
}
