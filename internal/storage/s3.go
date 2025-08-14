package storage

import (
	"context"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	Client *minio.Client
	Bucket string
}

func New(endpoint string, access string, secret string, useSSL bool, bucket string) (*S3, error) {
	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(access, secret, ""),
		Secure: useSSL,
	})
	if err != nil { return nil, err }
	return &S3{Client: c, Bucket: bucket}, nil
}

func (s *S3) PresignPut(ctx context.Context, objectName string, contentType string, expire time.Duration) (string, error) {
	reqParams := make(map[string]string)
	if contentType != "" {
		reqParams["response-content-type"] = contentType
	}
	u, err := s.Client.PresignedPutObject(ctx, s.Bucket, objectName, expire)
	if err != nil { return "", err }
	return u.String(), nil
}
