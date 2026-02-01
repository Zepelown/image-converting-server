package r2

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	appConfig "image-converting-server/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageClient defines the interface for R2 storage operations
type StorageClient interface {
	DownloadImage(ctx context.Context, key string) ([]byte, error)
	UploadImage(ctx context.Context, key string, data []byte, contentType string) error
	ListObjects(ctx context.Context, since time.Time) ([]string, error)
	TestConnection(ctx context.Context) error
	DeleteObject(ctx context.Context, key string) error
}

// s3API defines the subset of S3 client methods used by r2Client for testability
type s3API interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type r2Client struct {
	client s3API
	bucket string
}

// NewClient creates a new R2 storage client
func NewClient(ctx context.Context, cfg *appConfig.R2Config) (StorageClient, error) {
	// Load AWS configuration with static credentials and custom endpoint
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		awsConfig.WithRegion("auto"), // R2 uses 'auto' region
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with custom endpoint
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
	})

	return &r2Client{
		client: s3Client,
		bucket: cfg.Bucket,
	}, nil
}

// DownloadImage downloads an image from R2
func (r *r2Client) DownloadImage(ctx context.Context, key string) ([]byte, error) {
	output, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download image from R2 (key: %s): %w", key, err)
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data from R2 response (key: %s): %w", key, err)
	}

	return data, nil
}

// UploadImage uploads an image to R2
func (r *r2Client) UploadImage(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload image to R2 (key: %s): %w", key, err)
	}

	return nil
}

// ListObjects lists object keys modified after the given time
func (r *r2Client) ListObjects(ctx context.Context, since time.Time) ([]string, error) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(r.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(r.bucket),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects from R2: %w", err)
		}

		for _, obj := range page.Contents {
			// If since is Zero, include all objects
			// Otherwise only include objects modified after 'since'
			if since.IsZero() || obj.LastModified.After(since) {
				keys = append(keys, *obj.Key)
			}
		}
	}

	return keys, nil
}

// TestConnection verifies the connection to R2 by checking if the bucket exists
func (r *r2Client) TestConnection(ctx context.Context) error {
	_, err := r.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(r.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to R2 bucket %s: %w", r.bucket, err)
	}
	return nil
}

// DeleteObject deletes an object from R2
func (r *r2Client) DeleteObject(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from R2 (key: %s): %w", key, err)
	}
	return nil
}
