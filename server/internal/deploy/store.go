package deploy

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Run struct {
	SHA        string
	Status     string
	Log        string
	StartedAt  time.Time
	FinishedAt *time.Time
}

type history interface {
	Start(ctx context.Context, sha string) (uuid.UUID, error)
	Finish(ctx context.Context, id uuid.UUID, sha, status, logText string) error
	LastPublished(ctx context.Context) (string, error)
	Latest(ctx context.Context) (*Run, error)
	FailRunning(ctx context.Context) error
}

type pgStore struct {
	pool *pgxpool.Pool
}

func (p *pgStore) Start(ctx context.Context, sha string) (uuid.UUID, error) {
	id := uuid.New()
	_, err := p.pool.Exec(ctx, `INSERT INTO deploys (id, sha, status) VALUES ($1, $2, 'running')`, id, sha)
	return id, err
}

func (p *pgStore) Finish(ctx context.Context, id uuid.UUID, sha, status, logText string) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE deploys
		SET sha = $2, status = $3, log = $4, finished_at = now()
		WHERE id = $1`, id, sha, status, logText)
	return err
}

func (p *pgStore) LastPublished(ctx context.Context) (string, error) {
	var sha string
	err := p.pool.QueryRow(ctx, `
		SELECT sha FROM deploys
		WHERE status = 'published' AND sha <> ''
		ORDER BY finished_at DESC
		LIMIT 1`).Scan(&sha)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return sha, err
}

func (p *pgStore) Latest(ctx context.Context) (*Run, error) {
	var run Run
	err := p.pool.QueryRow(ctx, `
		SELECT sha, status, log, started_at, finished_at
		FROM deploys
		ORDER BY started_at DESC
		LIMIT 1`).Scan(&run.SHA, &run.Status, &run.Log, &run.StartedAt, &run.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (p *pgStore) FailRunning(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE deploys
		SET status = 'failed',
		    finished_at = now(),
		    log = CASE WHEN length(log) = 0 THEN 'interrupted' ELSE log || E'\ninterrupted' END
		WHERE status = 'running'`)
	return err
}
