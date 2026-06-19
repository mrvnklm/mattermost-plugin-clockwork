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
	status VARCHAR(16) NOT NULL DEFAULT 'open',
	created_at BIGINT NOT NULL,
	updated_at BIGINT NOT NULL
)`

	if _, err := s.db.Exec(createTable); err != nil {
		return errors.Wrap(err, "store: create timetracking_entries (postgres)")
	}

	// Additively add the status column for tables created before the approval
	// workflow. Postgres supports ADD COLUMN IF NOT EXISTS, so this is safe to
	// run on every activation. Existing rows default to 'open'.
	const addStatus = `ALTER TABLE timetracking_entries ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'open'`
	if _, err := s.db.Exec(addStatus); err != nil {
		return errors.Wrap(err, "store: add status column (postgres)")
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
	status VARCHAR(16) NOT NULL DEFAULT 'open',
	created_at BIGINT NOT NULL,
	updated_at BIGINT NOT NULL,
	running_user_id VARCHAR(26) GENERATED ALWAYS AS (CASE WHEN end_at IS NULL THEN user_id ELSE NULL END) STORED,
	INDEX idx_tt_user_start (user_id, start_at),
	UNIQUE KEY uniq_tt_user_running (running_user_id)
)`

	if _, err := s.db.Exec(createTable); err != nil {
		return errors.Wrap(err, "store: create timetracking_entries (mysql)")
	}

	// Additively add the status column for tables created before the approval
	// workflow. Stock MySQL 8 has no ADD COLUMN IF NOT EXISTS, so we guard the
	// ALTER on information_schema, keeping the migration idempotent on both
	// MySQL and MariaDB. Existing rows default to 'open'.
	if err := s.addMySQLStatusColumn(); err != nil {
		return errors.Wrap(err, "store: add status column (mysql)")
	}

	// Retroactively add the single-running-entry guard (the generated column and
	// its unique key) on any pre-existing table that lacks it. Tables created by
	// the CREATE TABLE above (or by any released version) already have it, so
	// this is normally a no-op; it exists so the invariant is never silently
	// absent on an upgraded install.
	if err := s.addMySQLRunningGuard(); err != nil {
		return errors.Wrap(err, "store: add running-entry guard (mysql)")
	}

	return nil
}

// addMySQLStatusColumn adds the status column to timetracking_entries only when
// it does not already exist, so re-running the migration on stock MySQL 8 (no
// ADD COLUMN IF NOT EXISTS) is safe.
func (s *SQLStore) addMySQLStatusColumn() error {
	var exists int
	const checkColumn = `
SELECT COUNT(*) FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'timetracking_entries'
  AND COLUMN_NAME = 'status'`
	if err := s.db.QueryRow(checkColumn).Scan(&exists); err != nil {
		return errors.Wrap(err, "store: check status column")
	}
	if exists > 0 {
		return nil
	}
	const addColumn = `ALTER TABLE timetracking_entries ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'open'`
	if _, err := s.db.Exec(addColumn); err != nil {
		return errors.Wrap(err, "store: alter add status column")
	}
	return nil
}

// addMySQLRunningGuard ensures the generated running_user_id column and its
// uniq_tt_user_running unique key exist, adding either if missing. This makes
// the single-running-entry invariant present even on a MySQL table that
// predates it. Adding the unique key fails loudly if the table already holds
// duplicate open rows per user (the invariant is genuinely violated), which is
// the correct outcome rather than running without the guard.
func (s *SQLStore) addMySQLRunningGuard() error {
	var colExists int
	const checkColumn = `
SELECT COUNT(*) FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'timetracking_entries'
  AND COLUMN_NAME = 'running_user_id'`
	if err := s.db.QueryRow(checkColumn).Scan(&colExists); err != nil {
		return errors.Wrap(err, "store: check running_user_id column")
	}
	if colExists == 0 {
		const addCol = `ALTER TABLE timetracking_entries ADD COLUMN running_user_id VARCHAR(26) GENERATED ALWAYS AS (CASE WHEN end_at IS NULL THEN user_id ELSE NULL END) STORED`
		if _, err := s.db.Exec(addCol); err != nil {
			return errors.Wrap(err, "store: alter add running_user_id column")
		}
	}

	var keyExists int
	const checkKey = `
SELECT COUNT(*) FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'timetracking_entries'
  AND INDEX_NAME = 'uniq_tt_user_running'`
	if err := s.db.QueryRow(checkKey).Scan(&keyExists); err != nil {
		return errors.Wrap(err, "store: check uniq_tt_user_running")
	}
	if keyExists == 0 {
		const addKey = `ALTER TABLE timetracking_entries ADD UNIQUE KEY uniq_tt_user_running (running_user_id)`
		if _, err := s.db.Exec(addKey); err != nil {
			return errors.Wrap(err, "store: alter add uniq_tt_user_running")
		}
	}
	return nil
}
