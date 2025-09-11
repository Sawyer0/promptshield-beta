-- Global directory of platform users (source of truth: Clerk userId)
CREATE TABLE IF NOT EXISTS platform_users (
  id TEXT PRIMARY KEY,                 -- Clerk userId
  email TEXT,
  first_name TEXT,
  last_name TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_platform_users_email ON platform_users(email);

