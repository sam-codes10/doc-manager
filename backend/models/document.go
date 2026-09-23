package models

import "time"

type Document struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Path             string    `json:"path"`
	Size             int64     `json:"size"`
	MimeType         string    `json:"mime_type"`
	Status           string    `json:"status"`
	RejectionReason  *string   `json:"rejection_reason,omitempty"`
	ExtractedContent *string   `json:"extracted_content,omitempty"`
	OptionalMeta     *string   `json:"optional_meta,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type DocumentSnapshot struct {
	DocumentID string    `json:"document_id"`
	Status     string    `json:"status"`
	DbSnapshot string    `json:"db_snapshot"`
	Timestamp  time.Time `json:"timestamp"`
}
