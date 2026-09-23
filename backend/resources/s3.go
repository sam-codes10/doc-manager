package resources

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var S3Client *s3.Client

// ConnectS3 initializes the S3 client using the provided S3Config
func ConnectS3(ctx context.Context, cfg S3Config) (*s3.Client, error) {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		if cfg.UsePathStyle {
			o.UsePathStyle = true
		}
	})

	S3Client = client
	fmt.Println("s3 connected successfully")
	return client, nil
}

// UploadToS3 uploads data to S3 and returns the public/canonical S3 URL.
// Includes transparent local fallback storage when running with mock/offline credentials.
func UploadToS3(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("failed to read upload payload: %w", err)
	}

	bucket := S3Cfg.Bucket
	if bucket == "" {
		bucket = "doc-manager-bucket"
	}

	var s3URL string
	if S3Cfg.Endpoint != "" {
		s3URL = fmt.Sprintf("%s/%s/%s", strings.TrimRight(S3Cfg.Endpoint, "/"), bucket, key)
	} else {
		region := S3Cfg.Region
		if region == "" {
			region = "us-east-1"
		}
		s3URL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, key)
	}

	// Always persist to local mock storage as well so local downloads always succeed
	for _, baseDir := range []string{"uploads", "../uploads"} {
		mockStoragePath := filepath.Join(baseDir, "s3", filepath.FromSlash(key))
		if err := os.MkdirAll(filepath.Dir(mockStoragePath), 0755); err == nil {
			_ = os.WriteFile(mockStoragePath, data, 0644)
		}
	}

	// If S3Client is available, attempt PutObject
	if S3Client != nil {
		_, err := S3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        bytes.NewReader(data),
			ContentType: aws.String(contentType),
		})
		if err != nil {
			log.Printf("[S3 Warning] S3 PutObject failed (%v), saved to local mock storage", err)
		} else {
			log.Printf("[S3] Successfully uploaded %s to s3://%s/%s", key, bucket, key)
		}
	}

	return s3URL, nil
}

// DownloadFromS3 downloads an object using its S3 URL or object key.
func DownloadFromS3(ctx context.Context, s3URLOrKey string) ([]byte, error) {
	key := ExtractS3Key(s3URLOrKey)
	bucket := S3Cfg.Bucket
	if bucket == "" {
		bucket = "doc-manager-bucket"
	}

	// 1. Attempt download via AWS S3 client
	if S3Client != nil {
		out, err := S3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err == nil {
			defer out.Body.Close()
			return io.ReadAll(out.Body)
		}
	}

	// 2. Check local mock storage fallback across candidate paths
	candidates := []string{
		filepath.Join("uploads", "s3", filepath.FromSlash(key)),
		filepath.Join("..", "uploads", "s3", filepath.FromSlash(key)),
		filepath.Join("uploads", filepath.Base(key)),
		filepath.Join("..", "uploads", filepath.Base(key)),
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("file %q not found in S3 or local mock storage", key)
}

// ExtractS3Key extracts the object key from a full S3 URL or returns the key as is.
func ExtractS3Key(s3URLOrKey string) string {
	if strings.HasPrefix(s3URLOrKey, "s3://") {
		trimmed := strings.TrimPrefix(s3URLOrKey, "s3://")
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		return trimmed
	}

	if strings.HasPrefix(s3URLOrKey, "http://") || strings.HasPrefix(s3URLOrKey, "https://") {
		u, err := url.Parse(s3URLOrKey)
		if err == nil {
			path := strings.TrimPrefix(u.Path, "/")
			if S3Cfg.Bucket != "" && strings.HasPrefix(path, S3Cfg.Bucket+"/") {
				return strings.TrimPrefix(path, S3Cfg.Bucket+"/")
			}
			return path
		}
	}

	return s3URLOrKey
}
