package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"doc-manager/models"
	"doc-manager/resources"

	"github.com/gocql/gocql"
)

var ErrDuplicateDocument = errors.New("duplicate document: a file with the same name and content already exists")

// AcceptDocument saves the uploaded file, inserts a record with DB-generated ID and timestamps, and fires an async worker.
func AcceptDocument(ctx context.Context, file *multipart.FileHeader, typeOfFile string, optionalMeta string) (*models.Document, error) {
	if file == nil {
		return nil, errors.New("file is required")
	}

	if resources.DB == nil {
		return nil, errors.New("database connection not initialized")
	}

	// Read file contents to compute SHA-256 content_hash and check for duplicates
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read uploaded file: %w", err)
	}

	hasher := sha256.New()
	hasher.Write(fileBytes)
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	// Ensure upload directory exists
	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Begin transaction to ensure atomic record creation and file persistence
	tx, err := resources.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check for duplicate document by composite key (content_hash, name)
	var existingID string
	err = tx.QueryRow(ctx, "SELECT id FROM documents WHERE content_hash = $1 AND name = $2", contentHash, file.Filename).Scan(&existingID)
	if err == nil {
		return nil, ErrDuplicateDocument
	}

	// Determine MIME type
	mimeType := strings.TrimSpace(typeOfFile)
	if mimeType == "" {
		mimeType = file.Header.Get("Content-Type")
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	status := "uploaded"

	var docID string
	var createdAt, updatedAt time.Time

	// id (gen_random_uuid()), created_at (CURRENT_TIMESTAMP), and updated_at (CURRENT_TIMESTAMP) handled at DB level
	insertQuery := `
		INSERT INTO documents (name, content_hash, size, mimeType, status, rejection_reason, extracted_content, optional_meta)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRow(
		ctx,
		insertQuery,
		file.Filename,
		contentHash,
		file.Size,
		mimeType,
		status,
		nil,          // rejection_reason
		nil,          // extracted_content
		optionalMeta, // optional_meta
	).Scan(&docID, &createdAt, &updatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "uq_document_content_hash_name") {
			return nil, ErrDuplicateDocument
		}
		return nil, fmt.Errorf("failed to insert document into database: %w", err)
	}

	safeFileName := filepath.Base(file.Filename)
	s3Key := fmt.Sprintf("documents/%s/%s", docID, safeFileName)

	// Upload to S3 and extract the URL
	s3URL, err := resources.UploadToS3(ctx, s3Key, bytes.NewReader(fileBytes), mimeType)
	if err != nil {
		log.Printf("[s3 error] failed to upload document to S3: %v", err)
		status = "s3 uploaded failed"
	} else {
		status = "s3 uploaded"
	}

	// Update path and status columns in database
	_, err = tx.Exec(ctx, "UPDATE documents SET path = $1, status = $2 WHERE id = $3", s3URL, status, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to update document path and status with S3 URL: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	var optMeta *string
	if strings.TrimSpace(optionalMeta) != "" {
		trimmed := strings.TrimSpace(optionalMeta)
		optMeta = &trimmed
	}

	doc := &models.Document{
		ID:               docID,
		Name:             file.Filename,
		ContentHash:      contentHash,
		Path:             s3URL,
		Size:             file.Size,
		MimeType:         mimeType,
		Status:           status,
		RejectionReason:  nil,
		ExtractedContent: nil,
		OptionalMeta:     optMeta,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	// Record initial event snapshot in Cassandra
	RecordDocumentEvent(*doc)

	// Publish to Redis
	pushDataInRedis(doc)

	return doc, nil
}

// RecordDocumentEvent saves a snapshot event to Cassandra document_snapshots table
func RecordDocumentEvent(doc models.Document) {
	if resources.Session == nil {
		return
	}

	docUUID, err := gocql.ParseUUID(doc.ID)
	if err != nil {
		log.Printf("[Cassandra Snapshot] Invalid UUID %q: %v", doc.ID, err)
		return
	}

	snapshotJSON, err := json.Marshal(doc)
	if err != nil {
		log.Printf("[Cassandra Snapshot] Failed to marshal document snapshot: %v", err)
		return
	}

	now := time.Now().UTC()
	query := `INSERT INTO document_snapshots (document_id, status, db_snapshot, timestamp) VALUES (?, ?, ?, ?)`
	if err := resources.Session.Query(query, docUUID, doc.Status, string(snapshotJSON), now).Exec(); err != nil {
		log.Printf("[Cassandra Snapshot] Failed to save snapshot for document %s (%s): %v", doc.ID, doc.Status, err)
	} else {
		log.Printf("[Cassandra Snapshot] Successfully recorded event for document %s (status: %s) at %s",
			doc.ID, doc.Status, now.Format(time.RFC3339))
	}
}

// GetDocumentEvents queries Cassandra for all events and snapshots for a given document ID
func GetDocumentEvents(ctx context.Context, docID string) ([]models.DocumentSnapshot, error) {
	if resources.Session == nil {
		return nil, errors.New("cassandra connection not initialized")
	}

	docUUID, err := gocql.ParseUUID(docID)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID %q: must be a valid UUID", docID)
	}

	query := `SELECT document_id, status, db_snapshot, timestamp FROM document_snapshots WHERE document_id = ?`
	iter := resources.Session.Query(query, docUUID).WithContext(ctx).Iter()

	events := make([]models.DocumentSnapshot, 0)
	var docUID gocql.UUID
	var status, dbSnapshot string
	var ts time.Time

	for iter.Scan(&docUID, &status, &dbSnapshot, &ts) {
		events = append(events, models.DocumentSnapshot{
			DocumentID: docUID.String(),
			Status:     status,
			DbSnapshot: dbSnapshot,
			Timestamp:  ts,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to query cassandra document events: %w", err)
	}

	return events, nil
}

const DocumentEventsChannel = "document_events"

// pushDataInRedis publishes document data to classic Redis Pub/Sub channel
func pushDataInRedis(doc *models.Document) {
	if resources.RDB == nil {
		log.Println("[Redis Publisher] Redis client not initialized")
		return
	}

	docJSON, err := json.Marshal(doc)
	if err != nil {
		log.Printf("[Redis Publisher] Failed to marshal doc: %v", err)
		return
	}

	ctx := context.Background()
	err = resources.RDB.Publish(ctx, DocumentEventsChannel, string(docJSON)).Err()
	if err != nil {
		log.Printf("[Redis Publisher] Failed to publish to channel %s: %v", DocumentEventsChannel, err)
		return
	}

	log.Printf("[Redis Publisher] Document %s published to channel %q", doc.ID, DocumentEventsChannel)
}
