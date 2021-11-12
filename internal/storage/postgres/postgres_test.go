//+build integration

package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	m "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Decentr-net/cerberus/internal/entities"
	"github.com/Decentr-net/cerberus/internal/schema"
	"github.com/Decentr-net/cerberus/internal/storage"
)

var (
	db  *sqlx.DB
	ctx = context.Background()
	s   storage.IndexStorage
)

func TestMain(m *testing.M) {
	shutdown := setup()

	s = New(db.DB)

	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() func() {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:12",
		Env:          map[string]string{"POSTGRES_PASSWORD": "root"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
	})
	if err != nil {
		logrus.WithError(err).Fatalf("failed to create container")
	}

	if err := c.Start(ctx); err != nil {
		logrus.WithError(err).Fatal("failed to start container")
	}

	host, err := c.Host(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("failed to get host")
	}

	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		logrus.WithError(err).Fatal("failed to map port")
	}

	dsn := fmt.Sprintf("host=%s port=%d user=postgres password=root sslmode=disable", host, port.Int())

	db, err = sqlx.Open("postgres", dsn)
	if err != nil {
		logrus.WithError(err).Fatal("failed to open connection")
	}

	if err := db.Ping(); err != nil {
		logrus.WithError(err).Fatal("failed to ping postgres")
	}

	shutdownFn := func() {
		if c != nil {
			c.Terminate(ctx)
		}
	}

	migrate("postgres", "root", host, "postgres", port.Int())

	return shutdownFn
}

func migrate(username, password, hostname, dbname string, port int) {
	_, currFile, _, ok := runtime.Caller(0)
	if !ok {
		logrus.Fatal("failed to get current file location")
	}

	migrations := filepath.Join(currFile, "..", "..", "..", "..", "scripts", "migrations", "postgres")

	migrator, err := m.New(
		fmt.Sprintf("file://%s", migrations),
		fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			username, password, hostname, port, dbname),
	)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create migrator")
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		logrus.WithError(err).Fatal("failed to migrate")
	}
}

func cleanup() {
	db.MustExecContext(ctx, `DELETE FROM profile`)
	db.MustExecContext(ctx, `DELETE FROM pdv`)
}

func TestPg_GetHeight(t *testing.T) {
	t.Cleanup(cleanup)

	h, err := s.GetHeight(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 0, h)
}

func TestPg_SetHeight(t *testing.T) {
	t.Cleanup(cleanup)

	require.NoError(t, s.SetHeight(ctx, 10))

	h, err := s.GetHeight(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 10, h)
}

func TestPg_InTx(t *testing.T) {
	t.Cleanup(cleanup)

	require.NoError(t, s.InTx(context.Background(), func(tx storage.IndexStorage) error {
		require.NoError(t, tx.SetHeight(ctx, 1))
		return nil
	}))

	h, err := s.GetHeight(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 1, h)
}

func TestPg_SetProfile(t *testing.T) {
	t.Cleanup(cleanup)

	compare := func(expected storage.SetProfileParams, p *storage.Profile) {
		assert.Equal(t, expected.Address, p.Address)
		assert.Equal(t, expected.FirstName, p.FirstName)
		assert.Equal(t, expected.LastName, p.LastName)
		assert.Equal(t, expected.Bio, p.Bio)
		assert.Equal(t, expected.Avatar, p.Avatar)
		assert.Equal(t, expected.Gender, p.Gender)
		assert.Equal(t, expected.Birthday.UTC(), p.Birthday.UTC())
	}

	expected := storage.SetProfileParams{
		Address:   "address",
		FirstName: "first_name",
		LastName:  "last_name",
		Emails:    []string{"email1", "email2"},
		Bio:       "bio",
		Avatar:    "avatar",
		Gender:    "male",
		Birthday:  date("2009-01-02"),
	}
	require.NoError(t, s.SetProfile(ctx, &expected))
	p, err := s.GetProfile(ctx, expected.Address)
	require.NoError(t, err)
	require.NotNil(t, p)
	compare(expected, p)
	assert.Nil(t, p.UpdatedAt)
	assert.False(t, p.CreatedAt.IsZero())

	expected = storage.SetProfileParams{
		Address:   "address",
		FirstName: "first_name2",
		LastName:  "last_name2",
		Emails:    []string{"email2"},
		Bio:       "bio2",
		Avatar:    "avatar2",
		Gender:    "male2",
		Birthday:  date("2008-01-02"),
	}
	require.NoError(t, s.SetProfile(ctx, &expected))
	p, err = s.GetProfile(ctx, expected.Address)
	require.NoError(t, err)
	require.NotNil(t, p)
	compare(expected, p)
	assert.NotNil(t, p.UpdatedAt)
	assert.False(t, p.CreatedAt.IsZero())

	_, err = s.GetProfile(ctx, "wrong")
	require.ErrorIs(t, err, storage.ErrNotFound)
}

