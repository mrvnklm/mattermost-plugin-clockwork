//go:build integration

// Package store integration tests exercise the SQL-backed Store against a REAL
// Postgres and a REAL MySQL/MariaDB. They are gated behind the `integration`
// build tag AND an env var per driver, so plain `go test ./...` is unaffected:
//
//	CLOCKWORK_TEST_POSTGRES_DSN  e.g. postgres://user:pass@localhost:5432/db?sslmode=disable
//	CLOCKWORK_TEST_MYSQL_DSN     e.g. user:pass@tcp(localhost:3306)/db?parseTime=true
//
// When a driver's env var is unset the corresponding subtests t.Skip(), so the
// suite is a no-op without a database. See docs/TESTING.md for how to run it
// locally with docker.
//
// Run with:
//
//	go test -tags=integration ./server/store/...
package store

import (
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	// Blank-imported DB drivers, registered for database/sql by side effect.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/mattermost/mattermost/server/public/model"
)

// newSQLStoreForTest builds a SQLStore directly from an *sql.DB, bypassing
// NewSQLStore (which needs a live *pluginapi.Client). The client field is left
// nil; the only production code path that touches it (ListAll's row-cap warning)
// guards against a nil client, so this is safe for tests. It does NOT run
// migrations — call s.migrate() explicitly so idempotency can be asserted.
func newSQLStoreForTest(db *sql.DB, driver string) *SQLStore {
	return &SQLStore{
		client: nil,
		db:     db,
		driver: driver,
	}
}

// testDriver describes one database backend the suite runs against.
type testDriver struct {
	name   string // subtest label
	driver string // value stored in SQLStore.driver
	envVar string // DSN env var; subtest skips when unset
	sqlDrv string // database/sql driver name to sql.Open
}

// drivers is the matrix of backends. Postgres uses the model constant the store
// branches on; MySQL uses the local "mysql" literal the store recognizes.
var drivers = []testDriver{
	{name: "postgres", driver: model.DatabaseDriverPostgres, envVar: "CLOCKWORK_TEST_POSTGRES_DSN", sqlDrv: "postgres"},
	{name: "mysql", driver: driverMySQL, envVar: "CLOCKWORK_TEST_MYSQL_DSN", sqlDrv: "mysql"},
}

// openTestDB opens (and pings) the DB for d, skipping the subtest if its DSN env
// var is unset. It registers cleanup to drop the table and close the pool so
// each test starts from a clean schema.
func openTestDB(t *testing.T, d testDriver) *sql.DB {
	t.Helper()
	dsn := os.Getenv(d.envVar)
	if dsn == "" {
		t.Skipf("%s not set; skipping %s integration test", d.envVar, d.name)
	}

	db, err := sql.Open(d.sqlDrv, dsn)
	if err != nil {
		t.Fatalf("open %s: %v", d.name, err)
	}
	// Retry the ping: a container health check can go green while the DB is
	// still accepting-but-initializing (notably MySQL on a cold start), so a
	// single ping is racy in CI. Back off over ~30s before giving up.
	var pingErr error
	for attempt := 0; attempt < 30; attempt++ {
		if pingErr = db.Ping(); pingErr == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if pingErr != nil {
		_ = db.Close()
		t.Fatalf("ping %s (%s) after retries: %v", d.name, d.envVar, pingErr)
	}

	// Start from a clean slate: the table may linger from a previous run.
	dropTable(t, db)
	t.Cleanup(func() {
		dropTable(t, db)
		_ = db.Close()
	})
	return db
}

// dropTable removes the timetracking_entries table if present so a test run is
// not influenced by leftover rows/schema. Errors are reported but not fatal in
// cleanup paths.
func dropTable(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DROP TABLE IF EXISTS timetracking_entries"); err != nil {
		t.Logf("drop table: %v", err)
	}
}

// freshStore opens a DB, builds a store, and runs migrate() once, returning a
// ready-to-use store. The caller gets a clean schema (openTestDB drops the
// table first).
func freshStore(t *testing.T, d testDriver) *SQLStore {
	t.Helper()
	db := openTestDB(t, d)
	s := newSQLStoreForTest(db, d.driver)
	if err := s.migrate(); err != nil {
		t.Fatalf("migrate %s: %v", d.name, err)
	}
	return s
}

