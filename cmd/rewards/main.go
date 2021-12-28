package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jessevdk/go-flags"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/Decentr-net/cerberus/internal/blockchain"
	"github.com/Decentr-net/cerberus/internal/health"
	"github.com/Decentr-net/cerberus/internal/pdvrewards"
	"github.com/Decentr-net/cerberus/internal/storage/postgres"
	"github.com/Decentr-net/logrus/sentry"
)

var opts = struct {
	SentryDSN string `long:"sentry.dsn" env:"SENTRY_DSN" description:"sentry dsn"`
	LogLevel  string `long:"log.level" env:"LOG_LEVEL" default:"info" description:"Log level" choice:"debug" choice:"info" choice:"warning" choice:"error"`

	PDVRewardsPoolSize int64         `long:"pdv-rewards.pool-size" env:"PDV_REWARDS_POOL_SIZE" default:"100000000000" description:"PDV rewards (uDEC)"`
	PDVRewardsInterval time.Duration `long:"pdv-rewards.interval" env:"PDV_REWARDS_INTERVAL" default:"720h" description:"how often to pay PDV rewards"`

	DBOpts
	BlockchainOpts
}{}

var errTerminated = errors.New("terminated")

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	parser.ShortDescription = "Rewards"
	parser.LongDescription = "Rewards"

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

	setupLogger()

	db := mustGetDB()
	b := mustGetBroadcaster()

	gr, ctx := errgroup.WithContext(context.Background())

	gr.Go(func() error {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		s := <-sigs

		logrus.Infof("terminating by %s signal", s)

		return errTerminated
	})

	gr.Go(func() error {
		distributor := pdvrewards.NewDistributor(
			opts.PDVRewardsPoolSize, blockchain.New(b), postgres.New(db))
		distributor.RunAsync(ctx, opts.PDVRewardsInterval)

		return nil
	})

	logrus.Info("service started")
}

func setupLogger() {
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
}
