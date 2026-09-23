package resources

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestExtractS3Key(t *testing.T) {
	S3Cfg = S3Config{
		Bucket: "my-test-bucket",
		Region: "us-east-1",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Canonical AWS HTTPS URL",
			input:    "https://my-test-bucket.s3.us-east-1.amazonaws.com/documents/doc-123/file.pdf",
			expected: "documents/doc-123/file.pdf",
		},
		{
			name:     "Path-style URL with bucket in path",
			input:    "https://s3.amazonaws.com/my-test-bucket/documents/doc-123/file.pdf",
			expected: "documents/doc-123/file.pdf",
		},
		{
			name:     "s3:// URI",
			input:    "s3://my-test-bucket/documents/doc-123/file.pdf",
			expected: "documents/doc-123/file.pdf",
		},
		{
			name:     "Plain key",
			input:    "documents/doc-123/file.pdf",
			expected: "documents/doc-123/file.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractS3Key(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractS3Key(%q) = %q; expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestS3UploadAndDownload(t *testing.T) {
	ctx := context.Background()

	// Initialize S3 config (uses mock fallback if no live bucket)
	S3Cfg = S3Config{
		Bucket: "test-doc-bucket",
		Region: "us-east-1",
	}

	testPayload := []byte("hello world s3 content for document test")
	key := "documents/test-uuid-456/invoice.txt"

	url, err := UploadToS3(ctx, key, bytes.NewReader(testPayload), "text/plain")
	if err != nil {
		t.Fatalf("UploadToS3 failed: %v", err)
	}

	if !strings.Contains(url, "test-doc-bucket") || !strings.Contains(url, key) {
		t.Errorf("unexpected s3 url: %s", url)
	}

	// Download using full S3 URL
	downloaded, err := DownloadFromS3(ctx, url)
	if err != nil {
		t.Fatalf("DownloadFromS3 using URL failed: %v", err)
	}

	if string(downloaded) != string(testPayload) {
		t.Errorf("downloaded content mismatch: got %q, expected %q", string(downloaded), string(testPayload))
	}

	// Download using key directly
	downloadedKey, err := DownloadFromS3(ctx, key)
	if err != nil {
		t.Fatalf("DownloadFromS3 using key failed: %v", err)
	}

	if string(downloadedKey) != string(testPayload) {
		t.Errorf("downloaded content by key mismatch: got %q, expected %q", string(downloadedKey), string(testPayload))
	}
}
