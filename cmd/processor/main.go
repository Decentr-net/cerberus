package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jessevdk/go-flags"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/Decentr-net/logrus/sentry"

	"github.com/Decentr-net/cerberus/internal/health"
	"github.com/Decentr-net/cerberus/internal/storage/postgres"
)

var opts = struct {
	Host      string `long:"http.host" env:"HTTP_HOST" default:"localhost" description:"IP to listen on"`
	Port      int    `long:"http.port" env:"HTTP_PORT" default:"8080" description:"port to listen on for insecure connections, defaults to a random value"`
	SentryDSN string `long:"sentry.dsn" env:"SENTRY_DSN" description:"sentry dsn"`
	LogLevel  string `long:"log.level" env:"LOG_LEVEL" default:"info" description:"Log level" choice:"debug" choice:"info" choice:"warning" choice:"error"`

	S3opts
	SQSOpts
	DBOpts
	BlockchainOpts
}{}

var errTerminated = errors.New("terminated")

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	parser.ShortDescription = "Processor"
	parser.LongDescription = "Processor"

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
	fs := mustGetFileStorage()
	is := postgres.New(db)
	b := mustGetBroadcaster()
	c := mustGetConsumer(fs, is, b)

	r := chi.NewMux()
	health.SetupRouter(r, fs, health.PingFunc(b.PingContext), health.PingFunc(db.PingContext))
	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		Handler: r,
	}

	gr, ctx := errgroup.WithContext(context.Background())
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

	gr.Go(func() error {
		if err := c.Run(ctx); err != nil {
			if errors.Is(ctx.Err(), errTerminated) {
				return err
			}

			logrus.WithError(err).Fatal("consumer unexpectedly stopped")
		}

		return nil
	})

	logrus.Info("service started")

	if err := gr.Wait(); err != nil && !errors.Is(err, errTerminated) && !errors.Is(err, http.ErrServerClosed) {
		logrus.WithError(err).Fatal("service unexpectedly closed")
	}
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
