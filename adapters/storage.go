package adapters

import (
	"bytes"
	"context"
	"io"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pshima/consul-snapshot/interfaces"
)

// S3Adapter implements StorageClient for AWS S3
type S3Adapter struct {
	session    *session.Session
	region     string
	endpoint   string
	encryption string
	kmsKeyID   string
}

// NewS3Adapter creates a new S3 adapter
func NewS3Adapter(region, endpoint, encryption, kmsKeyID string) interfaces.StorageClient {
	awsConfig := &aws.Config{Region: aws.String(region)}
	
	// If endpoint is provided, use it for S3-compatible services
	if endpoint != "" {
		awsConfig.Endpoint = aws.String(endpoint)
		awsConfig.S3ForcePathStyle = aws.Bool(true) // Required for MinIO and most S3-compatible services
	}
	
	sess := session.New(awsConfig)
	return &S3Adapter{
		session:    sess,
		region:     region,
		endpoint:   endpoint,
		encryption: encryption,
		kmsKeyID:   kmsKeyID,
	}
}

// Upload uploads data to S3
func (s *S3Adapter) Upload(bucket, key string, data []byte) error {
	uploader := s3manager.NewUploader(s.session)
	
	params := &s3manager.UploadInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   bytes.NewReader(data),
	}
	
	// Add server-side encryption if configured
	if s.encryption != "" {
		params.ServerSideEncryption = &s.encryption
		if s.encryption == "aws:kms" && s.kmsKeyID != "" {
			params.SSEKMSKeyId = &s.kmsKeyID
		}
	}
	
	_, err := uploader.Upload(params)
	return err
}

// Download downloads data from S3
func (s *S3Adapter) Download(bucket, key string) ([]byte, error) {
	downloader := s3manager.NewDownloader(s.session)
	
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err := downloader.Download(buf, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	
	return buf.Bytes(), err
}

// GCSAdapter implements StorageClient for Google Cloud Storage
type GCSAdapter struct {
	client *storage.Client
}

// NewGCSAdapter creates a new GCS adapter
func NewGCSAdapter() (interfaces.StorageClient, error) {
	client, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, err
	}
	return &GCSAdapter{client: client}, nil
}

// Upload uploads data to GCS
func (g *GCSAdapter) Upload(bucket, key string, data []byte) error {
	ctx := context.Background()
	obj := g.client.Bucket(bucket).Object(key)
	w := obj.NewWriter(ctx)
	
	_, err := w.Write(data)
	if err != nil {
		w.Close()
		return err
	}
	
	return w.Close()
}

// Download downloads data from GCS
func (g *GCSAdapter) Download(bucket, key string) ([]byte, error) {
	ctx := context.Background()
	obj := g.client.Bucket(bucket).Object(key)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	
	return io.ReadAll(r)
}