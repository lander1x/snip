package shortener

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrLinkNotFound = errors.New("link not found")

type Link struct {
	ID        uuid.UUID
	Code      string
	URL       string
	Clicks    int64
	CreatedAt time.Time
	ExpiresAt *time.Time
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, link *Link) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO links (id, code, url, created_at, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		link.ID, link.Code, link.URL, link.CreatedAt, link.ExpiresAt,
	)
	return err
}

func (r *Repository) GetByCode(ctx context.Context, code string) (*Link, error) {
	link := &Link{}
	err := r.db.QueryRow(ctx,
		`SELECT id, code, url, clicks, created_at, expires_at FROM links WHERE code = $1`,
		code,
	).Scan(&link.ID, &link.Code, &link.URL, &link.Clicks, &link.CreatedAt, &link.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLinkNotFound
	}
	return link, err
}

func (r *Repository) IncrementClicks(ctx context.Context, code string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE links SET clicks = clicks + 1 WHERE code = $1`,
		code,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, code string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM links WHERE code = $1`, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrLinkNotFound
	}
	return nil
}
