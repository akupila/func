package source

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
)

// S3 manages source code in AWS S3.
type S3 struct {
	cli    s3iface.ClientAPI
	bucket string
}

// NewS3 creates a new S3 client.
func NewS3(cfg aws.Config, bucket string) *S3 {
	return &S3{
		cli:    s3.New(cfg),
		bucket: bucket,
	}
}

// Has returns true if the given key exists in S3.
func (s *S3) Has(ctx context.Context, key string) (bool, error) {
	_, err := s.cli.HeadObjectRequest(&s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}).Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "NotFound" {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}

// Upload uploads a new item to S3.
func (s *S3) Upload(ctx context.Context, key string, body io.Reader) error {
	mgr := s3manager.NewUploaderWithClient(s.cli)
	_, err := mgr.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   body,
	})
	if err != nil {
		return err
	}
	return nil
}
