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
	"doc-manager/services"
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

	// 1. Mark status as 'processing' in PostgreSQL
	doc.Status = "processing"
	doc.UpdatedAt = time.Now().UTC()
	if resources.DB != nil {
		_, err := resources.DB.Exec(
			context.Background(),
			`UPDATE documents SET status = $1, updated_at = NOW() WHERE id = $2`,
			doc.Status,
			doc.ID,
		)
		if err != nil {
			log.Printf("[Postgres Update] Failed to update document %s to processing: %v", doc.ID, err)
		} else {
			log.Printf("[Postgres Update] Document %s status updated to 'processing'", doc.ID)
		}
	}

	// Record 'processing' event in Cassandra
	services.RecordDocumentEvent(doc)

	// 2. Run mock processing logic with retries and random record selection
	processedDoc, err := mockProcessingDocument(doc)
	if err != nil {
		log.Printf("[Redis Subscriber] Mock processing returned error for document %s: %v", doc.ID, err)
		processedDoc.Status = "failed"
		errMsg := err.Error()
		processedDoc.RejectionReason = &errMsg
		processedDoc.ExtractedContent = nil
		processedDoc.UpdatedAt = time.Now().UTC()
	}

	// 3. Update PostgreSQL record with terminal status (success or failed), extracted content, and rejection reason
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

	// 4. Persist terminal snapshot event to Cassandra
	services.RecordDocumentEvent(processedDoc)

	log.Printf("[Redis Subscriber] Finished processing document ID: %s (final status: %s)", processedDoc.ID, processedDoc.Status)
}

// mockProcessingDocument simulates document processing with S3 file download, time wait, 3 retries, and random record selection
func mockProcessingDocument(doc models.Document) (models.Document, error) {
	sample1 := `{"extracted_text": "Invoice #1024 - Total: $500.00", "pages": 1, "ocr_confidence": 0.98}`
	sample2 := `{"extracted_text": "Quarterly Financial Report Q3", "pages": 4, "ocr_confidence": 0.94}`
	sample3 := `{"extracted_text": "Employment Agreement Signed", "pages": 2, "ocr_confidence": 0.99}`
	sample4 := `{"extracted_text": "Hardware Purchase Receipt", "pages": 1, "ocr_confidence": 0.91}`

	reason1 := "Corrupted image data or unreadable scan quality"
	reason2 := "Unsupported document structure or missing required header fields"
	reason3 := "Document exceeds maximum parsing limit or contains security violation"

	// Dummy array of records with success and failed outcomes
	dummyRecords := []struct {
		status           string
		extractedContent *string
		rejectionReason  *string
	}{
		{
			status:           "success",
			extractedContent: &sample1,
			rejectionReason:  nil,
		},
		{
			status:           "success",
			extractedContent: &sample2,
			rejectionReason:  nil,
		},
		{
			status:           "success",
			extractedContent: &sample3,
			rejectionReason:  nil,
		},
		{
			status:           "success",
			extractedContent: &sample4,
			rejectionReason:  nil,
		},
		{
			status:           "failed",
			extractedContent: nil,
			rejectionReason:  &reason1,
		},
		{
			status:           "failed",
			extractedContent: nil,
			rejectionReason:  &reason2,
		},
		{
			status:           "failed",
			extractedContent: nil,
			rejectionReason:  &reason3,
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

	// If max retries reached without success, mark as failed with rejection reason
	doc.Status = "failed"
	failReason := fmt.Sprintf("failed to process document after %d attempts: %v", maxRetries, lastErr)
	doc.RejectionReason = &failReason
	doc.ExtractedContent = nil
	doc.UpdatedAt = time.Now().UTC()

	return doc, nil
}
