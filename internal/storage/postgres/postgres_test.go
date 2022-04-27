//go:build integration
// +build integration

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
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/pkg/schema"
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

func TestPg_SetProfileBanned(t *testing.T) {
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
	require.NoError(t, s.SetProfileBanned(ctx, p.Address))
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
		if expected.Birthday != nil {
			assert.Equal(t, expected.Birthday.UTC(), p.Birthday.UTC())
		} else {
			assert.Nil(t, p.Birthday)
		}
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

	expected = storage.SetProfileParams{
		Address:   "address",
		FirstName: "",
		LastName:  "",
		Emails:    []string{},
		Bio:       "",
		Avatar:    "",
		Gender:    "",
		Birthday:  nil,
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
		assert.Equal(t, false, v.Banned)
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

	require.NoError(t, s.SetPDVMeta(ctx, "1", 1, "tx", "ios", exp))

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

func TestPg_GetSetPDVRewardsDistributedDate(t *testing.T) {
	t.Cleanup(cleanup)

	date := time.Now().UTC()

	require.NoError(t, s.SetPDVRewardsDistributedDate(ctx, date))

	savedDate, err := s.GetPDVRewardsDistributedDate(ctx)
	require.NoError(t, err)

	require.Equal(t, date.Unix(), savedDate.Unix())
}

func TestPg_GetPDVDeltaTotal(t *testing.T) {
	t.Cleanup(cleanup)

	t.Run("zero", func(t *testing.T) {
		total, err := s.GetPDVTotalDelta(ctx)
		require.NoError(t, err)
		require.Zero(t, total)
	})

	t.Run("3 different accounts", func(t *testing.T) {
		require.NoError(t, s.SetPDVMeta(ctx, "address1", 1, "trx1", "ios", &entities.PDVMeta{
			Reward: sdk.NewDecWithPrec(1, 6),
		}))
		require.NoError(t, s.SetPDVMeta(ctx, "address2", 1, "trx2", "desktop", &entities.PDVMeta{
			Reward: sdk.NewDecWithPrec(2, 6),
		}))
		require.NoError(t, s.SetPDVMeta(ctx, "address3", 1, "trx3", "android", &entities.PDVMeta{
			Reward: sdk.NewDecWithPrec(3, 6),
		}))

		total, err := s.GetPDVTotalDelta(ctx)
		require.NoError(t, err)
		require.Equal(t, 6e-06, total)

		// update distribution date
		require.NoError(t, s.SetPDVRewardsDistributedDate(ctx, time.Now().UTC()))

		total, err = s.GetPDVTotalDelta(ctx)
		require.NoError(t, err)
		require.NoError(t, err)
		require.Zero(t, total)
	})
}

func TestPg_GetPDVDelta(t *testing.T) {
	t.Cleanup(cleanup)

	total, err := s.GetPDVDelta(ctx, "")
	require.NoError(t, err)
	require.Zero(t, total)

	t.Run("2 pdvs", func(t *testing.T) {
		const addr = "address"

		require.NoError(t, s.SetPDVMeta(ctx, addr, 1, "trx1", "ios", &entities.PDVMeta{
			Reward: sdk.NewDecWithPrec(1, 6),
		}))
		require.NoError(t, s.SetPDVMeta(ctx, addr, 2, "trx2", "ios", &entities.PDVMeta{
			Reward: sdk.NewDecWithPrec(2, 6),
		}))

		delta, err := s.GetPDVDelta(ctx, addr)
		require.NoError(t, err)
		require.Equal(t, 3e-06, delta)

		// update distribution date
		require.NoError(t, s.SetPDVRewardsDistributedDate(ctx, time.Now().UTC()))

		delta, err = s.GetPDVDelta(ctx, addr)
		require.NoError(t, err)
		require.NoError(t, err)
		require.Zero(t, delta)
	})
}

func TestPg_GetPDVDeltaList(t *testing.T) {
	t.Cleanup(cleanup)

	deltas, err := s.GetPDVDeltaList(ctx)
	require.NoError(t, err)
	require.Len(t, deltas, 0)

	require.NoError(t, s.SetPDVMeta(ctx, "address1", 1, "trx1", "ios", &entities.PDVMeta{
		Reward: sdk.NewDecWithPrec(1, 6),
	}))
	require.NoError(t, s.SetPDVMeta(ctx, "address2", 1, "trx2", "ios", &entities.PDVMeta{
		Reward: sdk.NewDecWithPrec(2, 6),
	}))
	require.NoError(t, s.SetPDVMeta(ctx, "address2", 2, "trx22", "ios", &entities.PDVMeta{
		Reward: sdk.NewDecWithPrec(20, 6),
	}))
	require.NoError(t, s.SetPDVMeta(ctx, "address3", 1, "trx3", "ios", &entities.PDVMeta{
		Reward: sdk.NewDecWithPrec(3, 6),
	}))

	deltas, err = s.GetPDVDeltaList(ctx)
	require.NoError(t, err)
	require.Len(t, deltas, 3)

	require.Equal(t, &storage.PDVDelta{Address: "address1", Delta: 0.000001}, deltas[0])
	require.Equal(t, &storage.PDVDelta{Address: "address2", Delta: 0.000022}, deltas[1])
	require.Equal(t, &storage.PDVDelta{Address: "address3", Delta: 0.000003}, deltas[2])
}

func TestPg_RewardsQueue(t *testing.T) {
	t.Cleanup(cleanup)

	require.NoError(t, s.CreateRewardsQueueItem(ctx, "address1", 5))
	require.NoError(t, s.CreateRewardsQueueItem(ctx, "address2", 10))
	require.NoError(t, s.CreateRewardsQueueItem(ctx, "address3", 15))

	items, err := s.GetRewardsQueueItemList(ctx)
	require.NoError(t, err)
	require.Len(t, items, 3)

	require.NoError(t, s.DeleteRewardsQueueItem(ctx, "address1"))
	require.NoError(t, s.DeleteRewardsQueueItem(ctx, "address2"))
	require.NoError(t, s.DeleteRewardsQueueItem(ctx, "address3"))

	items, err = s.GetRewardsQueueItemList(ctx)
	require.NoError(t, err)
	require.Len(t, items, 0)
}

func TestPg_ListPDV(t *testing.T) {
	t.Cleanup(cleanup)

	for i := 1; i <= 10; i++ {
		require.NoError(t, s.SetPDVMeta(ctx, "1", uint64(i), "tx", "ios", &entities.PDVMeta{
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
		require.NoError(t, s.SetPDVMeta(ctx, "1", uint64(i), "tx", "ios", &entities.PDVMeta{
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

func date(d string) *time.Time {
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		panic(err)
	}
	return &t
}
