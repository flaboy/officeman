package storage

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/github-flaboy/officeman/internal/api"
)

type fakeS3API struct {
	getObjectOutput  *s3.GetObjectOutput
	headObjectOutput *s3.HeadObjectOutput
	headObjectErr    error
	putBucket        string
	putKey           string
	putContentType   string
	putBody          []byte
}

func (f *fakeS3API) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return f.getObjectOutput, nil
}

func (f *fakeS3API) PutObject(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.putBucket = aws.ToString(in.Bucket)
	f.putKey = aws.ToString(in.Key)
	f.putContentType = aws.ToString(in.ContentType)
	body, _ := io.ReadAll(in.Body)
	f.putBody = body
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3API) HeadObject(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if f.headObjectErr != nil {
		return nil, f.headObjectErr
	}
	return f.headObjectOutput, nil
}

type fakeAPIError struct {
	code string
	msg  string
}

func (e fakeAPIError) Error() string     { return e.msg }
func (e fakeAPIError) ErrorCode() string { return e.code }
func (e fakeAPIError) ErrorFault() smithy.ErrorFault {
	return smithy.FaultClient
}

func TestS3Store_GetObjectBytes(t *testing.T) {
	store := NewS3Store(func(api.S3SetConfig) (s3API, error) {
		return &fakeS3API{
			getObjectOutput: &s3.GetObjectOutput{
				Body: io.NopCloser(bytes.NewReader([]byte("hello"))),
			},
		}, nil
	})

	body, err := store.GetObjectBytes(context.Background(), api.S3SetConfig{Bucket: "private"}, "a/b.xlsx")
	if err != nil {
		t.Fatalf("GetObjectBytes: %v", err)
	}
	if got, want := string(body), "hello"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestS3Store_PutObjectBytes(t *testing.T) {
	fake := &fakeS3API{}
	store := NewS3Store(func(api.S3SetConfig) (s3API, error) {
		return fake, nil
	})

	err := store.PutObjectBytes(context.Background(), api.S3SetConfig{Bucket: "private"}, "a/b.xlsx", []byte("hello"), xlsxContentType)
	if err != nil {
		t.Fatalf("PutObjectBytes: %v", err)
	}
	if got, want := fake.putBucket, "private"; got != want {
		t.Fatalf("bucket = %q, want %q", got, want)
	}
	if got, want := fake.putKey, "a/b.xlsx"; got != want {
		t.Fatalf("key = %q, want %q", got, want)
	}
	if got, want := fake.putContentType, xlsxContentType; got != want {
		t.Fatalf("contentType = %q, want %q", got, want)
	}
	if got, want := string(fake.putBody), "hello"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
