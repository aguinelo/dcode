package integration

import (
	"database/sql"
	"os"
	"testing"
)

// dsn points at the integration database. Nothing in this repository starts it.
func dsn() string {
	if v := os.Getenv("DCODE_TEST_DSN"); v != "" {
		return v
	}
	return "postgres://dcode:dcode@localhost:5432/dcode_test?sslmode=disable"
}

func TestAccountsRoundTrip(t *testing.T) {
	db, err := sql.Open("postgres", dsn())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("the integration database is not reachable: %v", err)
	}

	var n int
	if err := db.QueryRow("select count(*) from accounts").Scan(&n); err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != 3 {
		t.Errorf("accounts = %d, want 3", n)
	}
}
