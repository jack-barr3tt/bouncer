CREATE TABLE users (
  id uuid PRIMARY KEY,
  username text NOT NULL,
  username_key text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  role text NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
  disabled boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE user_apps (
  user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  slug text NOT NULL,
  PRIMARY KEY (user_id, slug)
);

CREATE TABLE access_codes (
  id uuid PRIMARY KEY,
  code text NOT NULL UNIQUE,
  label text NOT NULL DEFAULT '',
  expires_at timestamptz NOT NULL,
  max_signups integer NOT NULL CHECK (max_signups > 0),
  revoked_at timestamptz,
  created_by uuid NOT NULL REFERENCES users (id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE access_code_apps (
  code_id uuid NOT NULL REFERENCES access_codes (id) ON DELETE CASCADE,
  slug text NOT NULL,
  PRIMARY KEY (code_id, slug)
);

CREATE TABLE temporary_accounts (
  id uuid PRIMARY KEY,
  access_code_id uuid NOT NULL REFERENCES access_codes (id) ON DELETE CASCADE,
  nickname text NOT NULL,
  fingerprint_hash text NOT NULL,
  created_ip text NOT NULL,
  last_ip text NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (access_code_id, fingerprint_hash)
);

CREATE TABLE sessions (
  id uuid PRIMARY KEY,
  token_hash text NOT NULL UNIQUE,
  user_id uuid REFERENCES users (id) ON DELETE CASCADE,
  temporary_account_id uuid REFERENCES temporary_accounts (id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (
    (user_id IS NOT NULL AND temporary_account_id IS NULL)
    OR (user_id IS NULL AND temporary_account_id IS NOT NULL)
  )
);

CREATE TABLE auth_attempts (
  bucket text PRIMARY KEY,
  failures integer NOT NULL,
  locked_until timestamptz,
  updated_at timestamptz NOT NULL DEFAULT now()
);
