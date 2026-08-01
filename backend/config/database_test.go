package config

import "testing"

func TestDatabaseDSNFromEnvironment(t *testing.T) {
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_USER", "sigap")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "sigap_test")
	t.Setenv("DB_PORT", "5432")

	want := "host=db user=sigap password=secret dbname=sigap_test port=5432 sslmode=disable TimeZone=Asia/Makassar"
	if got := databaseDSN(); got != want {
		t.Fatalf("databaseDSN() = %q, want %q", got, want)
	}
}
