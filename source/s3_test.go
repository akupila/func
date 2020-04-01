package source

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
)

func TestS3_Has(t *testing.T) {
	bucket, key := "buc", "file.zip"
	hook := func(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
		if *input.Bucket == bucket && *input.Key == key {
			return &s3.HeadObjectOutput{}, nil
		}
		return nil, awserr.New("NotFound", "Not found", nil)
	}

	tests := []struct {
		name    string
		hook    HeadObjectHook
		bucket  string
		key     string
		want    bool
		wantErr bool
	}{
		{
			name:   "Exists",
			hook:   hook,
			bucket: bucket,
			key:    key,
			want:   true,
		},
		{
			name:   "NoExist",
			hook:   hook,
			bucket: bucket,
			key:    "otherkey",
			want:   false,
		},
		{
			name: "Error",
			hook: func(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
				return nil, awserr.New("TestError", "Err", nil)
			},
			bucket:  bucket,
			key:     key,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &S3{
				cli: &mockS3{
					HeadObject: tc.hook,
				},
				bucket: tc.bucket,
			}
			got, err := s.Has(context.Background(), tc.key)
			if (err != nil) != tc.wantErr {
				t.Errorf("Error = %v, want err = %t", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("Got = %t, want = %t", got, tc.want)
			}
		})
	}
}

func TestS3_Upload(t *testing.T) {
	bucket, key := "bucket", "file.zip"

	var got []byte
	hook := func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
		if *input.Bucket != bucket || *input.Key != key {
			return nil, fmt.Errorf("wrong bucket/key")
		}
		b, err := ioutil.ReadAll(input.Body)
		if err != nil {
			return nil, err
		}
		got = b
		return &s3.PutObjectOutput{}, nil
	}
	s := &S3{
		cli: &mockS3{
			PutObject: hook,
		},
		bucket: bucket,
	}

	data := []byte("hello")

	err := s.Upload(context.Background(), key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got, data) {
		t.Errorf("Uploaded data does not match\nGot\n%s\nWant\n%s", hex.Dump(got), hex.Dump(data))
	}
}

// ---

type (
	HeadObjectHook func(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error)
	PutObjectHook  func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
)

type mockS3 struct {
	s3iface.ClientAPI

	// Hooks
	HeadObject HeadObjectHook
	PutObject  PutObjectHook
}

func (m *mockS3) req() *aws.Request {
	return &aws.Request{
		HTTPRequest:  &http.Request{URL: &url.URL{}, Header: make(http.Header)},
		HTTPResponse: &http.Response{},
		Retryer:      aws.NoOpRetryer{},
	}
}

func (m *mockS3) HeadObjectRequest(input *s3.HeadObjectInput) s3.HeadObjectRequest {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		r.Data, r.Error = m.HeadObject(input)
	})
	return s3.HeadObjectRequest{Request: req}
}

func (m *mockS3) PutObjectRequest(input *s3.PutObjectInput) s3.PutObjectRequest {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		r.Data, r.Error = m.PutObject(input)
	})
	return s3.PutObjectRequest{Request: req}
}
