package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Decentr-net/cerberus/internal/throttler"

	cliflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-chi/chi"
	"github.com/golang-migrate/migrate/v4"
	migratep "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jessevdk/go-flags"
	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"

	"github.com/Decentr-net/decentr/app"
	"github.com/Decentr-net/go-broadcaster"
	"github.com/Decentr-net/logrus/sentry"

	"github.com/Decentr-net/cerberus/internal/blockchain"
	"github.com/Decentr-net/cerberus/internal/crypto/sio"
	"github.com/Decentr-net/cerberus/internal/health"
	"github.com/Decentr-net/cerberus/internal/server"
	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/internal/storage/postgres"
	"github.com/Decentr-net/cerberus/internal/storage/s3"
)

// nolint:lll,gochecknoglobals,maligned
var opts = struct {
	Host           string        `long:"http.host" env:"HTTP_HOST" default:"localhost" description:"IP to listen on"`
	Port           int           `long:"http.port" env:"HTTP_PORT" default:"8080" description:"port to listen on for insecure connections, defaults to a random value"`
	RequestTimeout time.Duration `long:"http.request-timeout" env:"HTTP_REQUEST_TIMEOUT" default:"45s" description:"request processing timeout"`
	MaxBodySize    int64         `long:"http.max-body-size" env:"HTTP_MAX_BODY_SIZE" default:"8000000" description:"max request's body size"`

	SentryDSN string `long:"sentry.dsn" env:"SENTRY_DSN" description:"sentry dsn"`

	SavePDVThrottlePeriod time.Duration `long:"save-pdv-throttle-period" env:"SAVE_PDV_THROTTLE_PERIOD" default:"10m" description:"how often the user can send PDV to save"`

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

	BlockchainNode               string `long:"blockchain.node" env:"BLOCKCHAIN_NODE" default:"http://zeus.testnet.decentr.xyz:26657" description:"decentr node address"`
	BlockchainFrom               string `long:"blockchain.from" env:"BLOCKCHAIN_FROM" description:"decentr account name to send stakes" required:"true"`
	BlockchainTxMemo             string `long:"blockchain.tx_memo" env:"BLOCKCHAIN_TX_MEMO" description:"decentr tx's memo'"`
	BlockchainChainID            string `long:"blockchain.chain_id" env:"BLOCKCHAIN_CHAIN_ID" default:"testnet" description:"decentr chain id"`
	BlockchainClientHome         string `long:"blockchain.client_home" env:"BLOCKCHAIN_CLIENT_HOME" default:"~/.decentrcli" description:"decentrcli home directory"`
	BlockchainKeyringBackend     string `long:"blockchain.keyring_backend" env:"BLOCKCHAIN_KEYRING_BACKEND" default:"test" description:"decentrcli keyring backend"`
	BlockchainKeyringPromptInput string `long:"blockchain.keyring_prompt_input" env:"BLOCKCHAIN_KEYRING_PROMPT_INPUT" description:"decentrcli keyring prompt input"`
	BlockchainGas                uint64 `long:"blockchain.gas" env:"BLOCKCHAIN_GAS" default:"10" description:"gas amount"`
	BlockchainFee                string `long:"blockchain.fee" env:"BLOCKCHAIN_FEE" default:"1udec" description:"transaction fee"`

	RewardMapConfig string `long:"reward-map-config" env:"REWARD_MAP_CONFIG" default:"configs/rewards.yml" description:"path to yaml config with pdv rewards"`
	MinPDVCount     uint16 `long:"min-pdv-count" env:"MIN_PDV_COUNT" default:"100" description:"minimal count of pdv to save"`
	MaxPDVCount     uint16 `long:"max-pdv-count" env:"MAX_PDV_COUNT" default:"100" description:"maximal count of pdv to save"`
	EncryptKey      string `long:"encrypt-key" env:"ENCRYPT_KEY" description:"encrypt key in hex which will be used for encrypting and decrypting user's data"`

	LogLevel string `long:"log.level" env:"LOG_LEVEL" default:"info" description:"Log level" choice:"debug" choice:"info" choice:"warning" choice:"error"`
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
		}, logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel)

		if err != nil {
			logrus.WithError(err).Fatal("failed to init sentry")
		}

		logrus.AddHook(hook)
	} else {
		logrus.Info("empty sentry dsn")
		logrus.Warn("skip sentry initialization")
	}

	r := chi.NewMux()

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
	is := postgres.New(db)

	b := mustGetBroadcaster()

	server.SetupRouter(newServiceOrDie(fs, is, blockchain.New(b)), r,
		opts.RequestTimeout, opts.MaxBodySize, throttler.New(opts.SavePDVThrottlePeriod), opts.MinPDVCount, opts.MaxPDVCount)
	health.SetupRouter(r, fs, health.PingFunc(b.PingContext), health.PingFunc(db.PingContext))

	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		Handler: r,
	}

	gr, _ := errgroup.WithContext(context.Background())
	gr.Go(srv.ListenAndServe)

	gr.Go(func() error {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		s := <-sigs

		logrus.Infof("terminating by %s signal", s)

		if err := srv.Shutdown(context.Background()); err != nil {
			logrus.WithError(err).Error("failed to gracefully shutdown server")
		}

		return errTerminated
	})

	logrus.Info("service started")

	if err := gr.Wait(); err != nil && !errors.Is(err, errTerminated) && !errors.Is(err, http.ErrServerClosed) {
		logrus.WithError(err).Fatal("service unexpectedly closed")
	}
}

func newServiceOrDie(fs storage.FileStorage, is storage.IndexStorage, bchain blockchain.Blockchain) service.Service {
	rewardMap := make(service.RewardMap)
	b, err := ioutil.ReadFile(opts.RewardMapConfig)
	if err != nil {
		logrus.WithError(err).Fatal("failed to read reward map config")
	}
	if err := yaml.Unmarshal(b, rewardMap); err != nil {
		logrus.WithError(err).Fatal("failed to unmarshal reward map config")
	}
	return service.New(sio.NewCrypto(mustExtractEncryptKey()), fs, is, bchain, rewardMap)
}

func mustExtractEncryptKey() [32]byte {
	k, err := hex.DecodeString(opts.EncryptKey)
	if err != nil {
		logrus.WithError(err).Fatal("failed to decode encrypt key")
	}

	if len(k) != 32 {
		logrus.Fatal("encrypt key must be 32 bytes slice")
	}

	r := [32]byte{}
	copy(r[:], k)

	return r
}

func mustGetBroadcaster() *broadcaster.Broadcaster {
	fee, err := sdk.ParseCoin(opts.BlockchainFee)
	if err != nil {
		logrus.WithError(err).Error("failed to parse fee")
	}

	b, err := broadcaster.New(app.MakeCodec(), broadcaster.Config{
		CLIHome:            opts.BlockchainClientHome,
		KeyringBackend:     opts.BlockchainKeyringBackend,
		KeyringPromptInput: opts.BlockchainKeyringPromptInput,
		NodeURI:            opts.BlockchainNode,
		BroadcastMode:      cliflags.BroadcastSync,
		From:               opts.BlockchainFrom,
		ChainID:            opts.BlockchainChainID,
		GenesisKeyPass:     keys.DefaultKeyPass,
		Gas:                opts.BlockchainGas,
		GasAdjust:          1.2,
		Fees:               sdk.Coins{fee},
	})

	if err != nil {
		logrus.WithError(err).Fatal("failed to create broadcaster")
	}

	return b
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
