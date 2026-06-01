package store

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

// migrate creates the timetracking_entries table and its supporting index if
// they do not already exist. It is idempotent and safe to run on every plugin
// activation. DDL is branched per driver because Postgres and MySQL disagree on
// inline-index and CREATE INDEX IF NOT EXISTS syntax.
func (s *SQLStore) migrate() error {
	switch s.driver {
	case model.DatabaseDriverPostgres:
		return s.migratePostgres()
	case driverMySQL:
		return s.migrateMySQL()
	default:
		return errors.Errorf("store: unsupported database driver %q", s.driver)
	}
}

// migratePostgres creates the table, then the index separately (Postgres does
// not support inline INDEX in CREATE TABLE but does support CREATE INDEX IF NOT
// EXISTS).
func (s *SQLStore) migratePostgres() error {
	const createTable = `
CREATE TABLE IF NOT EXISTS timetracking_entries (
	id VARCHAR(26) PRIMARY KEY,
	user_id VARCHAR(26) NOT NULL,
	start_at BIGINT NOT NULL,
	end_at BIGINT NULL,
	break_seconds BIGINT NOT NULL DEFAULT 0,
	break_started_at BIGINT NULL,
	project VARCHAR(255) NULL,
	description TEXT NULL,
	locked BOOLEAN NOT NULL DEFAULT FALSE,
	created_at BIGINT NOT NULL,
	updated_at BIGINT NOT NULL
)`

	if _, err := s.db.Exec(createTable); err != nil {
		return errors.Wrap(err, "store: create timetracking_entries (postgres)")
	}

	const createIndex = `CREATE INDEX IF NOT EXISTS idx_tt_user_start ON timetracking_entries (user_id, start_at)`
	if _, err := s.db.Exec(createIndex); err != nil {
		return errors.Wrap(err, "store: create idx_tt_user_start (postgres)")
	}

	// Partial unique index enforces at most one running (open) entry per user,
	// closing the start-timer race. MySQL has no partial indexes, so there the
	// invariant relies on the in-transaction count check in StartTimer.
	const createRunningIndex = `CREATE UNIQUE INDEX IF NOT EXISTS idx_tt_user_running ON timetracking_entries (user_id) WHERE end_at IS NULL`
	if _, err := s.db.Exec(createRunningIndex); err != nil {
		return errors.Wrap(err, "store: create idx_tt_user_running (postgres)")
	}

	return nil
}

// migrateMySQL creates the table with the index inlined (MySQL supports inline
// INDEX but not CREATE INDEX IF NOT EXISTS), giving an idempotent single
// statement.
func (s *SQLStore) migrateMySQL() error {
	// MySQL has no partial indexes, so the single-running-entry invariant is
	// enforced with a STORED generated column that equals user_id only while the
	// entry is open (end_at NULL) and is NULL otherwise — a UNIQUE key on it
	// permits many closed rows (NULLs repeat) but at most one open row per user.
	const createTable = `
CREATE TABLE IF NOT EXISTS timetracking_entries (
	id VARCHAR(26) PRIMARY KEY,
	user_id VARCHAR(26) NOT NULL,
	start_at BIGINT NOT NULL,
	end_at BIGINT NULL,
	break_seconds BIGINT NOT NULL DEFAULT 0,
	break_started_at BIGINT NULL,
	project VARCHAR(255) NULL,
	description TEXT NULL,
	locked BOOLEAN NOT NULL DEFAULT FALSE,
	created_at BIGINT NOT NULL,
	updated_at BIGINT NOT NULL,
	running_user_id VARCHAR(26) GENERATED ALWAYS AS (CASE WHEN end_at IS NULL THEN user_id ELSE NULL END) STORED,
	INDEX idx_tt_user_start (user_id, start_at),
	UNIQUE KEY uniq_tt_user_running (running_user_id)
)`

	if _, err := s.db.Exec(createTable); err != nil {
		return errors.Wrap(err, "store: create timetracking_entries (mysql)")
	}

	return nil
}
