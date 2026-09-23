package main

import (
	"context"
	"fmt"
	"log"

	"doc-manager/jobs"
	"doc-manager/resources"
	"doc-manager/routers"
)

// @title           Doc Manager API
// @version         1.0
// @description     Document Management Service API with PostgreSQL, Redis, Cassandra, and S3 datastores.
// @host            localhost:8080
// @BasePath        /api

func main() {
	// 1. Initialize configuration from secrets.json
	if err := resources.InitConfig(); err != nil {
		log.Fatalf("failed to initialize config: %v", err)
	}
	fmt.Println("configuration initialized successfully")

	ctx := context.Background()

	// 2. Connect to PostgreSQL
	pgPool, err := resources.Connect(ctx, resources.PostgresCfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pgPool.Close()

	// Run PostgreSQL migrations
	if err := resources.RunPostgresMigrations(ctx, pgPool, resources.PostgresCfg.MigrationsPath); err != nil {
		log.Fatalf("failed to run postgres migrations: %v", err)
	}

	// 3. Connect to Redis
	redisClient, err := resources.InitRedis()
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	// 4. Connect to Cassandra
	cassandraSession, err := resources.ConnectCassandra(ctx, resources.CassandraCfg)
	if err != nil {
		log.Printf("warning: failed to connect to cassandra: %v (is cassandra running?)", err)
	} else {
		defer cassandraSession.Close()

		// Run Cassandra migrations
		if err := resources.RunCassandraMigrations(ctx, cassandraSession, "migrations/cassandra"); err != nil {
			log.Fatalf("failed to run cassandra migrations: %v", err)
		}
	}

	// 5. Connect to S3
	_, err = resources.ConnectS3(ctx, resources.S3Cfg)
	if err != nil {
		log.Printf("warning: failed to connect to s3: %v", err)
	}

	fmt.Println("all resources setup complete")

	// 6. Start Redis background consumer worker
	go jobs.StartDocumentConsumer(ctx)

	// 7. Initialize and run HTTP server
	r := routers.InitRouters()
	fmt.Println("server running on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}