// forEachDriver runs fn as a subtest per configured driver. Subtests skip when
// their DSN env var is unset.
func forEachDriver(t *testing.T, fn func(t *testing.T, d testDriver)) {
	t.Helper()
	for _, d := range drivers {
		d := d
		t.Run(d.name, func(t *testing.T) {
			fn(t, d)
		})
	}
}

// --- migration idempotency ---

// TestMigrate_Idempotent verifies migrate() can run repeatedly with no error
// (mirrors the every-activation reality) and that the table is usable after.
func TestMigrate_Idempotent(t *testing.T) {
	forEachDriver(t, func(t *testing.T, d testDriver) {
		db := openTestDB(t, d)
		s := newSQLStoreForTest(db, d.driver)

		// Run the full migration three times; each must succeed.
		for i := 0; i < 3; i++ {
			if err := s.migrate(); err != nil {
				t.Fatalf("migrate run %d: %v", i+1, err)
			}
		}

		// The table must be queryable after repeated migrations.
		if _, err := s.List("nobody", 0, model.GetMillis()+1); err != nil {
			t.Fatalf("List after migrate: %v", err)
		}
	})
}

// TestMigrate_StatusColumnAddedAdditively simulates a table created BEFORE the
// approval workflow (no status column) and verifies migrate() adds status
// additively without dropping data. A pre-existing row must survive and default
// to StatusOpen.
func TestMigrate_StatusColumnAddedAdditively(t *testing.T) {
	forEachDriver(t, func(t *testing.T, d testDriver) {
		db := openTestDB(t, d)

		// Create a legacy table WITHOUT the status column, then seed a row.
		createLegacyTable(t, db, d)
		seedLegacyRow(t, db, d, "legacy-entry-1", "user-legacy")

		s := newSQLStoreForTest(db, d.driver)
		if err := s.migrate(); err != nil {
			t.Fatalf("migrate over legacy table: %v", err)
		}

		// The legacy row must still exist and default to 'open'.
		got, err := s.Get("legacy-entry-1")
		if err != nil {
			t.Fatalf("Get legacy row after migrate: %v", err)
		}
		if got.Status != StatusOpen {
			t.Errorf("legacy row status = %q, want %q", got.Status, StatusOpen)
		}

		// Migrate again to confirm the additive ADD is idempotent.
		if err := s.migrate(); err != nil {
			t.Fatalf("second migrate over upgraded table: %v", err)
		}
	})
}

// createLegacyTable creates timetracking_entries WITHOUT the status column,
// matching the pre-approval-workflow schema, so the additive migration can be
// exercised. The locked column is kept (it predates status).
func createLegacyTable(t *testing.T, db *sql.DB, d testDriver) {
	t.Helper()
	var ddl string
	switch d.driver {
	case model.DatabaseDriverPostgres:
		ddl = `
CREATE TABLE timetracking_entries (
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
	default: // mysql
		ddl = `
CREATE TABLE timetracking_entries (
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
	}
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
}

// seedLegacyRow inserts a closed entry into the legacy (no-status) table.
func seedLegacyRow(t *testing.T, db *sql.DB, d testDriver, id, userID string) {
	t.Helper()
	now := model.GetMillis()
	q := "INSERT INTO timetracking_entries (id, user_id, start_at, end_at, break_seconds, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
	if d.driver == model.DatabaseDriverPostgres {
		q = "INSERT INTO timetracking_entries (id, user_id, start_at, end_at, break_seconds, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)"
	}
	if _, err := db.Exec(q, id, userID, now-3600_000, now, 0, now, now); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
}

// --- single running-entry invariant ---

// TestStartTimer_SingleRunningInvariant_Concurrent fires many concurrent
// StartTimer calls for one user and asserts exactly one succeeds (the rest get
// ErrAlreadyRunning), proving the DB-level unique key (Postgres partial index /
// MySQL generated-column key) holds under contention.
func TestStartTimer_SingleRunningInvariant_Concurrent(t *testing.T) {
	forEachDriver(t, func(t *testing.T, d testDriver) {
		s := freshStore(t, d)
		const userID = "race-user"
		const n = 16

		var (
			wg        sync.WaitGroup
			mu        sync.Mutex
			successes int
			running   int
			others    []error
		)
		now := model.GetMillis()
		start := make(chan struct{})
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start // release all goroutines together to maximize contention
				_, err := s.StartTimer(userID, "proj", "desc", now)
				mu.Lock()
				defer mu.Unlock()
				switch {
				case err == nil:
					successes++
				case err == ErrAlreadyRunning:
					running++
				default:
					others = append(others, err)
				}
			}()
		}
		close(start)
		wg.Wait()

		if len(others) > 0 {
			t.Fatalf("unexpected errors from concurrent StartTimer: %v", others)
		}
		if successes != 1 {
			t.Errorf("successful StartTimer count = %d, want 1", successes)
		}
		if running != n-1 {
			t.Errorf("ErrAlreadyRunning count = %d, want %d", running, n-1)
		}

		// Exactly one open row must exist in the DB.
		if c := countOpenRows(t, s, userID); c != 1 {
			t.Errorf("open rows for user = %d, want 1", c)
		}

		// GetRunning must return that single entry.
		got, err := s.GetRunning(userID)
		if err != nil {
			t.Fatalf("GetRunning: %v", err)
		}
		if got == nil {
			t.Fatal("GetRunning returned nil after a successful StartTimer")
		}
	})
}

