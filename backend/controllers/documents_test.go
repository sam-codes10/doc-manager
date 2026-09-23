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

func TestAcceptDocumentAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Initialize config and DB
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
	}

	// Clean up created test document directory
	if dataMap, ok := res.Data.(map[string]interface{}); ok {
		if docID, ok := dataMap["id"].(string); ok && docID != "" {
			_ = os.RemoveAll(filepath.Join("uploads", "s3", "documents", docID))
		}
	}
}
