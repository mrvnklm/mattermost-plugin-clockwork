package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

// SQLStore is the SQL-backed implementation of Store. It supports both the
// Postgres and MySQL databases that a Mattermost server may be configured with.
type SQLStore struct {
	client *pluginapi.Client
	db     *sql.DB
	driver string
}

// columns is the canonical, ordered list of timetracking_entries columns used
// by SELECT/INSERT statements so scanning stays in sync.
const columns = "id, user_id, start_at, end_at, break_seconds, break_started_at, project, description, locked, created_at, updated_at"

// driverMySQL is the SQL driver name a Mattermost server reports for MySQL. The
// model package only exports the Postgres constant (model.DatabaseDriverPostgres),
// so we keep the MySQL name as a local literal.
const driverMySQL = "mysql"

// NewSQLStore constructs a SQLStore backed by the server's master database and
// runs any required schema migrations before returning.
func NewSQLStore(client *pluginapi.Client) (Store, error) {
	db, err := client.Store.GetMasterDB()
	if err != nil {
		return nil, errors.Wrap(err, "store: get master database")
	}
	if db == nil {
		return nil, errors.New("store: master database is not available")
	}

	driver := client.Store.DriverName()
	if driver != model.DatabaseDriverPostgres && driver != driverMySQL {
		return nil, errors.Errorf("store: unsupported database driver %q", driver)
	}

	s := &SQLStore{
		client: client,
		db:     db,
		driver: driver,
	}

	if err := s.migrate(); err != nil {
		return nil, errors.Wrap(err, "store: migration failed")
	}

	return s, nil
}

// ph rewrites a query written with sequential ? placeholders into the dialect
// the active driver expects. MySQL keeps ?; Postgres uses $1,$2,…
//
// NOTE: this is a naive byte scan — every '?' is treated as a placeholder. Do
// not embed a literal '?' inside a string literal or comment in any query
// passed here, or it will be renumbered on Postgres and break the statement.
func (s *SQLStore) ph(query string) string {
	if s.driver != model.DatabaseDriverPostgres {
		return query
	}

	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteString(fmt.Sprintf("$%d", n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

// nullInt converts a Go millis value to a nullable column value: the 0 sentinel
// becomes SQL NULL.
func nullInt(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

// scanEntry scans a single row (in canonical column order) into a TimeEntry,
// mapping NULL end_at/break_started_at back to the 0 sentinel.
func scanEntry(row interface{ Scan(...interface{}) error }) (*TimeEntry, error) {
	var (
		e          TimeEntry
		endAt      sql.NullInt64
		breakStart sql.NullInt64
		project    sql.NullString
		desc       sql.NullString
	)

	if err := row.Scan(
		&e.ID,
		&e.UserID,
		&e.StartAt,
		&endAt,
		&e.BreakSeconds,
		&breakStart,
		&project,
		&desc,
		&e.Locked,
		&e.CreatedAt,
		&e.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if endAt.Valid {
		e.EndAt = endAt.Int64
	}
	if breakStart.Valid {
		e.BreakStartedAt = breakStart.Int64
	}
	e.Project = project.String
	e.Description = desc.String

	return &e, nil
}

// insertEntry inserts e using the supplied executor (DB or Tx).
func (s *SQLStore) insertEntry(ex interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}, e *TimeEntry) error {
	query := s.ph("INSERT INTO timetracking_entries (" + columns + ") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	_, err := ex.Exec(query,
		e.ID,
		e.UserID,
		e.StartAt,
		nullInt(e.EndAt),
		e.BreakSeconds,
		nullInt(e.BreakStartedAt),
		e.Project,
		e.Description,
		e.Locked,
		e.CreatedAt,
		e.UpdatedAt,
	)
	return err
}

// --- live timer ---

// StartTimer opens a new entry for the user, enforcing the single-running-entry
// invariant atomically inside a transaction.
func (s *SQLStore) StartTimer(userID, project, description string, now int64) (*TimeEntry, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, errors.Wrap(err, "store: begin StartTimer tx")
	}
	defer func() { _ = tx.Rollback() }()

	// Check for an existing running entry. On MySQL we add FOR UPDATE so the
	// row/gap lock on the user_id index range serializes concurrent starts
	// (MySQL has no partial unique index). On Postgres the partial unique index
	// idx_tt_user_running is the hard backstop, so no row lock is needed here
	// (and FOR UPDATE is disallowed with aggregates/this read anyway).
	existsQuery := "SELECT id FROM timetracking_entries WHERE user_id = ? AND end_at IS NULL LIMIT 1"
	if s.driver == driverMySQL {
		existsQuery += " FOR UPDATE"
	}
	var existingID string
	switch scanErr := tx.QueryRow(s.ph(existsQuery), userID).Scan(&existingID); {
	case scanErr == nil:
		return nil, ErrAlreadyRunning
	case errors.Is(scanErr, sql.ErrNoRows):
		// No running entry — proceed to insert.
	default:
		return nil, errors.Wrap(scanErr, "store: check running entry")
	}

	created := model.GetMillis()
	e := &TimeEntry{
		ID:          model.NewId(),
		UserID:      userID,
		StartAt:     now,
		Project:     project,
		Description: description,
		CreatedAt:   created,
		UpdatedAt:   created,
	}

	if err := s.insertEntry(tx, e); err != nil {
		// On Postgres the partial unique index idx_tt_user_running is the
		// authoritative guard against a concurrent start slipping past the
		// count check above; map its violation to the domain error.
		if isUniqueViolation(err) {
			return nil, ErrAlreadyRunning
		}
		return nil, errors.Wrap(err, "store: insert running entry")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "store: commit StartTimer tx")
	}

	return e, nil
}

// isUniqueViolation reports whether err is a violation of the Postgres partial
// unique index idx_tt_user_running (one running entry per user). It matches that
// index by name so an unrelated constraint (e.g. the primary key) is never
// mistaken for the running-entry collision. MySQL has no such index and
// serializes starts via FOR UPDATE in StartTimer instead.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "idx_tt_user_running")
}

// mutateRunning loads the user's running entry inside a transaction with a row
// lock (SELECT ... FOR UPDATE), applies fn, and persists the mutated timer
// fields atomically. This serializes concurrent stop/break operations on the
// same entry so a double-click cannot corrupt break accounting.
func (s *SQLStore) mutateRunning(userID string, fn func(e *TimeEntry) error) (*TimeEntry, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, errors.Wrap(err, "store: begin mutateRunning tx")
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRow(s.ph("SELECT "+columns+" FROM timetracking_entries WHERE user_id = ? AND end_at IS NULL FOR UPDATE"), userID)
	e, err := scanEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotRunning
	}
	if err != nil {
		return nil, errors.Wrap(err, "store: load running entry")
	}

	if err := fn(e); err != nil {
		return nil, err
	}

	query := s.ph("UPDATE timetracking_entries SET end_at = ?, break_seconds = ?, break_started_at = ?, updated_at = ? WHERE id = ?")
	if _, err := tx.Exec(query,
		nullInt(e.EndAt),
		e.BreakSeconds,
		nullInt(e.BreakStartedAt),
		e.UpdatedAt,
		e.ID,
	); err != nil {
		return nil, errors.Wrap(err, "store: persist timer")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "store: commit mutateRunning tx")
	}
	return e, nil
}