// TestStartTimer_AfterStop verifies that once the running entry is stopped a new
// timer can start (the unique key permits many closed rows).
func TestStartTimer_AfterStop(t *testing.T) {
	forEachDriver(t, func(t *testing.T, d testDriver) {
		s := freshStore(t, d)
		const userID = "serial-user"
		now := model.GetMillis()

		if _, err := s.StartTimer(userID, "", "", now); err != nil {
			t.Fatalf("first StartTimer: %v", err)
		}
		if _, err := s.StartTimer(userID, "", "", now); err != ErrAlreadyRunning {
			t.Fatalf("second StartTimer err = %v, want ErrAlreadyRunning", err)
		}
		if _, err := s.StopTimer(userID, now+1000); err != nil {
			t.Fatalf("StopTimer: %v", err)
		}
		// After stop, a new timer must be allowed.
		if _, err := s.StartTimer(userID, "", "", now+2000); err != nil {
			t.Fatalf("StartTimer after stop: %v", err)
		}
		if c := countOpenRows(t, s, userID); c != 1 {
			t.Errorf("open rows after restart = %d, want 1", c)
		}
	})
}

// countOpenRows counts the user's open (end_at IS NULL) rows directly via SQL.
func countOpenRows(t *testing.T, s *SQLStore, userID string) int {
	t.Helper()
	var c int
	q := s.ph("SELECT COUNT(*) FROM timetracking_entries WHERE user_id = ? AND end_at IS NULL")
	if err := s.db.QueryRow(q, userID).Scan(&c); err != nil {
		t.Fatalf("count open rows: %v", err)
	}
	return c
}

// --- SetStatusRange transition matrix ---

