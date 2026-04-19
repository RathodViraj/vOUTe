CREATE TABLE IF NOT EXISTS vote_snapshots (
    vote_id BIGINT NOT NULL,
    option_id BIGINT NOT NULL,
    vote_count BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

SELECT create_hypertable('vote_snapshots', 'created_at');
CREATE INDEX IF NOT EXISTS idx_vote_snapshots_vote_option_time ON vote_snapshots (vote_id, option_id, created_at DESC);