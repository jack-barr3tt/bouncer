CREATE TABLE deploys (
  id uuid PRIMARY KEY,
  sha text NOT NULL,
  status text NOT NULL CHECK (status IN ('running', 'published', 'failed')),
  log text NOT NULL DEFAULT '',
  started_at timestamptz NOT NULL DEFAULT now(),
  finished_at timestamptz
);

CREATE INDEX deploys_started_at ON deploys (started_at DESC);
