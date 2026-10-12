CREATE TABLE IF NOT EXISTS repos (
    id            BIGSERIAL PRIMARY KEY,
    github_id     BIGINT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    full_name     TEXT,
    description   TEXT,
    html_url      TEXT,
    stars         INTEGER DEFAULT 0,
    language      TEXT,
    owner_login   TEXT,
    github_created_at TIMESTAMPTZ,
    github_updated_at TIMESTAMPTZ,
    inserted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_repos_stars ON repos (stars DESC);
CREATE INDEX IF NOT EXISTS idx_repos_name ON repos (name);
CREATE INDEX IF NOT EXISTS idx_repos_full_name ON repos (full_name);