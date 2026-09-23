package resources

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	AppConfig    Config
	PostgresCfg  PostgresConfig
	RedisCfg     RedisConfig
	CassandraCfg CassandraConfig
	S3Cfg        S3Config
)

// InitConfig reads secrets.json and unmarshals it into all resource config variables.
// A custom file path can optionally be provided.
func InitConfig(customPath ...string) error {
	filePath := resolveSecretsPath(customPath...)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read secrets file from %q: %w", filePath, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to unmarshal secrets json: %w", err)
	}

	// Initialize all config variables
	AppConfig = cfg
	PostgresCfg = cfg.Postgres
	RedisCfg = cfg.Redis
	CassandraCfg = cfg.Cassandra
	S3Cfg = cfg.S3

	return nil
}

// MustInitConfig initializes configs and panics if an error occurs.
func MustInitConfig(customPath ...string) {
	if err := InitConfig(customPath...); err != nil {
		panic(err)
	}
}

func resolveSecretsPath(customPath ...string) string {
	if len(customPath) > 0 && customPath[0] != "" {
		return customPath[0]
	}

	candidates := []string{
		"resources/secrets.json",
		"secrets.json",
		"../resources/secrets.json",
		"backend/resources/secrets.json",
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return "resources/secrets.json"
}

func init() {
	// Attempt automatic load on initialization; does not panic if file is missing at startup
	_ = InitConfig()
}
