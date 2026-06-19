// Package store defines the time-tracking domain type (TimeEntry) and the
// persistence interface (Store) used by the rest of the plugin. The SQL-backed
// implementation lives in sqlstore.go.
package store

import (
	"errors"

	"github.com/mattermost/mattermost/server/public/model"
)

// Sanity bounds for manually created/edited entries. These guard against
// fat-finger and malicious inputs (year-long spans, far-future timestamps) that
// would otherwise distort reports. They are deliberately generous so legitimate
// overnight or long shifts are not rejected.
const (
	// maxEntryMillis caps a single entry's gross span (start→end).
	maxEntryMillis = 24 * 60 * 60 * 1000 // 24h
	// futureSkewMillis tolerates minor client/server clock skew on end_at so a
	// just-now clock-out is never rejected as "in the future".
	futureSkewMillis = 5 * 60 * 1000 // 5m
)

// Sentinel errors returned by Store implementations. The API layer maps these
// to HTTP status codes; the command layer maps them to user-facing messages.
var (
	// ErrAlreadyRunning is returned by StartTimer when the user already has a
	// running (open) entry.
	ErrAlreadyRunning = errors.New("a timer is already running")
	// ErrNotRunning is returned by StopTimer/StartBreak/StopBreak when the user
	// has no running entry.
	ErrNotRunning = errors.New("no timer is running")
	// ErrAlreadyOnBreak is returned by StartBreak when a break is already active.
	ErrAlreadyOnBreak = errors.New("already on a break")
	// ErrNotOnBreak is returned by StopBreak when no break is active.
	ErrNotOnBreak = errors.New("not currently on a break")
	// ErrLocked is returned by Update/Delete when the target entry is locked.
	ErrLocked = errors.New("entry is locked and cannot be modified")
	// ErrNotFound is returned when an entry does not exist (or is not owned by
	// the requesting user, to avoid leaking existence).
	ErrNotFound = errors.New("entry not found")
)

// Approval-workflow statuses for a TimeEntry. The lifecycle is
// open → submitted → approved, with reject (submitted→open) and reopen
// (approved→open) returning an entry to the editable state.
const (
	// StatusOpen is the default: the entry is editable by its owner.
	StatusOpen = "open"
	// StatusSubmitted means the owner has submitted the entry for approval; it
	// is locked for the owner and awaits an admin decision.
	StatusSubmitted = "submitted"
	// StatusApproved means an admin has approved the entry; it stays locked
	// until an admin reopens it.
	StatusApproved = "approved"
)

// TimeEntry is a single work-time record (one clock-in → clock-out cycle).
//
// Time fields are UTC unix milliseconds. A zero value means "unset":
//   - EndAt == 0        → the entry is still running (open).
//   - BreakStartedAt == 0 → the user is not currently on a break.
//
// Net worked time is gross duration (start→end) minus total break time; see
// NetSeconds. The SQL layer maps the zero sentinels to/from SQL NULL.
type TimeEntry struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	StartAt        int64  `json:"start_at"`         // UTC unix millis, required
	EndAt          int64  `json:"end_at"`           // UTC unix millis; 0 ⇒ running
	BreakSeconds   int64  `json:"break_seconds"`    // accumulated completed break time
	BreakStartedAt int64  `json:"break_started_at"` // UTC unix millis; 0 ⇒ not on break
	Project        string `json:"project"`          // optional (hybrid model)
	Description    string `json:"description"`      // optional (hybrid model)
	Status         string `json:"status"`           // open|submitted|approved (approval workflow)
	Locked         bool   `json:"locked"`           // DERIVED: status != "open"; kept for the existing lock banner
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

// syncLocked derives the Locked convenience flag from Status. Locked is a pure
// function of Status (locked == status != open), kept in the JSON so the
// existing frontend lock banner keeps working without referencing status.
func (e *TimeEntry) syncLocked() {
	if e.Status == "" {
		e.Status = StatusOpen
	}
	e.Locked = e.Status != StatusOpen
}

// IsRunning reports whether the entry is still open (no end time).
func (e *TimeEntry) IsRunning() bool { return e.EndAt == 0 }

// OnBreak reports whether a break is currently active.
func (e *TimeEntry) OnBreak() bool { return e.BreakStartedAt != 0 }

