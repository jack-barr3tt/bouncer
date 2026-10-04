ALTER TABLE deploys ADD COLUMN remote text NOT NULL DEFAULT '';

CREATE INDEX deploys_remote_started_at ON deploys (remote, started_at DESC);