// StopTimer closes the user's running entry, folding any active break into the
// accumulated break time.
func (s *SQLStore) StopTimer(userID string, now int64) (*TimeEntry, error) {
	return s.mutateRunning(userID, func(e *TimeEntry) error {
		if e.BreakStartedAt != 0 {
			e.BreakSeconds += (now - e.BreakStartedAt) / 1000
			e.BreakStartedAt = 0
		}
		e.EndAt = now
		e.UpdatedAt = model.GetMillis()
		return nil
	})
}

// StartBreak begins a break on the user's running entry.
func (s *SQLStore) StartBreak(userID string, now int64) (*TimeEntry, error) {
	return s.mutateRunning(userID, func(e *TimeEntry) error {
		if e.BreakStartedAt != 0 {
			return ErrAlreadyOnBreak
		}
		e.BreakStartedAt = now
		e.UpdatedAt = model.GetMillis()
		return nil
	})
}

// StopBreak ends the active break on the user's running entry, adding the
// elapsed time to the accumulated break seconds.
func (s *SQLStore) StopBreak(userID string, now int64) (*TimeEntry, error) {
	return s.mutateRunning(userID, func(e *TimeEntry) error {
		if e.BreakStartedAt == 0 {
			return ErrNotOnBreak
		}
		e.BreakSeconds += (now - e.BreakStartedAt) / 1000
		e.BreakStartedAt = 0
		e.UpdatedAt = model.GetMillis()
		return nil
	})
}

// GetRunning returns the user's open entry, or (nil, nil) if none.
func (s *SQLStore) GetRunning(userID string) (*TimeEntry, error) {
	query := s.ph("SELECT " + columns + " FROM timetracking_entries WHERE user_id = ? AND end_at IS NULL")
	e, err := scanEntry(s.db.QueryRow(query, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "store: get running entry")
	}
	return e, nil
}

// --- entries ---

// Get returns an entry by id, or ErrNotFound.
func (s *SQLStore) Get(id string) (*TimeEntry, error) {
	query := s.ph("SELECT " + columns + " FROM timetracking_entries WHERE id = ?")
	e, err := scanEntry(s.db.QueryRow(query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "store: get entry")
	}
	return e, nil
}

// List returns the user's entries whose StartAt is in [from, to), newest first.
func (s *SQLStore) List(userID string, from, to int64) ([]*TimeEntry, error) {
	query := s.ph("SELECT " + columns + " FROM timetracking_entries WHERE user_id = ? AND start_at >= ? AND start_at < ? ORDER BY start_at DESC")
	return s.queryEntries(query, userID, from, to)
}