// NetSeconds returns the net worked seconds as of the reference instant now
// (UTC unix millis). For a running entry, now is used as the end. Any active
// (not-yet-closed) break is subtracted as well. The result is clamped at 0.
func (e *TimeEntry) NetSeconds(now int64) int64 {
	end := e.EndAt
	if end == 0 {
		end = now
	}
	gross := (end - e.StartAt) / 1000
	breaks := e.BreakSeconds
	if e.BreakStartedAt != 0 {
		breaks += (now - e.BreakStartedAt) / 1000
	}
	net := gross - breaks
	if net < 0 {
		return 0
	}
	return net
}

// Validate checks invariants for a manually created or edited entry, using the
// current server time for the not-in-future bound. See ValidateAt for the
// deterministic, testable form.
func (e *TimeEntry) Validate() error {
	return e.ValidateAt(model.GetMillis())
}

// ValidateAt checks invariants relative to the reference instant now (UTC unix
// millis), so callers (and tests) can validate deterministically.
func (e *TimeEntry) ValidateAt(now int64) error {
	if e.UserID == "" {
		return errors.New("user_id is required")
	}
	if e.StartAt <= 0 {
		return errors.New("start_at is required")
	}
	if e.EndAt != 0 && e.EndAt < e.StartAt {
		return errors.New("end_at must not be before start_at")
	}
	if e.BreakSeconds < 0 {
		return errors.New("break_seconds must not be negative")
	}
	// Absolute-time sanity (only meaningful for closed entries).
	if e.EndAt != 0 {
		if e.EndAt-e.StartAt > maxEntryMillis {
			return errors.New("entry duration exceeds the 24h limit")
		}
		if e.EndAt > now+futureSkewMillis {
			return errors.New("end_at must not be in the future")
		}
	}
	if e.StartAt > now+futureSkewMillis {
		return errors.New("start_at must not be in the future")
	}
	return nil
}

// Store is the persistence boundary for time entries. All time arguments are
// UTC unix milliseconds.
//
// Implementation contract (see sqlstore.go):
//
//	func NewSQLStore(client *pluginapi.Client) (Store, error)
//
// NewSQLStore must run any required schema migrations before returning.
type Store interface {
	// --- live timer (single running entry per user, enforced atomically) ---

	// StartTimer opens a new entry for the user. Returns ErrAlreadyRunning if
	// one is already open. project and description are optional.
	StartTimer(userID, project, description string, now int64) (*TimeEntry, error)
	// StopTimer closes the running entry. If a break is active it is ended
	// first (its elapsed time added to BreakSeconds). Returns ErrNotRunning.
	StopTimer(userID string, now int64) (*TimeEntry, error)
	// StartBreak begins a break on the running entry. Returns ErrNotRunning or
	// ErrAlreadyOnBreak.
	StartBreak(userID string, now int64) (*TimeEntry, error)
	// StopBreak ends the active break, adding elapsed time to BreakSeconds.
	// Returns ErrNotRunning or ErrNotOnBreak.
	StopBreak(userID string, now int64) (*TimeEntry, error)
	// GetRunning returns the user's open entry, or (nil, nil) if none.
	GetRunning(userID string) (*TimeEntry, error)

	// --- entries ---

	// Get returns an entry by id, or ErrNotFound.
	Get(id string) (*TimeEntry, error)
	// List returns the user's entries whose StartAt is in [from, to), newest
	// first.
	List(userID string, from, to int64) ([]*TimeEntry, error)
	// ListAll returns entries across users in [from, to), newest first. If
	// userID is non-empty it filters to that user. Intended for admin use; the
	// API layer is responsible for permission gating.
	ListAll(userID string, from, to int64) ([]*TimeEntry, error)
	// Create inserts a manual entry. If ID/CreatedAt/UpdatedAt are empty they
	// are assigned. The entry must Validate.
	Create(e *TimeEntry) error
	// Update saves changes to an existing owned entry, bumping UpdatedAt.
	// Returns ErrLocked if the stored entry is locked, ErrNotFound otherwise.
	Update(e *TimeEntry) error
	// Delete removes an owned entry. Returns ErrLocked if locked, ErrNotFound
	// if it does not exist or is not owned by userID.
	Delete(userID, id string) error

	// SetStatusRange transitions the user's entries whose StartAt is in
	// [from, to) and whose current status equals fromStatus to toStatus,
	// transactionally. Running (open-end) entries are never transitioned. It
	// returns the number of rows affected. This single primitive powers
	// submit/withdraw (user) and approve/reject/reopen (admin) by varying the
	// status pair.
	SetStatusRange(userID string, from, to int64, fromStatus, toStatus string) (int, error)

	// Suggestions returns the user's distinct non-empty project and note values,
	// most-recently-used first, each capped at limit. Used to power autocomplete.
	Suggestions(userID string, limit int) (projects, notes []string, err error)
}
