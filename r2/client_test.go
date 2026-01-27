package r2

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// mockS3Client is a mock implementation of the s3API interface
type mockS3Client struct {
	getObjectFunc     func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	putObjectFunc     func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	listObjectsV2Func func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	headBucketFunc    func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.getObjectFunc(ctx, params, optFns...)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return m.putObjectFunc(ctx, params, optFns...)
}

func (m *mockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return m.listObjectsV2Func(ctx, params, optFns...)
}

func (m *mockS3Client) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return m.headBucketFunc(ctx, params, optFns...)
}

func TestDownloadImage(t *testing.T) {
	mockData := []byte("fake image data")
	mockKey := "test-image.jpg"
	mockBucket := "test-bucket"

	mockClient := &mockS3Client{
		getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			if *params.Bucket != mockBucket {
				t.Errorf("expected bucket %s, got %s", mockBucket, *params.Bucket)
			}
			if *params.Key != mockKey {
				t.Errorf("expected key %s, got %s", mockKey, *params.Key)
			}
			return &s3.GetObjectOutput{
				Body: io.NopCloser(bytes.NewReader(mockData)),
			}, nil
		},
	}

	client := &r2Client{
		client: mockClient,
		bucket: mockBucket,
	}

	data, err := client.DownloadImage(context.Background(), mockKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(data, mockData) {
		t.Errorf("expected %s, got %s", string(mockData), string(data))
	}
}

func TestUploadImage(t *testing.T) {
	mockData := []byte("new image data")
	mockKey := "upload-test.webp"
	mockBucket := "test-bucket"
	mockContentType := "image/webp"

	mockClient := &mockS3Client{
		putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			if *params.Bucket != mockBucket {
				t.Errorf("expected bucket %s, got %s", mockBucket, *params.Bucket)
			}
			if *params.Key != mockKey {
				t.Errorf("expected key %s, got %s", mockKey, *params.Key)
			}
			if *params.ContentType != mockContentType {
				t.Errorf("expected content type %s, got %s", mockContentType, *params.ContentType)
			}

			buf := new(bytes.Buffer)
			_, _ = io.Copy(buf, params.Body)
			if !bytes.Equal(buf.Bytes(), mockData) {
				t.Errorf("expected %s, got %s", string(mockData), buf.String())
			}
			return &s3.PutObjectOutput{}, nil
		},
	}

	client := &r2Client{
		client: mockClient,
		bucket: mockBucket,
	}

	err := client.UploadImage(context.Background(), mockKey, mockData, mockContentType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListObjects(t *testing.T) {
	mockBucket := "test-bucket"
	now := time.Now()

	objects := []types.Object{
		{Key: aws.String("old.jpg"), LastModified: aws.Time(now.Add(-2 * time.Hour))},
		{Key: aws.String("new.jpg"), LastModified: aws.Time(now.Add(-1 * time.Hour))},
		{Key: aws.String("latest.png"), LastModified: aws.Time(now.Add(-30 * time.Minute))},
	}

	mockClient := &mockS3Client{
		listObjectsV2Func: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			return &s3.ListObjectsV2Output{
				Contents:    objects,
				IsTruncated: aws.Bool(false),
			}, nil
		},
	}

	client := &r2Client{
		client: mockClient,
		bucket: mockBucket,
	}

	// Test case 1: List all (since is Zero)
	keys, err := client.ListObjects(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	// Test case 2: List after since (should only return objects newer than 90 mins ago)
	since := now.Add(-90 * time.Minute)
	keys, err = client.ListObjects(context.Background(), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
	if keys[0] != "new.jpg" || keys[1] != "latest.png" {
		t.Errorf("unexpected keys: %v", keys)
	}
}

func TestTestConnection(t *testing.T) {
	mockBucket := "test-bucket"

	mockClient := &mockS3Client{
		headBucketFunc: func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
			if *params.Bucket != mockBucket {
				t.Errorf("expected bucket %s, got %s", mockBucket, *params.Bucket)
			}
			return &s3.HeadBucketOutput{}, nil
		},
	}

	client := &r2Client{
		client: mockClient,
		bucket: mockBucket,
	}

	err := client.TestConnection(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
