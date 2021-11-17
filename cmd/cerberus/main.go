package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jessevdk/go-flags"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"

	"github.com/Decentr-net/logrus/sentry"

	"github.com/Decentr-net/cerberus/internal/crypto/sio"
	"github.com/Decentr-net/cerberus/internal/health"
	"github.com/Decentr-net/cerberus/internal/producer"
	"github.com/Decentr-net/cerberus/internal/server"
	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/internal/storage/postgres"
	"github.com/Decentr-net/cerberus/internal/throttler"
)

var opts = struct {
	Host           string        `long:"http.host" env:"HTTP_HOST" default:"localhost" description:"IP to listen on"`
	Port           int           `long:"http.port" env:"HTTP_PORT" default:"8080" description:"port to listen on for insecure connections, defaults to a random value"`
	RequestTimeout time.Duration `long:"http.request-timeout" env:"HTTP_REQUEST_TIMEOUT" default:"45s" description:"request processing timeout"`
	MaxBodySize    int64         `long:"http.max-body-size" env:"HTTP_MAX_BODY_SIZE" default:"8000000" description:"max request's body size"`

	SentryDSN string `long:"sentry.dsn" env:"SENTRY_DSN" description:"sentry dsn"`
	LogLevel  string `long:"log.level" env:"LOG_LEVEL" default:"info" description:"Log level" choice:"debug" choice:"info" choice:"warning" choice:"error"`

	SavePDVThrottlePeriod time.Duration `long:"save-pdv-throttle-period" env:"SAVE_PDV_THROTTLE_PERIOD" default:"10m" description:"how often the user can send PDV to save"`
	RewardMapConfig       string        `long:"reward-map-config" env:"REWARD_MAP_CONFIG" default:"configs/rewards.yml" description:"path to yaml config with pdv rewards"`
	MinPDVCount           uint16        `long:"min-pdv-count" env:"MIN_PDV_COUNT" default:"100" description:"minimal count of pdv to save"`
	MaxPDVCount           uint16        `long:"max-pdv-count" env:"MAX_PDV_COUNT" default:"100" description:"maximal count of pdv to save"`
	EncryptKey            string        `long:"encrypt-key" env:"ENCRYPT_KEY" description:"encrypt key in hex which will be used for encrypting and decrypting user's data"`

	S3opts
	SQSOpts
	DBOpts
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

	db := mustGetDB()
	is := postgres.New(db)
	fs := mustGetFileStorage()

	server.SetupRouter(newServiceOrDie(fs, is, mustGetProducer()), r,
		opts.RequestTimeout, opts.MaxBodySize, throttler.New(opts.SavePDVThrottlePeriod), opts.MinPDVCount, opts.MaxPDVCount)
	health.SetupRouter(r, fs, health.PingFunc(db.PingContext))

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

func newServiceOrDie(fs storage.FileStorage, is storage.IndexStorage, p producer.Producer) service.Service {
	rewardMap := make(service.RewardMap)
	b, err := ioutil.ReadFile(opts.RewardMapConfig)
	if err != nil {
		logrus.WithError(err).Fatal("failed to read reward map config")
	}
	if err := yaml.Unmarshal(b, rewardMap); err != nil {
		logrus.WithError(err).Fatal("failed to unmarshal reward map config")
	}
	return service.New(sio.NewCrypto(mustExtractEncryptKey()), fs, is, p, rewardMap)
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
