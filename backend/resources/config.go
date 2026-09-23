package resources

type Config struct {
	Postgres  PostgresConfig  `json:"postgres"`
	Redis     RedisConfig     `json:"redis"`
	Cassandra CassandraConfig `json:"cassandra"`
	S3        S3Config        `json:"s3"`
}

type PostgresConfig struct {
	User           string `json:"user"`
	Password       string `json:"password"`
	Host           string `json:"host"`
	Port           string `json:"port"`
	DatabaseName   string `json:"database_name"`
	SSLMode        string `json:"sslmode"`
	MaxConns       int32  `json:"max_conns"`
	MinConns       int32  `json:"min_conns"`
	AutoMigrate    bool   `json:"auto_migrate"`
	Seed           bool   `json:"seed"`
	MigrationsPath string `json:"migrations_path"`
}

// DbConfig is maintained as an alias for PostgresConfig to ensure backward compatibility
type DbConfig = PostgresConfig

type RedisConfig struct {
	Addr     string `json:"addr"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type CassandraConfig struct {
	Hosts       []string `json:"hosts"`
	Port        int      `json:"port"`
	Keyspace    string   `json:"keyspace"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	Consistency string   `json:"consistency"`
}

type S3Config struct {
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	Endpoint     string `json:"endpoint"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
	UsePathStyle bool   `json:"use_path_style"`
}
