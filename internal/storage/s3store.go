package storage

import (
	"bytes"
	"context"
	"errors"
	"io"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/github-flaboy/officeman/internal/api"
)

const XLSXContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type ObjectStore interface {
	GetObjectBytes(ctx context.Context, cfg api.S3SetConfig, key string) ([]byte, error)
	PutObjectBytes(ctx context.Context, cfg api.S3SetConfig, key string, body []byte, contentType string) error
	HeadObject(ctx context.Context, cfg api.S3SetConfig, key string) (bool, error)
}

type s3API interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

type ClientFactory func(cfg api.S3SetConfig) (s3API, error)

type S3Store struct {
	factory ClientFactory
}

func NewS3Store(factory ClientFactory) *S3Store {
	if factory == nil {
		factory = defaultClientFactory
	}
	return &S3Store{factory: factory}
}

func (s *S3Store) GetObjectBytes(ctx context.Context, cfg api.S3SetConfig, key string) ([]byte, error) {
	client, err := s.factory(cfg)
	if err != nil {
		return nil, err
	}
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &cfg.Bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (s *S3Store) PutObjectBytes(ctx context.Context, cfg api.S3SetConfig, key string, body []byte, contentType string) error {
	client, err := s.factory(cfg)
	if err != nil {
		return err
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &cfg.Bucket,
		Key:         &key,
		Body:        bytes.NewReader(body),
		ContentType: &contentType,
	})
	return err
}

func (s *S3Store) HeadObject(ctx context.Context, cfg api.S3SetConfig, key string) (bool, error) {
	client, err := s.factory(cfg)
	if err != nil {
		return false, err
	}
	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &cfg.Bucket,
		Key:    &key,
	})
	if err == nil {
		return true, nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "404") {
		return false, nil
	}
	return false, err
}

func defaultClientFactory(cfg api.S3SetConfig) (s3API, error) {
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	if cfg.AccessKeyID != "" || cfg.SecretAccessKey != "" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}
	if cfg.Endpoint != "" {
		loadOptions = append(loadOptions, awsconfig.WithBaseEndpoint(cfg.Endpoint))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOptions...)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.ForcePathStyle
	}), nil
}
