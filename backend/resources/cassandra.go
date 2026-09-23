package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

var (
	Cluster *gocql.ClusterConfig
	Session *gocql.Session
)

func ConnectCassandra(ctx context.Context, cfg CassandraConfig) (*gocql.Session, error) {
	hosts := cfg.Hosts
	if len(hosts) == 0 {
		hosts = []string{"127.0.0.1"}
	}

	cluster := gocql.NewCluster(hosts...)
	if cfg.Port > 0 {
		cluster.Port = cfg.Port
	}
	if cfg.Keyspace != "" {
		cluster.Keyspace = cfg.Keyspace
	}
	cluster.Timeout = 5 * time.Second
	cluster.ConnectTimeout = 5 * time.Second
	cluster.DisableInitialHostLookup = true
	cluster.Consistency = gocql.One

	Cluster = cluster

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	Session = session
	fmt.Println("cassandra connected successfully")
	return session, nil
}
