package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratep "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jessevdk/go-flags"
	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/Decentr-net/ariadne"
	"github.com/Decentr-net/logrus/sentry"

	"github.com/Decentr-net/cerberus/internal/consumer"
	"github.com/Decentr-net/cerberus/internal/consumer/blockchain"
	"github.com/Decentr-net/cerberus/internal/health"
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/internal/storage/postgres"
	"github.com/Decentr-net/cerberus/internal/storage/s3"
)

var opts = struct {
	S3Endpoint        string `long:"s3.endpoint" env:"S3_ENDPOINT" default:"localhost:9000" description:"s3 endpoint'"`
	S3Region          string `long:"s3.region" env:"S3_REGION" default:"" description:"s3 region"`
	S3AccessKeyID     string `long:"s3.access-key-id" env:"S3_ACCESS_KEY_ID" description:"access key id for S3 storage'"`
	S3SecretAccessKey string `long:"s3.secret-access-key" env:"S3_SECRET_ACCESS_KEY" description:"secret access key for S3 storage'"`
	S3UseSSL          bool   `long:"s3.use-ssl" env:"S3_USE_SSL" description:"use ssl for S3 storage connection'"`
	S3Bucket          string `long:"s3.bucket" env:"S3_BUCKET" default:"cerberus" description:"S3 bucket for Cerberus files'"`

	Postgres                   string `long:"postgres" env:"POSTGRES" default:"host=localhost port=5432 user=postgres password=root sslmode=disable" description:"postgres dsn"`
	PostgresMaxOpenConnections int    `long:"postgres.max_open_connections" env:"POSTGRES_MAX_OPEN_CONNECTIONS" default:"0" description:"postgres maximal open connections count, 0 means unlimited"`
	PostgresMaxIdleConnections int    `long:"postgres.max_idle_connections" env:"POSTGRES_MAX_IDLE_CONNECTIONS" default:"5" description:"postgres maximal idle connections count"`
	PostgresMigrations         string `long:"postgres.migrations" env:"POSTGRES_MIGRATIONS" default:"migrations/postgres" description:"postgres migrations directory"`

	BlockchainNode                   string        `long:"blockchain.node" env:"BLOCKCHAIN_NODE" default:"zeus.testnet.decentr.xyz:9090" description:"decentr node grpc address"`
	BlockchainTimeout                time.Duration `long:"blockchain.timeout" env:"BLOCKCHAIN_TIMEOUT" default:"5s" description:"timeout for requests to blockchain node"`
	BlockchainRetryInterval          time.Duration `long:"blockchain.retry_interval" env:"BLOCKCHAIN_RETRY_INTERVAL" default:"2s" description:"interval to be waited on error before retry"`
	BlockchainLastBlockRetryInterval time.Duration `long:"blockchain.last_block_retry_interval" env:"BLOCKCHAIN_LAST_BLOCK_RETRY_INTERVAL" default:"1s" description:"duration to be waited when new block isn't produced before retry"`

	SentryDSN string `long:"sentry.dsn" env:"SENTRY_DSN" description:"sentry dsn"`
	LogLevel  string `long:"log.level" env:"LOG_LEVEL" default:"info" description:"Log level" choice:"debug" choice:"info" choice:"warning" choice:"error"`
}{}

var errTerminated = errors.New("terminated")

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	parser.ShortDescription = "Cerberus"
	parser.LongDescription = "Cerberus"

	_, err := parser.Parse()

	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			parser.WriteHelp(os.Stdout)
			os.Exit(0)
		}
		logrus.WithError(err).Warn("error occurred while parsing flags")
	}

	lvl, _ := logrus.ParseLevel(opts.LogLevel) // err will always be nil
	logrus.SetLevel(lvl)

	logrus.Info("service started")
	logrus.Infof("%+v", opts)

	if opts.SentryDSN != "" {
		hook, err := sentry.NewHook(sentry.Options{
			Dsn:              opts.SentryDSN,
			AttachStacktrace: true,
			Release:          health.GetVersion(),
			ServerName:       "sync",
		}, logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel)

		if err != nil {
			logrus.WithError(err).Fatal("failed to init sentry")
		}

		logrus.AddHook(hook)
	} else {
		logrus.Info("empty sentry dsn")
		logrus.Warn("skip sentry initialization")
	}

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

	db := mustGetDB()

	ctx, cancel := context.WithCancel(context.Background())

	gr, _ := errgroup.WithContext(context.Background())
	gr.Go(func() error {
		return mustGetConsumer(fs, postgres.New(db)).Run(ctx)
	})

	gr.Go(func() error {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		s := <-sigs

		logrus.Infof("terminating by %s signal", s)

		cancel()

		return errTerminated
	})

	logrus.Info("service started")

	if err := gr.Wait(); err != nil && !errors.Is(err, errTerminated) && !errors.Is(err, http.ErrServerClosed) {
		logrus.WithError(err).Fatal("service unexpectedly closed")
	}
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

func mustGetConsumer(fs storage.FileStorage, is storage.IndexStorage) consumer.Consumer {
	fetcher, err := ariadne.New(context.Background(), opts.BlockchainNode, opts.BlockchainTimeout)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create blocks fetcher")
	}

	return blockchain.New(fetcher, fs, is, opts.BlockchainRetryInterval, opts.BlockchainLastBlockRetryInterval)
}