// ListAll returns entries across users in [from, to), newest first. If userID is
// non-empty it filters to that user.
func (s *SQLStore) ListAll(userID string, from, to int64) ([]*TimeEntry, error) {
	if userID != "" {
		query := s.ph("SELECT " + columns + " FROM timetracking_entries WHERE user_id = ? AND start_at >= ? AND start_at < ? ORDER BY start_at DESC")
		return s.queryEntries(query, userID, from, to)
	}
	query := s.ph("SELECT " + columns + " FROM timetracking_entries WHERE start_at >= ? AND start_at < ? ORDER BY start_at DESC")
	return s.queryEntries(query, from, to)
}

// queryEntries runs a SELECT returning multiple rows and scans them all.
func (s *SQLStore) queryEntries(query string, args ...interface{}) ([]*TimeEntry, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "store: query entries")
	}
	defer rows.Close()

	entries := []*TimeEntry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, errors.Wrap(err, "store: scan entry")
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "store: iterate entries")
	}
	return entries, nil
}

// Create inserts a manual entry, assigning ID/CreatedAt/UpdatedAt if empty.
func (s *SQLStore) Create(e *TimeEntry) error {
	if e.ID == "" {
		e.ID = model.NewId()
	}
	now := model.GetMillis()
	if e.CreatedAt == 0 {
		e.CreatedAt = now
	}
	if e.UpdatedAt == 0 {
		e.UpdatedAt = now
	}

	if err := e.Validate(); err != nil {
		return err
	}

	if err := s.insertEntry(s.db, e); err != nil {
		return errors.Wrap(err, "store: create entry")
	}
	return nil
}

// Update saves changes to an existing owned, closed entry. The ownership/lock
// check and the write happen atomically inside one transaction (row locked with
// FOR UPDATE) to avoid a TOCTOU window. Returns ErrLocked if the stored entry is
// locked, ErrNotFound if it does not exist or is not owned by e.UserID.
func (s *SQLStore) Update(e *TimeEntry) error {
	if err := e.Validate(); err != nil {
		return err
	}
	// Update only edits closed entries. A zero EndAt would re-open the entry
	// (end_at IS NULL), violating the single-running invariant and, on Postgres,
	// colliding with idx_tt_user_running. Running entries are mutated only via
	// the timer methods.
	if e.EndAt == 0 {
		return errors.New("store: update requires a non-zero end_at")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return errors.Wrap(err, "store: begin Update tx")
	}
	defer func() { _ = tx.Rollback() }()

	var locked bool
	row := tx.QueryRow(s.ph("SELECT locked FROM timetracking_entries WHERE id = ? AND user_id = ? FOR UPDATE"), e.ID, e.UserID)
	switch scanErr := row.Scan(&locked); {
	case errors.Is(scanErr, sql.ErrNoRows):
		return ErrNotFound
	case scanErr != nil:
		return errors.Wrap(scanErr, "store: lock entry for update")
	}
	if locked {
		return ErrLocked
	}

	e.UpdatedAt = model.GetMillis()
	query := s.ph("UPDATE timetracking_entries SET start_at = ?, end_at = ?, break_seconds = ?, project = ?, description = ?, updated_at = ? WHERE id = ?")
	if _, err := tx.Exec(query,
		e.StartAt,
		nullInt(e.EndAt),
		e.BreakSeconds,
		e.Project,
		e.Description,
		e.UpdatedAt,
		e.ID,
	); err != nil {
		return errors.Wrap(err, "store: update entry")
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "store: commit Update tx")
	}
	return nil
}

// Delete removes an owned entry. Returns ErrLocked if locked, ErrNotFound if it
// does not exist or is not owned by userID.
func (s *SQLStore) Delete(userID, id string) error {
	existing, err := s.Get(id)
	if err != nil {
		return err
	}
	if existing.UserID != userID {
		return ErrNotFound
	}
	if existing.Locked {
		return ErrLocked
	}

	if _, err := s.db.Exec(s.ph("DELETE FROM timetracking_entries WHERE id = ?"), id); err != nil {
		return errors.Wrap(err, "store: delete entry")
	}
	return nil
}

// Suggestions returns the user's distinct non-empty project and note values,
// most-recently-used first, each capped at limit (for autocomplete).
func (s *SQLStore) Suggestions(userID string, limit int) (projects, notes []string, err error) {
	if projects, err = s.distinctValues(userID, "project", limit); err != nil {
		return nil, nil, err
	}
	if notes, err = s.distinctValues(userID, "description", limit); err != nil {
		return nil, nil, err
	}
	return projects, notes, nil
}

// distinctValues returns distinct non-empty values of a fixed column for the
// user, ordered by most-recent use. column is a compile-time constant supplied
// by Suggestions (never user input), so interpolating it is safe.
func (s *SQLStore) distinctValues(userID, column string, limit int) ([]string, error) {
	query := s.ph("SELECT " + column + " FROM timetracking_entries WHERE user_id = ? AND " + column + " <> '' GROUP BY " + column + " ORDER BY MAX(start_at) DESC LIMIT ?")
	rows, err := s.db.Query(query, userID, limit)
	if err != nil {
		return nil, errors.Wrap(err, "store: query suggestions")
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, errors.Wrap(err, "store: scan suggestion")
		}
		values = append(values, v)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "store: iterate suggestions")
	}
	return values, nil
}
