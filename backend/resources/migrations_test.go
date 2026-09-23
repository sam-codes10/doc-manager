package resources

import (
	"context"
	"testing"
)

func TestMigrations(t *testing.T) {
	_ = InitConfig("secrets.json")

	// Test Postgres migrations
	if DB == nil {
		_, err := Connect(context.Background(), PostgresCfg)
		if err != nil {
			t.Skipf("Skipping postgres migration test: %v", err)
		}
	}

	err := RunPostgresMigrations(context.Background(), DB, "../migrations/postgres")
	if err != nil {
		t.Fatalf("RunPostgresMigrations failed: %v", err)
	}

	// Test Cassandra migrations
	if Session == nil {
		_, err := ConnectCassandra(context.Background(), CassandraCfg)
		if err != nil {
			t.Skipf("Skipping cassandra migration test: %v", err)
		}
	}

	err = RunCassandraMigrations(context.Background(), Session, "../migrations/cassandra")
	if err != nil {
		t.Fatalf("RunCassandraMigrations failed: %v", err)
	}
}
