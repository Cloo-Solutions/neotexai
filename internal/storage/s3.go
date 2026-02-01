package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3ClientConfig holds configuration for S3Client
type S3ClientConfig struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UsePathStyle    bool
}

// S3Client provides operations for S3-compatible storage (e.g., RustFS)
type S3Client struct {
	client            *s3.Client
	presignClient     *s3.PresignClient
	bucket            string
	uploadURLExpiry   time.Duration
	downloadURLExpiry time.Duration
}

// NewS3Client creates a new S3Client with the given configuration
func NewS3Client(ctx context.Context, cfg S3ClientConfig) (*S3Client, error) {
	// Create custom resolver for S3-compatible endpoints
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if cfg.Endpoint != "" {
				return aws.Endpoint{
					URL:               cfg.Endpoint,
					HostnameImmutable: true,
				}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		},
	)

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with path-style addressing for S3-compatible services
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
	})

	presignClient := s3.NewPresignClient(client)

	return &S3Client{
		client:            client,
		presignClient:     presignClient,
		bucket:            cfg.Bucket,
		uploadURLExpiry:   15 * time.Minute,
		downloadURLExpiry: 1 * time.Hour,
	}, nil
}

// GenerateUploadURL creates a presigned URL for uploading an object
func (c *S3Client) GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	presignedReq, err := c.presignClient.PresignPutObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = c.uploadURLExpiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return presignedReq.URL, nil
}

// GenerateDownloadURL creates a presigned URL for downloading an object
func (c *S3Client) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	presignedReq, err := c.presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = c.downloadURLExpiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return presignedReq.URL, nil
}

// DeleteObject removes an object from storage
func (c *S3Client) DeleteObject(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	_, err := c.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// HeadObject checks if an object exists and returns its metadata
func (c *S3Client) HeadObject(ctx context.Context, key string) (*ObjectMetadata, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	output, err := c.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to head object: %w", err)
	}

	return &ObjectMetadata{
		ContentLength: aws.ToInt64(output.ContentLength),
		ContentType:   aws.ToString(output.ContentType),
		ETag:          aws.ToString(output.ETag),
	}, nil
}

// ObjectMetadata contains metadata about an S3 object
type ObjectMetadata struct {
	ContentLength int64
	ContentType   string
	ETag          string
}

// EnsureBucket creates the bucket if it doesn't exist
func (c *S3Client) EnsureBucket(ctx context.Context) error {
	// Check if bucket exists
	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err == nil {
		return nil // Bucket exists
	}

	// Create bucket
	_, err = c.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}
