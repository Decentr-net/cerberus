package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migratep "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jessevdk/go-flags"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"

	"github.com/Decentr-net/cerberus/internal/entities"
	"github.com/Decentr-net/cerberus/internal/storage/postgres"
)

type DBOpts struct {
	Postgres                   string `long:"postgres" env:"POSTGRES" default:"host=localhost port=5432 user=postgres password=root sslmode=disable" description:"postgres dsn"`
	PostgresMaxOpenConnections int    `long:"postgres.max_open_connections" env:"POSTGRES_MAX_OPEN_CONNECTIONS" default:"0" description:"postgres maximal open connections count, 0 means unlimited"`
	PostgresMaxIdleConnections int    `long:"postgres.max_idle_connections" env:"POSTGRES_MAX_IDLE_CONNECTIONS" default:"5" description:"postgres maximal idle connections count"`
	PostgresMigrations         string `long:"postgres.migrations" env:"POSTGRES_MIGRATIONS" default:"migrations/postgres" description:"postgres migrations directory"`
}

type S3opts struct {
	S3Endpoint        string `long:"s3.endpoint" env:"S3_ENDPOINT" default:"localhost:9000" description:"s3 endpoint"`
	S3Region          string `long:"s3.region" env:"S3_REGION" default:"" description:"s3 region"`
	S3AccessKeyID     string `long:"s3.access-key-id" env:"S3_ACCESS_KEY_ID" description:"access key id for S3 storage"`
	S3SecretAccessKey string `long:"s3.secret-access-key" env:"S3_SECRET_ACCESS_KEY" description:"secret access key for S3 storage"`
	S3UseSSL          bool   `long:"s3.use-ssl" env:"S3_USE_SSL" description:"use ssl for S3 storage connection"`
	S3Bucket          string `long:"s3.bucket" env:"S3_BUCKET" default:"cerberus" description:"S3 bucket for Cerberus files"`
}

var opts = struct {
	S3opts
	DBOpts
}{}

func main() {
	parser := flags.NewParser(&opts, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		logrus.WithError(err).Fatal(err)
	}

	is := postgres.New(mustGetDB())

	s3client, err := minio.New(opts.S3Endpoint, &minio.Options{
		Region: opts.S3Region,
		Creds:  credentials.NewStaticV4(opts.S3AccessKeyID, opts.S3SecretAccessKey, ""),
		Secure: opts.S3UseSSL,
	})
	if err != nil {
		logrus.WithError(err).Fatal("failed to connect to S3 storage")
	}

	iterateMeta(context.Background(), s3client, opts.S3Bucket, func(owner string, id uint64, meta *entities.PDVMeta) error {
		if err := is.SetPDVMeta(context.Background(), owner, id, "", meta); err != nil {
			return fmt.Errorf("failed to save meta: %w", err)
		}

		return nil
	})

	return
}

func parseFilepath(f string) (res struct {
	Address string
	Type    string
	ID      uint64
}, err error) {
	s := strings.Split(f, "/")
	if len(s) != 3 {
		err = errors.New("not a cerberus file")
		return
	}

	res.Address = s[0]
	res.Type = s[1]

	res.ID, err = getIDFromFilename(s[2])

	return
}

func iterateMeta(ctx context.Context, c *minio.Client, bucket string, f func(string, uint64, *entities.PDVMeta) error) {
	ch := c.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Recursive: true,
	})

	for v := range ch {
		log := logrus.WithField("filepath", v.Key)
		if v.Err != nil {
			log.WithError(v.Err).Error("failed to get object key")
			continue
		}

		d, err := parseFilepath(v.Key)
		if err != nil {
			log.WithError(err).Error("failed to parse filepath")
		}

		if d.Type != "meta" {
			log.Info("not a meta")
			continue
		}

		obj, err := c.GetObject(ctx, bucket, v.Key, minio.GetObjectOptions{})
		if err != nil {
			log.WithError(err).Error("failed to get object")
			continue
		}

		var m entities.PDVMeta
		if err := json.NewDecoder(obj).Decode(&m); err != nil {
			log.WithError(err).Error("failed to decode meta")
			continue
		}

		if err := f(d.Address, d.ID, &m); err != nil {
			log.WithError(err).Error("failed to save meta")
			continue
		}

		if err := c.RemoveObject(ctx, bucket, v.Key, minio.RemoveObjectOptions{}); err != nil {
			log.WithError(err).Error("failed to remove meta")
		}
	}
}

func getIDFromFilename(s string) (uint64, error) {
	v, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0, err
	}
	return math.MaxUint64 - v, nil
}

func mustGetDB() *sql.DB {
	db, err := sql.Open("postgres", opts.Postgres)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create postgres connection")
	}
	db.SetMaxOpenConns(opts.PostgresMaxOpenConnections)
	db.SetMaxIdleConns(opts.PostgresMaxIdleConnections)

	if err := db.PingContext(context.Background()); err != nil {
		logrus.WithError(err).Fatal("failed to ping postgres")
	}

	driver, err := migratep.WithInstance(db, &migratep.Config{})
	if err != nil {
		logrus.WithError(err).Fatal("failed to create database migrate driver")
	}

	migrator, err := migrate.NewWithDatabaseInstance(fmt.Sprintf("file://%s", opts.PostgresMigrations), "postgres", driver)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create migrator")
	}

	switch v, d, err := migrator.Version(); err {
	case nil:
		logrus.Infof("database version %d with dirty state %t", v, d)
	case migrate.ErrNilVersion:
		logrus.Info("database version: nil")
	default:
		logrus.WithError(err).Fatal("failed to get version")
	}

	switch err := migrator.Up(); err {
	case nil:
		logrus.Info("database was migrated")
	case migrate.ErrNoChange:
		logrus.Info("database is up-to-date")
	default:
		logrus.WithError(err).Fatal("failed to migrate db")
	}

	return db
}
