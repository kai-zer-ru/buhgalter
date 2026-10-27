-- +goose Up
ALTER TABLE users ADD COLUMN kind TEXT NOT NULL DEFAULT 'person'
  CHECK (kind IN ('person', 'household'));

CREATE TABLE households (
  id                TEXT PRIMARY KEY,
  name              TEXT NOT NULL,
  owner_user_id     TEXT NOT NULL REFERENCES users(id),
  principal_user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  created_at        TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at        TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE household_members (
  household_id TEXT NOT NULL REFERENCES households(id) ON DELETE CASCADE,
  user_id      TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  role         TEXT NOT NULL CHECK (role IN ('owner', 'member')),
  joined_at    TEXT NOT NULL DEFAULT (datetime('now')),
  PRIMARY KEY (household_id, user_id)
);
CREATE UNIQUE INDEX idx_household_one_owner ON household_members(household_id) WHERE role = 'owner';

CREATE TABLE household_invites (
  id            TEXT PRIMARY KEY,
  household_id  TEXT NOT NULL REFERENCES households(id) ON DELETE CASCADE,
  code          TEXT NOT NULL UNIQUE,
  status        TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'used', 'revoked')),
  created_by    TEXT NOT NULL REFERENCES users(id),
  created_at    TEXT NOT NULL DEFAULT (datetime('now')),
  expires_at    TEXT NOT NULL,
  used_by       TEXT REFERENCES users(id) ON DELETE SET NULL,
  used_at       TEXT
);

INSERT OR IGNORE INTO feature_flags (key, enabled) VALUES ('household', 1);

-- +goose Down
-- Forward-only migration.
