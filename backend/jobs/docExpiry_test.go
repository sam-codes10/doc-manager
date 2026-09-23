package jobs

import (
	"context"
	"testing"
	"time"

	"doc-manager/resources"
)

func TestSetDocExpiry(t *testing.T) {
	_ = resources.InitConfig("../resources/secrets.json")
	if resources.DB == nil {
		_, err := resources.Connect(context.Background(), resources.PostgresCfg)
		if err != nil {
			t.Skipf("Skipping doc expiry test: cannot connect to postgres: %v", err)
		}
	}

	ctx := context.Background()

	// Insert a test document created 2 hours ago
	var docID string
	err := resources.DB.QueryRow(ctx, `
		INSERT INTO documents (name, status, created_at, updated_at)
		VALUES ('expiry_test.pdf', 'uploaded', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours')
		RETURNING id
	`).Scan(&docID)
	if err != nil {
		t.Fatalf("failed to insert test document: %v", err)
	}

	// Run SetDocExpiry
	SetDocExpiry(ctx)

	// Verify status updated to 'expired'
	var status string
	var updatedAt time.Time
	err = resources.DB.QueryRow(ctx, `SELECT status, updated_at FROM documents WHERE id = $1`, docID).Scan(&status, &updatedAt)
	if err != nil {
		t.Fatalf("failed to query document: %v", err)
	}

	if status != "expired" {
		t.Errorf("expected status 'expired', got %q", status)
	}

	// Clean up
	_, _ = resources.DB.Exec(ctx, `DELETE FROM documents WHERE id = $1`, docID)
}