func TestPg_GetProfiles(t *testing.T) {
	t.Cleanup(cleanup)

	p := storage.SetProfileParams{
		Address:   "address_1",
		FirstName: "first_name",
		LastName:  "last_name",
		Emails:    []string{"email1", "email2"},
		Bio:       "bio",
		Avatar:    "avatar",
		Gender:    "male",
		Birthday:  date("2009-01-02"),
	}

	require.NoError(t, s.SetProfile(ctx, &p))

	p.Address = "address_2"
	require.NoError(t, s.SetProfile(ctx, &p))

	p.Address = "address_3"
	require.NoError(t, s.SetProfile(ctx, &p))

	pp, err := s.GetProfiles(ctx, []string{"address_1", "address_2", "address_4"})
	require.NoError(t, err)
	require.Len(t, pp, 2)

	for i, v := range pp {
		assert.Equal(t, fmt.Sprintf("address_%d", i+1), v.Address)
		assert.Equal(t, p.FirstName, v.FirstName)
		assert.Equal(t, p.LastName, v.LastName)
		assert.Equal(t, p.Emails, v.Emails)
		assert.Equal(t, p.Bio, v.Bio)
		assert.Equal(t, p.Avatar, v.Avatar)
		assert.Equal(t, p.Gender, v.Gender)
		assert.Equal(t, p.Birthday.UTC(), v.Birthday.UTC())
	}
}

func TestPg_DeleteProfile(t *testing.T) {
	t.Cleanup(cleanup)

	p := storage.SetProfileParams{
		Address:   "address",
		FirstName: "first_name",
		LastName:  "last_name",
		Bio:       "bio",
		Avatar:    "avatar",
		Gender:    "male",
		Birthday:  date("2009-01-02"),
	}

	require.NoError(t, s.SetProfile(ctx, &p))

	_, err := s.GetProfile(ctx, p.Address)
	require.NoError(t, err)

	require.NoError(t, s.DeleteProfile(ctx, p.Address))
	_, err = s.GetProfile(ctx, p.Address)
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func TestPg_SetPDVMeta(t *testing.T) {
	t.Cleanup(cleanup)

	exp := &entities.PDVMeta{
		ObjectTypes: map[schema.Type]uint16{
			"cookie": 1,
		},
		Reward: sdk.NewDecWithPrec(1, 6),
	}

	require.NoError(t, s.SetPDVMeta(ctx, "1", 1, "tx", exp))

	var tx string
	require.NoError(t, db.GetContext(ctx, &tx, `SELECT tx FROM pdv WHERE id = 1`))
	require.Equal(t, "tx", tx)

	act, err := s.GetPDVMeta(ctx, "1", 1)
	require.NoError(t, err)
	require.Equal(t, exp, act)
}

func TestPg_GetPDVMeta(t *testing.T) {
	t.Cleanup(cleanup)

	act, err := s.GetPDVMeta(ctx, "1", 1)
	require.ErrorIs(t, err, storage.ErrNotFound)
	require.Nil(t, act)
}

func TestPg_ListPDV(t *testing.T) {
	t.Cleanup(cleanup)

	for i := 1; i <= 10; i++ {
		require.NoError(t, s.SetPDVMeta(ctx, "1", uint64(i), "tx", &entities.PDVMeta{
			ObjectTypes: map[schema.Type]uint16{
				"cookie": 1,
			},
			Reward: sdk.NewDecWithPrec(1, 6),
		}))
	}

	ids, err := s.ListPDV(ctx, "1", 0, 3)
	require.NoError(t, err)
	require.Equal(t, []uint64{10, 9, 8}, ids)

	ids, err = s.ListPDV(ctx, "1", 3, 3)
	require.NoError(t, err)
	require.Equal(t, []uint64{2, 1}, ids)

	ids, err = s.ListPDV(ctx, "2", 3, 3)
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestPg_DeletePDV(t *testing.T) {
	t.Cleanup(cleanup)

	for i := 1; i <= 10; i++ {
		require.NoError(t, s.SetPDVMeta(ctx, "1", uint64(i), "tx", &entities.PDVMeta{
			ObjectTypes: map[schema.Type]uint16{
				"cookie": 1,
			},
			Reward: sdk.NewDecWithPrec(1, 6),
		}))
	}

	require.NoError(t, s.DeletePDV(ctx, "1"))

	ids, err := s.ListPDV(ctx, "1", 0, 10)
	require.NoError(t, err)
	require.Empty(t, ids)
}

func date(d string) time.Time {
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		panic(err)
	}
	return t
}
