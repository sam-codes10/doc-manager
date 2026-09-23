package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-manager/apihelpers"
	"doc-manager/resources"

	"github.com/gin-gonic/gin"
)

func setupTestEnvironment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = resources.InitConfig("../resources/secrets.json")
	if resources.DB == nil {
		_, err := resources.Connect(context.Background(), resources.PostgresCfg)
		if err != nil {
			t.Skipf("Skipping integration test: cannot connect to postgres: %v", err)
		}
	}
	if resources.RDB == nil {
		_, _ = resources.InitRedis()
	}
	if resources.Session == nil {
		_, _ = resources.ConnectCassandra(context.Background(), resources.CassandraCfg)
	}
}

func TestAcceptDocumentAPI(t *testing.T) {
	setupTestEnvironment(t)
	_, _ = resources.DB.Exec(context.Background(), "DELETE FROM documents WHERE name = 'sample_test.csv'")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile("file", "sample_test.csv")
	if err != nil {
		t.Fatalf("CreateFormFile error: %v", err)
	}
	_, err = io.WriteString(part, "id,name,amount\n1,Alpha,100\n2,Beta,200\n")
	if err != nil {
		t.Fatalf("WriteString error: %v", err)
	}

	// Add typeOfFile
	_ = writer.WriteField("typeOfFile", "text/csv")
	// Add optionalMeta
	_ = writer.WriteField("optionalMeta", `{"source":"test_upload"}`)
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/documents", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	r := gin.New()
	r.POST("/api/documents", AcceptDocument)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var res apihelpers.ApiRes
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal ApiRes: %v", err)
	}

	if !res.Status {
		t.Errorf("expected Status true, got false. Message: %s", res.Message)
	}

	if res.Data == nil {
		t.Errorf("expected Data not to be nil")
	} else if dataMap, ok := res.Data.(map[string]interface{}); ok {
		pathVal, _ := dataMap["path"].(string)
		if pathVal == "" || !strings.Contains(pathVal, "s3") {
			t.Errorf("expected S3 URL in document path, got %q", pathVal)
		}
		statusVal, _ := dataMap["status"].(string)
		if statusVal != "s3 uploaded" {
			t.Errorf("expected status 's3 uploaded', got %q", statusVal)
		}
		hashVal, _ := dataMap["content_hash"].(string)
		if hashVal == "" {
			t.Errorf("expected non-empty content_hash")
		}
	}

	// Clean up created test document directory & record
	if dataMap, ok := res.Data.(map[string]interface{}); ok {
		if docID, ok := dataMap["id"].(string); ok && docID != "" {
			_ = os.RemoveAll(filepath.Join("uploads", "s3", "documents", docID))
			_, _ = resources.DB.Exec(context.Background(), "DELETE FROM documents WHERE id = $1", docID)
		}
	}
}

func TestDuplicateDocumentUpload(t *testing.T) {
	setupTestEnvironment(t)
	_, _ = resources.DB.Exec(context.Background(), "DELETE FROM documents WHERE name = 'duplicate_check.txt'")

	r := gin.New()
	r.POST("/api/documents", AcceptDocument)

	createUploadReq := func() *http.Request {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "duplicate_check.txt")
		_, _ = io.WriteString(part, "exact duplicate file content bytes for test")
		_ = writer.WriteField("typeOfFile", "text/plain")
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/documents", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return req
	}

	// First upload should succeed (200 OK)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, createUploadReq())
	if w1.Code != http.StatusOK {
		t.Fatalf("first upload expected 200 OK, got %d: %s", w1.Code, w1.Body.String())
	}

	var res1 apihelpers.ApiRes
	_ = json.Unmarshal(w1.Body.Bytes(), &res1)
	var docID string
	if dataMap, ok := res1.Data.(map[string]interface{}); ok {
		docID, _ = dataMap["id"].(string)
	}

	// Second upload with identical filename and content must fail with 409 Conflict
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, createUploadReq())
	if w2.Code != http.StatusConflict {
		t.Fatalf("second upload expected 409 Conflict, got %d: %s", w2.Code, w2.Body.String())
	}

	var res2 apihelpers.ApiRes
	_ = json.Unmarshal(w2.Body.Bytes(), &res2)
	if res2.Status {
		t.Errorf("expected status false for duplicate upload")
	}
	if !strings.Contains(res2.Message, "duplicate") {
		t.Errorf("expected duplicate message, got: %s", res2.Message)
	}

	// Clean up
	if docID != "" {
		_ = os.RemoveAll(filepath.Join("uploads", "s3", "documents", docID))
		_, _ = resources.DB.Exec(context.Background(), "DELETE FROM documents WHERE id = $1", docID)
	}
}

func TestGetDocumentEventsAPI(t *testing.T) {
	setupTestEnvironment(t)
	_, _ = resources.DB.Exec(context.Background(), "DELETE FROM documents WHERE name = 'events_test.txt'")

	r := gin.New()
	r.POST("/api/documents", AcceptDocument)
	r.GET("/api/documents/:id/events", GetDocumentEvents)

	// 1. Upload a document to generate initial Cassandra event
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "events_test.txt")
	_, _ = io.WriteString(part, "events test document content")
	_ = writer.WriteField("typeOfFile", "text/plain")
	writer.Close()

	uploadReq, _ := http.NewRequest(http.MethodPost, "/api/documents", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	wUpload := httptest.NewRecorder()
	r.ServeHTTP(wUpload, uploadReq)

	if wUpload.Code != http.StatusOK {
		t.Fatalf("upload failed: %s", wUpload.Body.String())
	}

	var resUpload apihelpers.ApiRes
	_ = json.Unmarshal(wUpload.Body.Bytes(), &resUpload)
	dataMap, _ := resUpload.Data.(map[string]interface{})
	docID, _ := dataMap["id"].(string)

	// 2. Query events endpoint for this document
	wEvents := httptest.NewRecorder()
	reqEvents, _ := http.NewRequest(http.MethodGet, "/api/documents/"+docID+"/events", nil)
	r.ServeHTTP(wEvents, reqEvents)

	if wEvents.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for events API, got %d: %s", wEvents.Code, wEvents.Body.String())
	}

	var resEvents apihelpers.ApiRes
	if err := json.Unmarshal(wEvents.Body.Bytes(), &resEvents); err != nil {
		t.Fatalf("failed to unmarshal events API response: %v", err)
	}

	if !resEvents.Status {
		t.Errorf("expected Status true for events API")
	}

	eventsList, ok := resEvents.Data.([]interface{})
	if !ok {
		t.Fatalf("expected Data to be an array of snapshots, got: %T", resEvents.Data)
	}

	if len(eventsList) == 0 {
		t.Errorf("expected at least 1 event in Cassandra for uploaded document")
	}

	// 3. Test invalid UUID format
	wInvalid := httptest.NewRecorder()
	reqInvalid, _ := http.NewRequest(http.MethodGet, "/api/documents/not-a-valid-uuid/events", nil)
	r.ServeHTTP(wInvalid, reqInvalid)

	if wInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for invalid UUID, got %d", wInvalid.Code)
	}

	// Clean up
	if docID != "" {
		_ = os.RemoveAll(filepath.Join("uploads", "s3", "documents", docID))
		_, _ = resources.DB.Exec(context.Background(), "DELETE FROM documents WHERE id = $1", docID)
		if resources.Session != nil {
			_ = resources.Session.Query("DELETE FROM document_snapshots WHERE document_id = ?", docID).Exec()
		}
	}
}
