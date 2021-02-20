package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	clicontext "github.com/cosmos/cosmos-sdk/client/context"
	cliflags "github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"gopkg.in/yaml.v2"

	"github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
	"github.com/go-chi/chi"
	"github.com/jessevdk/go-flags"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/Decentr-net/decentr/app"
	"github.com/Decentr-net/logrus/sentry"

	"github.com/Decentr-net/cerberus/internal/blockchain"
	"github.com/Decentr-net/cerberus/internal/crypto/sio"
	"github.com/Decentr-net/cerberus/internal/health"
	"github.com/Decentr-net/cerberus/internal/server"
	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/internal/storage/s3"
)

// nolint:lll,gochecknoglobals,maligned
var opts = struct {
	Host        string `long:"http.host" env:"HTTP_HOST" default:"localhost" description:"IP to listen on"`
	Port        int    `long:"http.port" env:"HTTP_PORT" default:"8080" description:"port to listen on for insecure connections, defaults to a random value"`
	MaxBodySize int64  `long:"http.max-body-size" env:"HTTP_MAX_BODY_SIZE" default:"8000000" description:"max request's body size"`

	SentryDSN string `long:"sentry.dsn" env:"SENTRY_DSN" description:"sentry dsn"`

	S3Endpoint        string `long:"s3.endpoint" env:"S3_ENDPOINT" default:"localhost:9000" description:"s3 endpoint'"`
	S3Region          string `long:"s3.region" env:"S3_REGION" default:"" description:"s3 region"`
	S3AccessKeyID     string `long:"s3.access-key-id" env:"S3_ACCESS_KEY_ID" description:"access key id for S3 storage'"`
	S3SecretAccessKey string `long:"s3.secret-access-key" env:"S3_SECRET_ACCESS_KEY" description:"secret access key for S3 storage'"`
	S3UseSSL          bool   `long:"s3.use-ssl" env:"S3_USE_SSL" description:"use ssl for S3 storage connection'"`
	S3Bucket          string `long:"s3.bucket" env:"S3_BUCKET" default:"cerberus" description:"S3 bucket for Cerberus files'"`

	BlockchainNode               string `long:"blockchain.node" env:"BLOCKCHAIN_NODE" default:"http://zeus.testnet.decentr.xyz:26657" description:"decentr node address"`
	BlockchainFrom               string `long:"blockchain.from" env:"BLOCKCHAIN_FROM" description:"decentr account name to send stakes" required:"true"`
	BlockchainTxMemo             string `long:"blockchain.tx_memo" env:"BLOCKCHAIN_TX_MEMO" description:"decentr tx's memo'"`
	BlockchainChainID            string `long:"blockchain.chain_id" env:"BLOCKCHAIN_CHAIN_ID" default:"testnet" description:"decentr chain id"`
	BlockchainClientHome         string `long:"blockchain.client_home" env:"BLOCKCHAIN_CLIENT_HOME" default:"~/.decentrcli" description:"decentrcli home directory"`
	BlockchainKeyringBackend     string `long:"blockchain.keyring_backend" env:"BLOCKCHAIN_KEYRING_BACKEND" default:"test" description:"decentrcli keyring backend"`
	BlockchainKeyringPromptInput string `long:"blockchain.keyring_prompt_input" env:"BLOCKCHAIN_KEYRING_PROMPT_INPUT" description:"decentrcli keyring prompt input"`

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

	storage, err := s3.NewStorage(s3client, opts.S3Bucket)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create storage")
	}

	bchain := mustGetBlockchain()

	server.SetupRouter(newServiceOrDie(storage, bchain), r,
		opts.MaxBodySize, opts.MinPDVCount, opts.MaxPDVCount)
	health.SetupRouter(r, storage)

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

func newServiceOrDie(s storage.Storage, bchain blockchain.Blockchain) service.Service {
	rewardMap := make(service.RewardMap)
	b, err := ioutil.ReadFile(opts.RewardMapConfig)
	if err != nil {
		logrus.WithError(err).Fatal("failed to read reward map config")
	}
	if err := yaml.Unmarshal(b, rewardMap); err != nil {
		logrus.WithError(err).Fatal("failed to unmarshal reward map config")
	}
	return service.New(sio.NewCrypto(mustExtractEncryptKey()), s, bchain, rewardMap)
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

func mustGetBlockchain() blockchain.Blockchain {
	cdc := app.MakeCodec()

	kb, err := keys.NewKeyring(sdk.KeyringServiceName(),
		opts.BlockchainKeyringBackend,
		opts.BlockchainClientHome,
		bytes.NewBufferString(opts.BlockchainKeyringPromptInput),
	)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create keyring")
	}

	acc, err := kb.Get(opts.BlockchainFrom)
	if err != nil {
		logrus.WithError(err).Fatal("failed to get blockchain account info")
	}

	cliCtx := clicontext.NewCLIContext().
		WithCodec(cdc).
		WithBroadcastMode(cliflags.BroadcastSync).
		WithNodeURI(opts.BlockchainNode).
		WithFrom(acc.GetName()).
		WithFromName(acc.GetName()).
		WithFromAddress(acc.GetAddress()).
		WithChainID(opts.BlockchainChainID)
	cliCtx.Keybase = kb

	txBldr := auth.NewTxBuilder(utils.GetTxEncoder(cdc), 0, 0, 0, 1.0, false,
		opts.BlockchainChainID, opts.BlockchainTxMemo, nil, nil).WithKeybase(kb)

	return blockchain.NewBlockchain(cliCtx, txBldr)
}
