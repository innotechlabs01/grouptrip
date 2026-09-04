package authservice

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

func openTestDB(dir string) (*sql.DB, error) {
	db, err := sql.Open("libsql", "file:"+filepath.Join(dir, "test.db"))
	if err != nil {
		return nil, fmt.Errorf("open test db: %w", err)
	}
	return db, nil
}
