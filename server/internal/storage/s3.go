package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type Bucket struct {
	client *s3.Client
	bucket string
}

func FromEnv(ctx context.Context) (Store, error) {
	endpoint := strings.TrimSpace(os.Getenv("STORAGE_ENDPOINT"))
	if endpoint == "" {
		return nil, nil
	}
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		return nil, errors.New("STORAGE_ENDPOINT must start with http:// or https://")
	}
	access := strings.TrimSpace(os.Getenv("STORAGE_ACCESS_KEY"))
	secret := strings.TrimSpace(os.Getenv("STORAGE_SECRET_KEY"))
	if access == "" || secret == "" {
		return nil, errors.New("STORAGE_ACCESS_KEY and STORAGE_SECRET_KEY are required when STORAGE_ENDPOINT is set")
	}
	region := strings.TrimSpace(os.Getenv("STORAGE_REGION"))
	if region == "" {
		region = "us-east-1"
	}
	bucket := strings.TrimSpace(os.Getenv("STORAGE_BUCKET"))
	if bucket == "" {
		bucket = "bouncer"
	}
	return Open(ctx, endpoint, region, bucket, access, secret)
}

func Open(ctx context.Context, endpoint, region, bucket, accessKey, secretKey string) (*Bucket, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
	opened := &Bucket{client: client, bucket: bucket}
	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil && !bucketExists(err) {
		return nil, err
	}
	return opened, nil
}

func (b *Bucket) Get(ctx context.Context, key string) (Object, error) {
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if missing(err) {
			return Object{}, ErrNotFound
		}
		return Object{}, err
	}
	defer func() { _ = out.Body.Close() }()
	body, err := io.ReadAll(out.Body)
	if err != nil {
		return Object{}, err
	}
	revision := ""
	if out.ETag != nil {
		revision = *out.ETag
	}
	return Object{Body: body, Revision: revision}, nil
}

func (b *Bucket) Put(ctx context.Context, key string, body []byte, revision string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(b.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	}
	if revision == "" {
		input.IfNoneMatch = aws.String("*")
	} else {
		input.IfMatch = aws.String(revision)
	}
	out, err := b.client.PutObject(ctx, input)
	if err != nil {
		if conflict(err) {
			return "", ErrConflict
		}
		return "", err
	}
	if out.ETag == nil || *out.ETag == "" {
		return "", fmt.Errorf("storage save for %s returned no revision", key)
	}
	return *out.ETag, nil
}

func (b *Bucket) Delete(ctx context.Context, key string) error {
	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if missing(err) {
		return nil
	}
	return err
}

func (b *Bucket) DeletePrefix(ctx context.Context, prefix string) error {
	var token *string
	for {
		page, err := b.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(b.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: token,
		})
		if err != nil {
			return err
		}
		ids := make([]types.ObjectIdentifier, 0, len(page.Contents))
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			ids = append(ids, types.ObjectIdentifier{Key: obj.Key})
		}
		if len(ids) > 0 {
			deleted, err := b.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(b.bucket),
				Delete: &types.Delete{Objects: ids, Quiet: aws.Bool(true)},
			})
			if err != nil {
				return err
			}
			if len(deleted.Errors) > 0 {
				item := deleted.Errors[0]
				detail := "delete failed"
				if item.Message != nil && *item.Message != "" {
					detail = *item.Message
				}
				return fmt.Errorf("storage cleanup: %s", detail)
			}
		}
		if page.IsTruncated == nil || !*page.IsTruncated {
			return nil
		}
		token = page.NextContinuationToken
	}
}

func bucketExists(err error) bool {
	switch apiCode(err) {
	case "BucketAlreadyExists", "BucketAlreadyOwnedByYou":
		return true
	default:
		return false
	}
}

func missing(err error) bool {
	if err == nil {
		return false
	}
	switch apiCode(err) {
	case "NoSuchKey", "NotFound":
		return true
	}
	return httpStatus(err) == 404
}

func conflict(err error) bool {
	switch httpStatus(err) {
	case 409, 412:
		return true
	}
	switch apiCode(err) {
	case "PreconditionFailed", "ConditionalRequestConflict", "Conflict":
		return true
	default:
		return false
	}
}

func apiCode(err error) string {
	var api smithy.APIError
	if errors.As(err, &api) {
		return api.ErrorCode()
	}
	return ""
}

func httpStatus(err error) int {
	var resp *smithyhttp.ResponseError
	if errors.As(err, &resp) {
		return resp.HTTPStatusCode()
	}
	return 0
}
