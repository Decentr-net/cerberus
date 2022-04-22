package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

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

	var total uint64
	for f := range s3client.ListObjects(context.Background(), opts.S3Bucket, minio.ListObjectsOptions{
		Recursive: true,
	}) {
		if p, _ := parseFilepath(f.Key); p.Type == "meta" {
			total++
		}
	}
	l := logrus.WithField("total", total)

	var processed uint
	l.Info("start processing")
	iterateMeta(context.Background(), s3client, opts.S3Bucket, func(owner string, id uint64, meta *entities.PDVMeta) error {
		if err := is.SetPDVMeta(context.Background(), owner, id, "", "", meta); err != nil {
			return fmt.Errorf("failed to save meta: %w", err)
		}

		processed++
		l.WithField("processed", processed).Infof("%s/%d moved to db", owner, id)

		return nil
	})
	l.Info("done")

	return
}

func parseFilepath(f string) (res struct {
	Address string
	Type    string
	ID      uint64
}, err error) {
	s := strings.Split(f, "/")
	if len(s) != 3 {
		res.Type = "unknown"
		return
	}

	res.Address = s[0]
	res.Type = s[1]

	if res.Type != "meta" && res.Type != "pdv" {
		res.Type = "unknown"
		return
	}

	res.ID, err = getIDFromFilename(s[2])

	return
}

func iterateMeta(ctx context.Context, c *minio.Client, bucket string, f func(string, uint64, *entities.PDVMeta) error) {
	ch := c.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Recursive: true,
	})

	wg := sync.WaitGroup{}
	for i := 0; i < 16; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
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
		}()
	}

	wg.Wait()
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

	return db
}
