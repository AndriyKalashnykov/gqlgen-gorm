package common

import (
	"fmt"
	"os"

	"github.com/AndriyKalashnykov/gqlgen-gorm/graph/customTypes"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// defaultDSN is the SQLite data source used when DB_DSN is unset.
const defaultDSN = "dev.db"

// InitDb opens the SQLite database (DSN from the DB_DSN env var, falling
// back to dev.db) and applies the schema migration.
func InitDb() (*gorm.DB, error) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = defaultDSN
	}
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database %q: %w", dsn, err)
	}
	if err := db.AutoMigrate(&customTypes.Todo{}); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	return db, nil
}
