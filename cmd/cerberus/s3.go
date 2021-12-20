package main

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"

	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/internal/storage/s3"
)

type S3Opts struct {
	S3Endpoint        string `long:"s3.endpoint" env:"S3_ENDPOINT" default:"localhost:9000" description:"s3 endpoint"`
	S3Region          string `long:"s3.region" env:"S3_REGION" default:"" description:"s3 region"`
	S3AccessKeyID     string `long:"s3.access-key-id" env:"S3_ACCESS_KEY_ID" description:"access key id for S3 storage"`
	S3SecretAccessKey string `long:"s3.secret-access-key" env:"S3_SECRET_ACCESS_KEY" description:"secret access key for S3 storage"`
	S3UseSSL          bool   `long:"s3.use-ssl" env:"S3_USE_SSL" description:"use ssl for S3 storage connection"`
	S3Bucket          string `long:"s3.bucket" env:"S3_BUCKET" default:"cerberus" description:"S3 bucket for Cerberus files"`
}

func mustGetFileStorage() storage.FileStorage {
	s3client, err := minio.New(opts.S3Endpoint, &minio.Options{
		Region: opts.S3Region,
		Creds:  credentials.NewStaticV4(opts.S3AccessKeyID, opts.S3SecretAccessKey, ""),
		Secure: opts.S3UseSSL,
	})
	if err != nil {
		logrus.WithError(err).Fatal("failed to connect to S3 storage")
	}

	fs, err := s3.NewStorage(s3client, opts.S3Bucket)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create storage")
	}

	return fs
}