// TestSetStatusRange_TransitionMatrix walks a closed entry through the full
// approval lifecycle (open→submitted→approved, reject, reopen) and asserts the
// affected-count and resulting status at each step, on both drivers.
func TestSetStatusRange_TransitionMatrix(t *testing.T) {
	forEachDriver(t, func(t *testing.T, d testDriver) {
		s := freshStore(t, d)
		const userID = "flow-user"
		now := model.GetMillis()
		from := now - 10_000
		to := now + 10_000

		// A single closed, open-status entry in range.
		entryID := createClosedEntry(t, s, userID, now-5_000, now-1_000, StatusOpen)

		assertStatus := func(want string) {
			t.Helper()
			got, err := s.Get(entryID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if got.Status != want {
				t.Fatalf("status = %q, want %q", got.Status, want)
			}
		}

		step := func(label, fromStatus, toStatus string, wantAffected int) {
			t.Helper()
			n, err := s.SetStatusRange(userID, from, to, fromStatus, toStatus)
			if err != nil {
				t.Fatalf("%s: SetStatusRange: %v", label, err)
			}
			if n != wantAffected {
				t.Fatalf("%s: affected = %d, want %d", label, n, wantAffected)
			}
		}

		// submit: open → submitted
		step("submit", StatusOpen, StatusSubmitted, 1)
		assertStatus(StatusSubmitted)

		// approve: submitted → approved
		step("approve", StatusSubmitted, StatusApproved, 1)
		assertStatus(StatusApproved)

		// reopen: approved → open
		step("reopen", StatusApproved, StatusOpen, 1)
		assertStatus(StatusOpen)

		// submit again, then reject: submitted → open
		step("resubmit", StatusOpen, StatusSubmitted, 1)
		step("reject", StatusSubmitted, StatusOpen, 1)
		assertStatus(StatusOpen)

		// A no-op transition (wrong fromStatus) must affect zero rows.
		step("noop", StatusSubmitted, StatusApproved, 0)
		assertStatus(StatusOpen)
	})
}

// TestSetStatusRange_ExcludesRunningEntry proves a running (open-end) entry is
// never swept into a submitted batch, even when its StartAt falls in range.
func TestSetStatusRange_ExcludesRunningEntry(t *testing.T) {
	forEachDriver(t, func(t *testing.T, d testDriver) {
		s := freshStore(t, d)
		const userID = "mixed-user"
		now := model.GetMillis()
		from := now - 10_000
		to := now + 10_000

		// One closed entry (eligible) and one running entry (must be excluded).
		closedID := createClosedEntry(t, s, userID, now-5_000, now-1_000, StatusOpen)
		if _, err := s.StartTimer(userID, "", "", now-2_000); err != nil {
			t.Fatalf("StartTimer (running entry): %v", err)
		}

		// Submit the range: only the closed entry must transition.
		n, err := s.SetStatusRange(userID, from, to, StatusOpen, StatusSubmitted)
		if err != nil {
			t.Fatalf("SetStatusRange: %v", err)
		}
		if n != 1 {
			t.Fatalf("affected = %d, want 1 (running entry must be excluded)", n)
		}

		// The running entry must still be open and running.
		running, err := s.GetRunning(userID)
		if err != nil {
			t.Fatalf("GetRunning: %v", err)
		}
		if running == nil {
			t.Fatal("running entry disappeared")
		}
		if running.Status != StatusOpen {
			t.Errorf("running entry status = %q, want %q", running.Status, StatusOpen)
		}

		// The closed entry must now be submitted.
		closed, err := s.Get(closedID)
		if err != nil {
			t.Fatalf("Get closed: %v", err)
		}
		if closed.Status != StatusSubmitted {
			t.Errorf("closed entry status = %q, want %q", closed.Status, StatusSubmitted)
		}
	})
}

// TestSetStatusRange_AffectedCountMultiple confirms the affected count reflects
// the number of in-range, matching-status, closed entries.
func TestSetStatusRange_AffectedCountMultiple(t *testing.T) {
	forEachDriver(t, func(t *testing.T, d testDriver) {
		s := freshStore(t, d)
		const userID = "batch-user"
		now := model.GetMillis()
		from := now - 100_000
		to := now + 100_000

		// Three in-range closed open entries...
		createClosedEntry(t, s, userID, now-30_000, now-29_000, StatusOpen)
		createClosedEntry(t, s, userID, now-20_000, now-19_000, StatusOpen)
		createClosedEntry(t, s, userID, now-10_000, now-9_000, StatusOpen)
		// ...one already-submitted (wrong fromStatus, excluded)...
		createClosedEntry(t, s, userID, now-5_000, now-4_000, StatusSubmitted)
		// ...and one out-of-range (excluded by the time window).
		createClosedEntry(t, s, userID, now-1_000_000, now-999_000, StatusOpen)

		n, err := s.SetStatusRange(userID, from, to, StatusOpen, StatusSubmitted)
		if err != nil {
			t.Fatalf("SetStatusRange: %v", err)
		}
		if n != 3 {
			t.Errorf("affected = %d, want 3", n)
		}
	})
}

// createClosedEntry inserts a closed entry with an explicit status via the
// store's Create path (status set on the struct), returning its id.
func createClosedEntry(t *testing.T, s *SQLStore, userID string, startAt, endAt int64, status string) string {
	t.Helper()
	e := &TimeEntry{
		UserID:  userID,
		StartAt: startAt,
		EndAt:   endAt,
		Status:  status,
	}
	if err := s.Create(e); err != nil {
		t.Fatalf("Create closed entry: %v", err)
	}
	return e.ID
}
