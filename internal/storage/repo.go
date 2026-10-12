package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knorVV/github_sync_go/internal/github"
)

type RepoStore struct {
	pool *pgxpool.Pool
}

func NewRepoStore(pool *pgxpool.Pool) *RepoStore {
	return &RepoStore{pool: pool}
}

const upsertQuery = `
	INSERT INTO repos (
		github_id, name, full_name, description, html_url, stars,
		language, owner_login, github_created_at, github_updated_at,
		inserted_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, NOW(), NOW())
	ON CONFLICT (github_id) DO UPDATE SET
		stars = EXCLUDED.stars,
		description = EXCLUDED.description,
		full_name = EXCLUDED.full_name,
		html_url = EXCLUDED.html_url,
		language = EXCLUDED.language,
		owner_login = EXCLUDED.owner_login,
		github_updated_at = EXCLUDED.github_updated_at,
		updated_at = NOW()
`

const selectFields = `
	github_id, name, full_name, description, html_url, stars,
	language, owner_login, github_created_at, github_updated_at
`

func (s *RepoStore) UpsertAll(ctx context.Context, repos []github.Repository) error {
	if len(repos) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, r := range repos {
		batch.Queue(upsertQuery,
			r.GithubID, r.Name, r.FullName, r.Description, r.HTMLURL,
			r.Stars, r.Language, r.OwnerLogin,
			r.GithubCreatedAt, r.GithubUpdatedAt,
		)
	}

	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range repos {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (s *RepoStore) ListAll(ctx context.Context) ([]github.Repository, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT "+selectFields+" FROM repos ORDER BY stars DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRepos(rows)
}

func (s *RepoStore) GetByGithubID(ctx context.Context, id int64) (*github.Repository, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT "+selectFields+" FROM repos WHERE github_id = $1 LIMIT 1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	repos, err := scanRepos(rows)
	if err != nil || len(repos) == 0 {
		return nil, err
	}
	return &repos[0], nil
}

func (s *RepoStore) GetByName(ctx context.Context, name string) (*github.Repository, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT "+selectFields+" FROM repos WHERE name = $1 OR full_name = $1 ORDER BY stars DESC LIMIT 1", name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	repos, err := scanRepos(rows)
	if err != nil || len(repos) == 0 {
		return nil, err
	}
	return &repos[0], nil
}

func scanRepos(rows pgx.Rows) ([]github.Repository, error) {
	var repos []github.Repository
	for rows.Next() {
		var r github.Repository
		if err := rows.Scan(
			&r.GithubID, &r.Name, &r.FullName, &r.Description, &r.HTMLURL,
			&r.Stars, &r.Language, &r.OwnerLogin,
			&r.GithubCreatedAt, &r.GithubUpdatedAt,
		); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}
