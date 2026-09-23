package services

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"doc-manager/resources"
)

func TestAcceptDocumentStatus(t *testing.T) {
	_ = resources.InitConfig("../resources/secrets.json")
	if resources.DB == nil {
		_, err := resources.Connect(context.Background(), resources.PostgresCfg)
		if err != nil {
			t.Skipf("Skipping test: cannot connect to postgres: %v", err)
		}
	}
	if resources.RDB == nil {
		_, _ = resources.InitRedis()
	}
	_, _ = resources.DB.Exec(context.Background(), "DELETE FROM documents WHERE name = 'status_test.txt'")

	// Prepare mock file header
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "status_test.txt")
	if err != nil {
		t.Fatalf("CreateFormFile error: %v", err)
	}
	_, _ = part.Write([]byte("status test file content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_ = req.ParseMultipartForm(10 << 20)
	fileHeader := req.MultipartForm.File["file"][0]

	doc, err := AcceptDocument(context.Background(), fileHeader, "text/plain", `{"tag":"status_check"}`)
	if err != nil {
		t.Fatalf("AcceptDocument failed: %v", err)
	}

	if doc.Status != "s3 uploaded" && doc.Status != "s3 uploaded failed" {
		t.Errorf("expected status 's3 uploaded' or 's3 uploaded failed', got %q", doc.Status)
	}

	// Query DB to verify the status column was actually updated in PostgreSQL
	var dbStatus string
	err = resources.DB.QueryRow(context.Background(), "SELECT status FROM documents WHERE id = $1", doc.ID).Scan(&dbStatus)
	if err != nil {
		t.Fatalf("failed to query status from db: %v", err)
	}

	if dbStatus != doc.Status {
		t.Errorf("db status %q does not match doc status %q", dbStatus, doc.Status)
	}

	// Clean up test record
	_, _ = resources.DB.Exec(context.Background(), "DELETE FROM documents WHERE id = $1", doc.ID)
}
