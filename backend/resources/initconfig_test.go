package resources

import (
	"testing"
)

func TestInitConfig(t *testing.T) {
	err := InitConfig("secrets.json")
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	if PostgresCfg.User != "postgres" {
		t.Errorf("expected PostgresCfg.User 'postgres', got %q", PostgresCfg.User)
	}

	if RedisCfg.Addr != "localhost:6379" {
		t.Errorf("expected RedisCfg.Addr 'localhost:6379', got %q", RedisCfg.Addr)
	}

	if len(CassandraCfg.Hosts) == 0 || CassandraCfg.Hosts[0] != "127.0.0.1" {
		t.Errorf("expected CassandraCfg.Hosts to contain '127.0.0.1', got %v", CassandraCfg.Hosts)
	}
}
