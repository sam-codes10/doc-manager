package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"doc-manager/models"
	"doc-manager/resources"
)

// AcceptDocument saves the uploaded file, inserts a record with DB-generated ID and timestamps, and fires an async worker.
func AcceptDocument(ctx context.Context, file *multipart.FileHeader, typeOfFile string, optionalMeta string) (*models.Document, error) {
	if file == nil {
		return nil, errors.New("file is required")
	}

	if resources.DB == nil {
		return nil, errors.New("database connection not initialized")
	}

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
		INSERT INTO documents (name, size, mimeType, status, rejection_reason, extracted_content, optional_meta)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRow(
		ctx,
		insertQuery,
		file.Filename,
		file.Size,
		mimeType,
		status,
		nil,          // rejection_reason
		nil,          // extracted_content
		optionalMeta, // optional_meta
	).Scan(&docID, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document into database: %w", err)
	}

	safeFileName := filepath.Base(file.Filename)
	s3Key := fmt.Sprintf("documents/%s/%s", docID, safeFileName)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Upload to S3 and extract the URL
	s3URL, err := resources.UploadToS3(ctx, s3Key, src, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload document to S3: %w", err)
	}

	// Update path column in database with the S3 URL
	_, err = tx.Exec(ctx, "UPDATE documents SET path = $1 WHERE id = $2", s3URL, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to update document path with S3 URL: %w", err)
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
		Path:             s3URL, // Mention S3 URL in document path
		Size:             file.Size,
		MimeType:         mimeType,
		Status:           status,
		RejectionReason:  nil,
		ExtractedContent: nil,
		OptionalMeta:     optMeta,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	// Publish to Redis
	pushDataInRedis(doc)

	return doc, nil
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
