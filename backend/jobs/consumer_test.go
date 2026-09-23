package jobs

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"doc-manager/models"
	"doc-manager/resources"
)

func TestClassicPubSub(t *testing.T) {
	_ = resources.InitConfig("../resources/secrets.json")
	if resources.RDB == nil {
		_, err := resources.InitRedis()
		if err != nil {
			t.Skipf("Skipping pub/sub test: cannot connect to redis: %v", err)
		}
	}

	if resources.Session == nil {
		_, _ = resources.ConnectCassandra(context.Background(), resources.CassandraCfg)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Start consumer in background
	go StartDocumentConsumer(ctx)

	// Allow subscriber connection to register with Redis server
	time.Sleep(150 * time.Millisecond)

	// Upload a dummy test file to S3 for testing S3 download in mockProcessingDocument
	s3URL, _ := resources.UploadToS3(context.Background(), "documents/12345678-1234-1234-1234-123456789abc/test_pubsub.pdf", strings.NewReader("sample test content for s3 download"), "application/pdf")

	// Publish test document message with valid UUID and S3 URL
	testDoc := models.Document{
		ID:     "12345678-1234-1234-1234-123456789abc",
		Name:   "test_pubsub.pdf",
		Path:   s3URL,
		Status: "uploaded",
		Size:   1024,
	}
	payload, _ := json.Marshal(testDoc)

	err := resources.RDB.Publish(context.Background(), DocumentEventsChannel, string(payload)).Err()
	if err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}

	// Give subscriber time to process through retries
	time.Sleep(1500 * time.Millisecond)
}
