// Package postgres is implementation of storage interface.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/Decentr-net/cerberus/internal/entities"
	"github.com/Decentr-net/cerberus/internal/storage"
)

var log = logrus.WithField("layer", "storage").WithField("package", "postgres")
var errBeginCalledWithinTx = errors.New("can not run WithLockedHeight in tx")

var _ storage.IndexStorage = pg{}

type pg struct {
	ext sqlx.ExtContext
}

type profileDTO struct {
	Address   string         `db:"address"`
	FirstName string         `db:"first_name"`
	LastName  string         `db:"last_name"`
	Emails    pq.StringArray `db:"emails"`
	Bio       string         `db:"bio"`
	Avatar    string         `db:"avatar"`
	Gender    string         `db:"gender"`
	Banned    bool           `db:"banned"`
	Birthday  pq.NullTime    `db:"birthday"`
	UpdatedAt pq.NullTime    `db:"updated_at"`
	CreatedAt time.Time      `db:"created_at"`
}

// New creates new instance of pg.
func New(db *sql.DB) *pg { // nolint:golint
	return &pg{
		ext: sqlx.NewDb(db, "postgres"),
	}
}

func (s pg) InTx(ctx context.Context, f func(s storage.IndexStorage) error) error {
	db, ok := s.ext.(*sqlx.DB)
	if !ok {
		return errBeginCalledWithinTx
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to create tx: %w", err)
	}

	if err := func(s storage.IndexStorage) error {
		if err := f(s); err != nil {
			return err
		}

		return nil
	}(pg{ext: tx}); err != nil {
		if err := tx.Rollback(); err != nil {
			log.WithError(err).Error("failed to rollback tx")
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commint tx: %w", err)
	}

	return nil
}

func (s pg) GetHeight(ctx context.Context) (uint64, error) {
	var h uint64
	if err := sqlx.GetContext(ctx, s.ext, &h, `SELECT height FROM height`); err != nil {
		return 0, fmt.Errorf("failed to query: %w", err)
	}

	return h, nil
}

func (s pg) SetHeight(ctx context.Context, height uint64) error {
	if _, err := s.ext.ExecContext(ctx, `UPDATE height SET height = $1`, height); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	return nil
}

func (s pg) GetProfile(ctx context.Context, addr string) (*storage.Profile, error) {
	var p profileDTO
	if err := sqlx.GetContext(ctx, s.ext, &p, `
		SELECT
			address, first_name, last_name, emails, bio, avatar, gender, birthday, banned, updated_at, created_at
		FROM profile
		WHERE address = $1
	`, addr); err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
	}

	return toStorageProfile(&p), nil
}

func (s pg) GetProfiles(ctx context.Context, addr []string) ([]*storage.Profile, error) {
	if len(addr) == 0 {
		return []*storage.Profile{}, nil
	}

	query, args, err := sqlx.In(`
			SELECT
				address, first_name, last_name, emails, bio, avatar, gender, birthday, banned, updated_at, created_at
			FROM profile
			WHERE address IN (?)
			ORDER BY address
		`, stringsUnique(addr))

	if err != nil {
		return nil, fmt.Errorf("failed to construct IN clause: %w", err)
	}

	var pp []*profileDTO

	if err := sqlx.SelectContext(ctx, s.ext, &pp, s.ext.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	out := make([]*storage.Profile, len(pp))
	for i, v := range pp {
		out[i] = toStorageProfile(v)
	}

	return out, nil
}

func (s pg) SetProfile(ctx context.Context, p *storage.SetProfileParams) error {
	profile := profileDTO{
		Address:   p.Address,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		Emails:    p.Emails,
		Bio:       p.Bio,
		Avatar:    p.Avatar,
		Gender:    p.Gender,
	}

	if p.Birthday != nil {
		profile.Birthday = pq.NullTime{
			Time:  *p.Birthday,
			Valid: true,
		}
	}

	if _, err := sqlx.NamedExecContext(ctx, s.ext,
		`
			INSERT INTO profile(address, first_name, last_name, emails, bio, avatar, gender, birthday, banned)
			VALUES(:address, :first_name, :last_name, :emails, :bio, :avatar, :gender, :birthday, FALSE)
			ON CONFLICT(address) DO UPDATE SET
				first_name=excluded.first_name,
				last_name=excluded.last_name,
				emails=excluded.emails,
				bio=excluded.bio,
				avatar=excluded.avatar,
				gender=excluded.gender,
				birthday=excluded.birthday
		`, profile,
	); err != nil {
		return fmt.Errorf("failed to exec: %w", err)
	}

	return nil
}

// DeleteProfile deletes profile.
func (s pg) DeleteProfile(ctx context.Context, addr string) error {
	if _, err := s.ext.ExecContext(ctx, `DELETE FROM profile WHERE address = $1`, addr); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	return nil
}

func (s pg) ListPDV(ctx context.Context, owner string, from uint64, limit uint16) ([]uint64, error) {
	if from == 0 {
		from = math.MaxInt64
	}

	var out []uint64
	if err := sqlx.SelectContext(ctx, s.ext, &out, `
		SELECT id FROM pdv
		WHERE owner = $1 AND id < $2
		ORDER BY id DESC
		LIMIT $3
	`, owner, from, limit); err != nil {
		return nil, fmt.Errorf("failed to select: %w", err)
	}

	if len(out) == 0 {
		out = []uint64{}
	}

	return out, nil
}

func (s pg) GetPDVMeta(ctx context.Context, address string, id uint64) (*entities.PDVMeta, error) {
	var meta json.RawMessage
	if err := sqlx.GetContext(ctx, s.ext, &meta, `
		SELECT meta FROM pdv
		WHERE owner = $1 AND id = $2
	`, address, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get: %w", err)
	}

	var out entities.PDVMeta
	if err := json.Unmarshal(meta, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal meta: %w", err)
	}

	return &out, nil
}

func (s pg) SetPDVMeta(ctx context.Context, address string, id uint64, tx string, m *entities.PDVMeta) error {
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal meta: %w", err)
	}

	reward, _ := m.Reward.Float64()

	if _, err := s.ext.ExecContext(ctx, `
		INSERT INTO pdv(owner, id, tx, meta, reward) VALUES($1, $2, $3, $4, $5) ON CONFLICT (owner, id) DO UPDATE
			SET tx = EXCLUDED.tx, meta = EXCLUDED.meta, reward = EXCLUDED.reward
	`, address, id, tx, b, reward); err != nil {
		return fmt.Errorf("failed to insert: %w", err)
	}

	return nil
}

