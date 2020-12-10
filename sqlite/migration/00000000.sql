CREATE TABLE users (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL,
	email      TEXT UNIQUE,
	api_key    TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE auths (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id       INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	source        TEXT NOT NULL,
	source_id     TEXT NOT NULL,
	access_token  TEXT NOT NULL,
	refresh_token TEXT NOT NULL,
	expiry        TEXT,
	created_at    TEXT NOT NULL,
	updated_at    TEXT NOT NULL,

	UNIQUE(user_id, source),  -- one source per user
	UNIQUE(source, source_id) -- one auth per source user
);

CREATE TABLE dials (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id     INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name        TEXT NOT NULL,
	invite_code TEXT UNIQUE NOT NULL,
	value       INTEGER NOT NULL DEFAULT 0,
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);

CREATE INDEX dials_user_id_idx ON dials (user_id);

CREATE TABLE dial_values (
	dial_id      INTEGER NOT NULL REFERENCES dials (id) ON DELETE CASCADE,
	"timestamp"  TEXT NOT NULL, -- per-minute precision
	value        INTEGER NOT NULL,

	PRIMARY KEY (dial_id, "timestamp")
);

CREATE TABLE dial_memberships (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	dial_id    INTEGER NOT NULL REFERENCES dials (id) ON DELETE CASCADE,
	user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	value      INTEGER NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,

	UNIQUE(dial_id, user_id)
);

CREATE INDEX dial_memberships_dial_id_idx ON dial_memberships (dial_id);
CREATE INDEX dial_memberships_user_id_idx ON dial_memberships (user_id);
