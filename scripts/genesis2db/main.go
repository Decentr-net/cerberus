package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/Decentr-net/decentr/app"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang-migrate/migrate/v4"
	migratep "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jessevdk/go-flags"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	_ "github.com/Decentr-net/cerberus/internal/consumer/blockchain"
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/internal/storage/postgres"
)

var opts = struct {
	Genesis            string `long:"genesis" env:"GENESIS" default:"genesis.json" description:"path to genesis"`
	Postgres           string `long:"postgres" env:"POSTGRES" default:"host=localhost port=5432 user=postgres password=root sslmode=disable" description:"postgres dsn"`
	PostgresMigrations string `long:"postgres.migrations" env:"POSTGRES_MIGRATIONS" default:"scripts/migrations/postgres" description:"postgres migrations directory"`
}{}

type genesis struct {
	AppState struct {
		Profile ProfileGenesisState `json:"profile"`
	} `json:"app_state"`
}

type ProfileGenesisState struct {
	ProfileRecords []Profile `json:"profiles"`
}

// Profile represent an account settings storage
type Profile struct {
	// Owner is Profile owner
	Owner sdk.AccAddress `json:"owner"`
	// Public profile data
	Public Public `json:"public"`
}

// Public profile data
type Public struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Avatar    string `json:"avatar"`
	Bio       string `json:"bio"`
	Gender    string `json:"gender"`
	Birthday  string `json:"birthday"`
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	parser.ShortDescription = "genesis2db"
	parser.LongDescription = "Genesis to database importer"

	_, err := parser.Parse()

	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			parser.WriteHelp(os.Stdout)
			os.Exit(0)
		}
		logrus.WithError(err).Fatal("error occurred while parsing flags")
	}

	logrus.Info("db2migration started")
	logrus.Infof("%+v", opts)

	b, err := ioutil.ReadFile(opts.Genesis)
	if err != nil {
		logrus.WithError(err).Fatal("failed to read genesis")
	}

	var g genesis

	cdc := app.MakeCodec()
	cdc.MustUnmarshalJSON(b, &g)

	db := mustGetDB()
	s := postgres.New(db)

	logrus.Info("import profiles")

	for i, v := range g.AppState.Profile.ProfileRecords {
		bd, err := time.Parse(time.RFC3339, v.Public.Birthday)
		if err != nil {
			logrus.WithError(err).Fatal("failed to put parse brithday")
		}

		if err := s.SetProfile(context.Background(), &storage.SetProfileParams{
			Address:   v.Owner.String(),
			FirstName: v.Public.FirstName,
			LastName:  v.Public.LastName,
			Bio:       v.Public.Bio,
			Avatar:    v.Public.Avatar,
			Gender:    v.Public.Gender,
			Birthday:  bd,
		}); err != nil {
			logrus.WithError(err).Fatal("failed to put profile into db")
		}

		if i%20 == 0 {
			logrus.Infof("%d of %d profiles imported", i+1, len(g.AppState.Profile.ProfileRecords))
		}
	}

	logrus.Info("done")
}

func mustGetDB() *sql.DB {
	db, err := sql.Open("postgres", opts.Postgres)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create postgres connection")
	}

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