func (s pg) DeletePDV(ctx context.Context, address string) error {
	if _, err := s.ext.ExecContext(ctx, `DELETE FROM pdv WHERE owner = $1`, address); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

func (s pg) GetPDVDelta(ctx context.Context, address string) (float64, error) {
	var delta float64
	err := sqlx.GetContext(ctx, s.ext, &delta, `
		SELECT COALESCE(SUM(reward), 0) FROM pdv
        WHERE
              owner NOT IN (SELECT address FROM profile WHERE banned) AND
              created_at > (SELECT date FROM pdv_rewards_distributed_date) AND 
              owner = $1
              
    `, address)
	return delta, err
}

func (s pg) GetPDVTotalDelta(ctx context.Context) (float64, error) {
	var total float64
	err := sqlx.GetContext(ctx, s.ext, &total, `
		SELECT COALESCE(SUM(reward), 0) FROM pdv
        WHERE
              owner NOT IN (SELECT address FROM profile WHERE banned) AND
              created_at > (SELECT date FROM pdv_rewards_distributed_date)
    `)
	return total, err
}

func (s pg) GetPDVRewardsDistributedDate(ctx context.Context) (time.Time, error) {
	var t time.Time
	if err := sqlx.GetContext(ctx, s.ext, &t, `SELECT date FROM pdv_rewards_distributed_date`); err != nil {
		return time.Time{}, fmt.Errorf("failed to query: %w", err)
	}

	return t, nil
}

func (s pg) SetPDVRewardsDistributedDate(ctx context.Context, date time.Time) error {
	if _, err := s.ext.ExecContext(ctx, `UPDATE pdv_rewards_distributed_date SET date = $1`, date); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}
	return nil
}

func (s pg) GetPDVDeltaList(ctx context.Context) ([]*storage.PDVDelta, error) {
	var out []*storage.PDVDelta
	if err := sqlx.SelectContext(ctx, s.ext, &out, `
		SELECT owner AS address, SUM(reward) AS delta  FROM pdv
        WHERE
              owner NOT IN (SELECT address FROM profile WHERE banned) AND
              created_at > (SELECT date FROM pdv_rewards_distributed_date) 
        GROUP BY owner
        HAVING SUM(reward) > 0
		ORDER BY owner
	`); err != nil {
		return nil, fmt.Errorf("failed to select: %w", err)
	}

	return out, nil
}

func (s pg) CreateRewardsQueueItem(ctx context.Context, addr string, reward int64) error {
	_, err := s.ext.ExecContext(ctx, `
	INSERT INTO rewards_queue(address, reward) VALUES($1, $2)
	`, addr, reward)
	return err
}

func (s pg) GetRewardsQueueItemList(ctx context.Context) ([]*storage.RewardsQueueItem, error) {
	var out []*storage.RewardsQueueItem

	if err := sqlx.SelectContext(ctx, s.ext, &out, `
		SELECT address, reward FROM rewards_queue ORDER BY created_at
	`); err != nil {
		return nil, fmt.Errorf("failed to select: %w", err)
	}

	return out, nil
}

func (s pg) DeleteRewardsQueueItem(ctx context.Context, addr string) error {
	_, err := s.ext.ExecContext(ctx, `
	DELETE FROM rewards_queue WHERE address = $1
	`, addr)
	return err
}

func stringsUnique(s []string) []string {
	m := make(map[string]struct{}, len(s))
	out := make([]string, 0, len(s))

	for _, v := range s {
		if _, ok := m[v]; !ok {
			m[v] = struct{}{}
			out = append(out, v)
		}
	}

	return out
}

func toStorageProfile(p *profileDTO) *storage.Profile {
	out := storage.Profile{
		Address:   p.Address,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		Emails:    p.Emails,
		Bio:       p.Bio,
		Avatar:    p.Avatar,
		Banned:    p.Banned,
		Gender:    p.Gender,
		CreatedAt: p.CreatedAt,
	}

	if p.Birthday.Valid {
		out.Birthday = &p.Birthday.Time
	}

	if p.UpdatedAt.Valid {
		out.UpdatedAt = &p.UpdatedAt.Time
	}

	return &out
}
