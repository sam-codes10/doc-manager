package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"doc-manager/models"
	"doc-manager/resources"

	"github.com/gocql/gocql"
)

const DocumentEventsChannel = "document_events"

// StartDocumentConsumer subscribes to the classic Redis Pub/Sub channel and processes incoming document events
func StartDocumentConsumer(ctx context.Context) {
	if resources.RDB == nil {
		log.Println("[Redis Subscriber] Redis client not initialized; subscriber cannot start")
		return
	}

	pubsub := resources.RDB.Subscribe(ctx, DocumentEventsChannel)
	defer pubsub.Close()

	// Wait for confirmation that subscription is active
	_, err := pubsub.Receive(ctx)
	if err != nil {
		log.Printf("[Redis Subscriber] Failed to subscribe to channel %q: %v", DocumentEventsChannel, err)
		return
	}

	log.Printf("[Redis Subscriber] Subscribed to channel %q, listening for events...", DocumentEventsChannel)

	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Redis Subscriber] Context canceled, stopping subscriber...")
			return
		case msg, ok := <-ch:
			if !ok {
				log.Println("[Redis Subscriber] Channel closed, stopping consumer...")
				return
			}
			processDocumentMessage(msg.Payload)
		}
	}
}

func processDocumentMessage(payload string) {
	var doc models.Document
	if err := json.Unmarshal([]byte(payload), &doc); err != nil {
		log.Printf("[Redis Subscriber] Failed to unmarshal document payload: %v", err)
		return
	}

	log.Printf("[Redis Subscriber] Received document ID: %s, Name: %s, Size: %d bytes",
		doc.ID, doc.Name, doc.Size)

	// Run dummy processing logic with retries and random record selection
	processedDoc, err := mockProcessingDocument(doc)
	if err != nil {
		log.Printf("[Redis Subscriber] Mock processing failed for document %s: %v", doc.ID, err)
		return
	}

	// Update PostgreSQL record status and extracted content if DB is available
	if resources.DB != nil {
		var extracted interface{}
		if processedDoc.ExtractedContent != nil {
			extracted = *processedDoc.ExtractedContent
		}
		_, err := resources.DB.Exec(
			context.Background(),
			`UPDATE documents SET status = $1, extracted_content = $2, rejection_reason = $3, updated_at = NOW() WHERE id = $4`,
			processedDoc.Status,
			extracted,
			processedDoc.RejectionReason,
			processedDoc.ID,
		)
		if err != nil {
			log.Printf("[Postgres Update] Failed to update document %s: %v", processedDoc.ID, err)
		} else {
			log.Printf("[Postgres Update] Document %s status updated to %q", processedDoc.ID, processedDoc.Status)
		}
	}

	// Persist snapshot to Cassandra document_snapshots table
	if resources.Session != nil {
		docUUID, err := gocql.ParseUUID(processedDoc.ID)
		if err != nil {
			log.Printf("[Cassandra Snapshot] Invalid UUID %q: %v", processedDoc.ID, err)
		} else {
			snapshotJSON, _ := json.Marshal(processedDoc)
			query := `INSERT INTO document_snapshots (document_id, status, db_snapshot, timestamp) VALUES (?, ?, ?, ?)`
			now := time.Now().UTC()
			if err := resources.Session.Query(query, docUUID, processedDoc.Status, string(snapshotJSON), now).Exec(); err != nil {
				log.Printf("[Cassandra Snapshot] Failed to save snapshot for %s: %v", processedDoc.ID, err)
			} else {
				log.Printf("[Cassandra Snapshot] Successfully persisted snapshot for document %s (status: %s) at %s",
					processedDoc.ID, processedDoc.Status, now.Format(time.RFC3339))
			}
		}
	}

	log.Printf("[Redis Subscriber] Finished processing document ID: %s", processedDoc.ID)
}

// mockProcessingDocument simulates document processing with S3 file download, time wait, 3 retries, and random record selection
func mockProcessingDocument(doc models.Document) (models.Document, error) {
	sample1 := `{"extracted_text": "Invoice #1024 - Total: $500.00", "pages": 1, "ocr_confidence": 0.98}`
	sample2 := `{"extracted_text": "Quarterly Financial Report Q3", "pages": 4, "ocr_confidence": 0.94}`
	sample3 := `{"extracted_text": "Employment Agreement Signed", "pages": 2, "ocr_confidence": 0.99}`
	sample4 := `{"extracted_text": "Hardware Purchase Receipt", "pages": 1, "ocr_confidence": 0.91}`

	// Dummy array of records to be chosen randomly
	dummyRecords := []struct {
		status           string
		extractedContent *string
		rejectionReason  *string
	}{
		{
			status:           "processed",
			extractedContent: &sample1,
			rejectionReason:  nil,
		},
		{
			status:           "processed",
			extractedContent: &sample2,
			rejectionReason:  nil,
		},
		{
			status:           "completed",
			extractedContent: &sample3,
			rejectionReason:  nil,
		},
		{
			status:           "completed",
			extractedContent: &sample4,
			rejectionReason:  nil,
		},
	}

	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[Document Processor] Processing document %s (attempt %d/%d)...", doc.ID, attempt, maxRetries)

		// Download the file using S3 URL
		if doc.Path != "" {
			log.Printf("[Document Processor] Downloading file using S3 URL: %s", doc.Path)
			fileBytes, err := resources.DownloadFromS3(context.Background(), doc.Path)
			if err != nil {
				lastErr = fmt.Errorf("failed to download file from S3: %w", err)
				log.Printf("[Document Processor] S3 download attempt %d failed: %v. Retrying...", attempt, lastErr)
				time.Sleep(150 * time.Millisecond)
				continue
			}
			log.Printf("[Document Processor] Successfully downloaded %d bytes from S3 URL for document %s", len(fileBytes), doc.ID)
		}

		// Time wait to simulate async processing delay
		time.Sleep(200 * time.Millisecond)

		// Simulate possible transient failure on attempt < maxRetries (10% probability)
		if attempt < maxRetries && rand.Float64() < 0.10 {
			lastErr = fmt.Errorf("transient timeout connecting to OCR parsing engine")
			log.Printf("[Document Processor] Attempt %d failed: %v. Retrying...", attempt, lastErr)
			time.Sleep(150 * time.Millisecond)
			continue
		}

		// Randomly select one dummy record from the array
		chosenIndex := rand.Intn(len(dummyRecords))
		chosen := dummyRecords[chosenIndex]

		// Apply the chosen mock record fields
		doc.Status = chosen.status
		doc.ExtractedContent = chosen.extractedContent
		doc.RejectionReason = chosen.rejectionReason
		doc.UpdatedAt = time.Now().UTC()

		log.Printf("[Document Processor] Successfully processed document %s on attempt %d with dummy record index %d (status: %s)",
			doc.ID, attempt, chosenIndex, doc.Status)
		return doc, nil
	}

	return doc, fmt.Errorf("failed to process document after %d attempts: %w", maxRetries, lastErr)
}
