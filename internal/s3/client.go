package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// Client wraps AWS S3 client for pre-signed URL generation
type Client struct {
	s3Client       *s3.Client
	presignClient  *s3.PresignClient
	bucket         string
}

// NewClient creates a new S3 client (MinIO compatible)
func NewClient(endpoint, bucket, accessKey, secretKey, region string) (*Client, error) {
	// Create custom credentials
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	// Load AWS config with custom endpoint (MinIO)
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Create S3 client with custom endpoint resolver
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = true // Required for MinIO
	})

	presignClient := s3.NewPresignClient(s3Client)

	return &Client{
		s3Client:      s3Client,
		presignClient: presignClient,
		bucket:        bucket,
	}, nil
}

// GeneratePresignedUploadURL generates a pre-signed PUT URL for uploading an artifact
func (c *Client) GeneratePresignedUploadURL(ctx context.Context, artifactID uuid.UUID, mimeType string, duration time.Duration) (string, error) {
	key := fmt.Sprintf("artifacts/%s", artifactID.String())

	// Create PutObject request
	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(mimeType),
	}

	// Generate pre-signed URL
	presignedURL, err := c.presignClient.PresignPutObject(ctx, putObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = duration
	})
	if err != nil {
		return "", fmt.Errorf("presign put object: %w", err)
	}

	return presignedURL.URL, nil
}

// GeneratePresignedDownloadURL generates a pre-signed GET URL for downloading an artifact
func (c *Client) GeneratePresignedDownloadURL(ctx context.Context, artifactID uuid.UUID, duration time.Duration) (string, error) {
	key := fmt.Sprintf("artifacts/%s", artifactID.String())

	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	presignedURL, err := c.presignClient.PresignGetObject(ctx, getObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = duration
	})
	if err != nil {
		return "", fmt.Errorf("presign get object: %w", err)
	}

	return presignedURL.URL, nil
}

// VerifyUpload verifies that an artifact was uploaded successfully (checks existence and size)
func (c *Client) VerifyUpload(ctx context.Context, artifactID uuid.UUID, expectedBytes int64) error {
	key := fmt.Sprintf("artifacts/%s", artifactID.String())

	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	resp, err := c.s3Client.HeadObject(ctx, headObjectInput)
	if err != nil {
		return fmt.Errorf("head object: %w", err)
	}

	if *resp.ContentLength != expectedBytes {
		return fmt.Errorf("size mismatch: expected %d, got %d", expectedBytes, *resp.ContentLength)
	}

	return nil
}
