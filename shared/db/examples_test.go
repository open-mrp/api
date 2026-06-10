package db_test

import (
	"github.com/augno/api/shared/db"
)

// ExampleNewDbPool shows the minimal configuration for creating a connection
// pool: only DBURI is required; all other fields receive production defaults.
func ExampleNewDbPool() {
	pool, err := db.NewDbPool(&db.Config{
		DBURI: "user:pass@tcp(localhost:3306)/app",
	})
	if err != nil {
		panic(err)
	}
	defer pool.Close()
}
