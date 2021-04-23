// Package postgres is implementation of storage interface.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/Decentr-net/cerberus/internal/storage"
)

type pg struct {
	ext sqlx.ExtContext
}

type profileDTO struct {
	Address   string      `db:"address"`
	FirstName string      `db:"first_name"`
	LastName  string      `db:"last_name"`
	Bio       string      `db:"bio"`
	Avatar    string      `db:"avatar"`
	Gender    string      `db:"gender"`
	Birthday  time.Time   `db:"birthday"`
	UpdatedAt pq.NullTime `db:"updated_at"`
	CreatedAt time.Time   `db:"created_at"`
}

// New creates new instance of pg.
func New(db *sql.DB) storage.IndexStorage {
	return pg{
		ext: sqlx.NewDb(db, "postgres"),
	}
}

func (s pg) GetProfile(ctx context.Context, addr string) (*storage.Profile, error) {
	var p profileDTO
	if err := sqlx.GetContext(ctx, s.ext, &p, `
		SELECT
			address, first_name, last_name, bio, avatar, gender, birthday, updated_at, created_at
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
				address, first_name, last_name, bio, avatar, gender, birthday, updated_at, created_at
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
		Bio:       p.Bio,
		Avatar:    p.Avatar,
		Gender:    p.Gender,
		Birthday:  p.Birthday,
	}

	if _, err := sqlx.NamedExecContext(ctx, s.ext,
		`
			INSERT INTO profile(address, first_name, last_name, bio, avatar, gender, birthday)
			VALUES(:address, :first_name, :last_name, :bio, :avatar, :gender, :birthday)
			ON CONFLICT(address) DO UPDATE SET
				first_name=excluded.first_name,
				last_name=excluded.last_name,
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
		Bio:       p.Bio,
		Avatar:    p.Avatar,
		Gender:    p.Gender,
		Birthday:  p.Birthday,
		CreatedAt: p.CreatedAt,
	}

	if p.UpdatedAt.Valid {
		out.UpdatedAt = &p.UpdatedAt.Time
	}

	return &out
}
