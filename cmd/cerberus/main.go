package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/Decentr-net/cerberus/internal/crypto/sio"
	"github.com/Decentr-net/cerberus/internal/server"
	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/internal/storage/ipfs"
)

// nolint:lll
var opts = struct {
	Host string `long:"host" env:"HOST" default:"localhost" description:"IP to listen on"`
	Port int    `long:"port" env:"PORT" default:"8080" description:"port to listen on for insecure connections, defaults to a random value"`

	IPFS       string `long:"ipfs" env:"IPFS" default:"localhost:5001" description:"ipfs node's address'"`
	EncryptKey string `long:"encrypt.key" env:"ENCRYPT_KEY" description:"encrypt key in hex which will be used for encrypting and decrypting PDV"`

	LogLevel string `long:"log.level" env:"LOG_LEVEL" default:"info" description:"Log level" choice:"debug" choice:"info" choice:"warning" choice:"error"`

	MaxBodySize int64 `long:"max.body.size" env:"MAX_BODY_SIZE" default:"8000000" description:"Max request's body size'"`
}{}

var errTerminated = errors.New("terminated")

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	parser.ShortDescription = "Cerberus"
	parser.LongDescription = "Cerberus"

	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
	}

	lvl, _ := logrus.ParseLevel(opts.LogLevel) // err will always be nil
	logrus.SetLevel(lvl)

	logrus.Info("service started")
	logrus.Infof("%+v", opts)

	r := chi.NewMux()

	sh := shell.NewShellWithClient(opts.IPFS, &http.Client{Timeout: 5 * time.Second})
	if !sh.IsUp() {
		logrus.Fatal("ipfs node is unavailable")
	}

	// nolint: godox
	// todo: add health checker

	server.SetupRouter(service.New(sio.NewCrypto(mustExtractEncryptKey()), ipfs.NewStorage(sh)), r, opts.MaxBodySize)

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

	if err := gr.Wait(); err != nil && !errors.Is(err, errTerminated) && !errors.Is(err, http.ErrServerClosed) {
		logrus.WithError(err).Fatal("service unexpectedly closed")
	}
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
