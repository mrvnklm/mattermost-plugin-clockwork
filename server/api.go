package main

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"

	"github.com/vsjwl/mm-time-tracking/server/store"
)

// defaultWindow is the lookback used for list/summary/export endpoints when no
// explicit from/to range is supplied.
const defaultWindow = 7 * 24 * time.Hour

// userClient is the slice of the Mattermost user API the HTTP handlers depend
// on. It is satisfied by *pluginapi.UserService and lets handler tests inject a
// mock for username resolution and admin permission checks.
type userClient interface {
	Get(userID string) (*model.User, error)
	HasPermissionTo(userID string, permission *model.Permission) bool
}

// initRouter initializes the HTTP router for the plugin.
func (p *Plugin) initRouter() *mux.Router {
	router := mux.NewRouter()

	// Middleware to require that the user is logged in.
	router.Use(p.MattermostAuthorizationRequired)

	api := router.PathPrefix("/api/v1").Subrouter()

	// Plugin configuration (any logged-in user).
	api.HandleFunc("/config", p.handleConfig).Methods(http.MethodGet)

	// Live timer.
	api.HandleFunc("/timer/current", p.handleTimerCurrent).Methods(http.MethodGet)
	api.HandleFunc("/timer/start", p.handleTimerStart).Methods(http.MethodPost)
	api.HandleFunc("/timer/stop", p.handleTimerStop).Methods(http.MethodPost)
	api.HandleFunc("/timer/break/start", p.handleBreakStart).Methods(http.MethodPost)
	api.HandleFunc("/timer/break/stop", p.handleBreakStop).Methods(http.MethodPost)

	// Entries (current user).
	api.HandleFunc("/entries", p.handleListEntries).Methods(http.MethodGet)
	api.HandleFunc("/entries", p.handleCreateEntry).Methods(http.MethodPost)
	api.HandleFunc("/entries/{id}", p.handleUpdateEntry).Methods(http.MethodPut)
	api.HandleFunc("/entries/{id}", p.handleDeleteEntry).Methods(http.MethodDelete)

	// Reports (current user).
	api.HandleFunc("/reports/summary", p.handleReportSummary).Methods(http.MethodGet)
	api.HandleFunc("/reports/export", p.handleReportExport).Methods(http.MethodGet)

	// Autocomplete suggestions (current user).
	api.HandleFunc("/suggestions", p.handleSuggestions).Methods(http.MethodGet)

	// Approval workflow (current user). Each handler returns 404 when the
	// approval workflow is disabled in the plugin configuration.
	api.HandleFunc("/timesheet/submit", p.handleTimesheetSubmit).Methods(http.MethodPost)
	api.HandleFunc("/timesheet/withdraw", p.handleTimesheetWithdraw).Methods(http.MethodPost)

	// Admin.
	api.HandleFunc("/admin/entries", p.handleAdminEntries).Methods(http.MethodGet)
	api.HandleFunc("/admin/export", p.handleAdminExport).Methods(http.MethodGet)

	// Approval workflow (admin). Also gated on EnableApproval (404 when off).
	api.HandleFunc("/admin/approve", p.handleAdminApprove).Methods(http.MethodPost)
	api.HandleFunc("/admin/reject", p.handleAdminReject).Methods(http.MethodPost)
	api.HandleFunc("/admin/reopen", p.handleAdminReopen).Methods(http.MethodPost)

	return router
}

// ServeHTTP routes incoming plugin HTTP requests through the gorilla mux router.
// The root URL is <siteUrl>/plugins/com.vsjwl.mm-time-tracking/api/v1/.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}

// MattermostAuthorizationRequired rejects any request lacking the
// Mattermost-User-ID header injected by the server for authenticated sessions.
func (p *Plugin) MattermostAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- JSON helpers ---

func (p *Plugin) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		p.client.Log.Error("Failed to write JSON response", "error", err)
	}
}

func (p *Plugin) writeError(w http.ResponseWriter, status int, msg string) {
	p.writeJSON(w, status, map[string]string{"error": msg})
}

// writeStoreError maps store sentinel errors to HTTP status codes and writes the
// error response. Returns true if err was non-nil (and handled).
func (p *Plugin) writeStoreError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, store.ErrAlreadyRunning),
		errors.Is(err, store.ErrNotRunning),
		errors.Is(err, store.ErrAlreadyOnBreak),
		errors.Is(err, store.ErrNotOnBreak):
		p.writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrLocked):
		p.writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, store.ErrNotFound):
		p.writeError(w, http.StatusNotFound, err.Error())
	default:
		p.client.Log.Error("Store operation failed", "error", err)
		p.writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

// parseRange reads from/to query params as int64 unix millis. Missing values
// default to [now-defaultWindow, now). Returns ok=false (and writes 400) on a
// malformed integer.
func (p *Plugin) parseRange(w http.ResponseWriter, r *http.Request) (from, to int64, ok bool) {
	now := model.GetMillis()
	window := defaultWindow
	if d := p.getConfiguration().DefaultReportDays; d > 0 {
		window = time.Duration(d) * 24 * time.Hour
	}
	from = now - window.Milliseconds()
	to = now

	q := r.URL.Query()
	if v := q.Get("from"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			p.writeError(w, http.StatusBadRequest, "invalid 'from' parameter")
			return 0, 0, false
		}
		from = parsed
	}
	if v := q.Get("to"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			p.writeError(w, http.StatusBadRequest, "invalid 'to' parameter")
			return 0, 0, false
		}
		to = parsed
	}
	if to < from {
		p.writeError(w, http.StatusBadRequest, "'to' must not be before 'from'")
		return 0, 0, false
	}
	return from, to, true
}

// userLocation resolves a user's preferred Mattermost timezone to a
// *time.Location, falling back to UTC.
func (p *Plugin) userLocation(userID string) *time.Location {
	user, err := p.users.Get(userID)
	if err != nil {
		return time.UTC
	}
	return locationForUser(user)
}

func locationForUser(user *model.User) *time.Location {
	name := model.GetPreferredTimezone(user.Timezone)
	if name == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// --- Live timer handlers ---

func (p *Plugin) handleTimerCurrent(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	entry, err := p.store.GetRunning(userID)
	if p.writeStoreError(w, err) {
		return
	}
	p.writeJSON(w, http.StatusOK, map[string]interface{}{"entry": entry})
}

func (p *Plugin) handleTimerStart(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")

	var body struct {
		Project     string `json:"project"`
		Description string `json:"description"`
	}
	// Body is optional; ignore decode errors for an empty body.
	_ = json.NewDecoder(r.Body).Decode(&body)

	entry, err := p.store.StartTimer(userID, body.Project, body.Description, model.GetMillis())
	if p.writeStoreError(w, err) {
		return
	}
	p.writeJSON(w, http.StatusOK, map[string]interface{}{"entry": entry})
}

func (p *Plugin) handleTimerStop(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	entry, err := p.store.StopTimer(userID, model.GetMillis())
	if p.writeStoreError(w, err) {
		return
	}
	p.writeJSON(w, http.StatusOK, map[string]interface{}{"entry": entry})
}

func (p *Plugin) handleBreakStart(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	entry, err := p.store.StartBreak(userID, model.GetMillis())
	if p.writeStoreError(w, err) {
		return
	}
	p.writeJSON(w, http.StatusOK, map[string]interface{}{"entry": entry})
}

func (p *Plugin) handleBreakStop(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	entry, err := p.store.StopBreak(userID, model.GetMillis())
	if p.writeStoreError(w, err) {
		return
	}
	p.writeJSON(w, http.StatusOK, map[string]interface{}{"entry": entry})
}

// --- Entry handlers ---

func (p *Plugin) handleListEntries(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	from, to, ok := p.parseRange(w, r)
	if !ok {
		return
	}
	entries, err := p.store.List(userID, from, to)
	if p.writeStoreError(w, err) {
		return
	}
	if entries == nil {
		entries = []*store.TimeEntry{}
	}
	p.writeJSON(w, http.StatusOK, map[string]interface{}{"entries": entries})
}

func (p *Plugin) handleCreateEntry(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")

	var entry store.TimeEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		p.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Never trust caller-supplied identity/metadata.
	entry.UserID = userID
	entry.ID = ""
	entry.Locked = false
	entry.Status = store.StatusOpen
	entry.CreatedAt = 0
	entry.UpdatedAt = 0

	if err := entry.Validate(); err != nil {
		p.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// A manual entry must be closed. Open (running) entries are created only
	// through the live-timer workflow, which owns the single-running invariant.
	if entry.EndAt == 0 {
		p.writeError(w, http.StatusBadRequest, "end_at is required for a manual entry")
		return
	}

	if err := p.store.Create(&entry); err != nil {
		if p.writeStoreError(w, err) {
			return
		}
	}
	p.writeJSON(w, http.StatusCreated, map[string]interface{}{"entry": &entry})
}

func (p *Plugin) handleUpdateEntry(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	id := mux.Vars(r)["id"]

	existing, err := p.store.Get(id)
	if p.writeStoreError(w, err) {
		return
	}
	// Ownership check: hide existence of other users' entries.
	if existing.UserID != userID {
		p.writeError(w, http.StatusNotFound, store.ErrNotFound.Error())
		return
	}

	// Decode partial update with pointer fields so absent keys leave the stored
	// value untouched.
	var patch struct {
		StartAt      *int64  `json:"start_at"`
		EndAt        *int64  `json:"end_at"`
		BreakSeconds *int64  `json:"break_seconds"`
		Project      *string `json:"project"`
		Description  *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		p.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if patch.StartAt != nil {
		existing.StartAt = *patch.StartAt
	}
	if patch.EndAt != nil {
		existing.EndAt = *patch.EndAt
	}
	if patch.BreakSeconds != nil {
		existing.BreakSeconds = *patch.BreakSeconds
	}
	if patch.Project != nil {
		existing.Project = *patch.Project
	}
	if patch.Description != nil {
		existing.Description = *patch.Description
	}

	// A closed entry must not be re-opened via PUT (that belongs to the timer
	// workflow and would violate the single-running invariant). This also blocks
	// editing a still-running entry through this endpoint — stop it first.
	if existing.EndAt == 0 {
		p.writeError(w, http.StatusBadRequest, "an end time is required; stop the running timer before editing")
		return
	}

	if err := existing.Validate(); err != nil {
		p.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if p.writeStoreError(w, p.store.Update(existing)) {
		return
	}
	p.writeJSON(w, http.StatusOK, map[string]interface{}{"entry": existing})
}

func (p *Plugin) handleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	id := mux.Vars(r)["id"]

	if p.writeStoreError(w, p.store.Delete(userID, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Report handlers ---

type daySummary struct {
	Date         string             `json:"date"`
	NetSeconds   int64              `json:"net_seconds"`
	BreakSeconds int64              `json:"break_seconds"`
	Entries      []*store.TimeEntry `json:"entries"`
}

type summaryResponse struct {
	TotalNetSeconds int64        `json:"total_net_seconds"`
	Days            []daySummary `json:"days"`
}

func (p *Plugin) handleReportSummary(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	from, to, ok := p.parseRange(w, r)
	if !ok {
		return
	}

	entries, err := p.store.List(userID, from, to)
	if p.writeStoreError(w, err) {
		return
	}

	loc := p.userLocation(userID)
	now := model.GetMillis()

	// Group entries by local calendar date.
	byDate := map[string]*daySummary{}
	var order []string
	for _, e := range entries {
		key := time.UnixMilli(e.StartAt).In(loc).Format("2006-01-02")
		day, found := byDate[key]
		if !found {
			day = &daySummary{Date: key, Entries: []*store.TimeEntry{}}
			byDate[key] = day
			order = append(order, key)
		}
		day.NetSeconds += e.NetSeconds(now)
		day.BreakSeconds += totalBreakSeconds(e, now)
		day.Entries = append(day.Entries, e)
	}

	// Newest day first.
	sort.Sort(sort.Reverse(sort.StringSlice(order)))

	resp := summaryResponse{Days: make([]daySummary, 0, len(order))}
	for _, key := range order {
		resp.Days = append(resp.Days, *byDate[key])
		resp.TotalNetSeconds += byDate[key].NetSeconds
	}

	p.writeJSON(w, http.StatusOK, resp)
}

// totalBreakSeconds returns accumulated break seconds including any active break
// as of now.
func totalBreakSeconds(e *store.TimeEntry, now int64) int64 {
	breaks := e.BreakSeconds
	if e.BreakStartedAt != 0 {
		breaks += (now - e.BreakStartedAt) / 1000
	}
	if breaks < 0 {
		return 0
	}
	return breaks
}

func (p *Plugin) handleReportExport(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	from, to, ok := p.parseRange(w, r)
	if !ok {
		return
	}

	entries, err := p.store.List(userID, from, to)
	if p.writeStoreError(w, err) {
		return
	}

	loc := p.userLocation(userID)
	p.writeCSV(w, entries, false, func(string) *time.Location { return loc }, nil)
}

func (p *Plugin) handleSuggestions(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	projects, notes, err := p.store.Suggestions(userID, 50)
	if p.writeStoreError(w, err) {
		return
	}
	if projects == nil {
		projects = []string{}
	}
	if notes == nil {
		notes = []string{}
	}
	p.writeJSON(w, http.StatusOK, map[string]interface{}{"projects": projects, "notes": notes})
}

// --- Config handler ---

// handleConfig returns the subset of plugin configuration the webapp needs to
// decide whether to show the approval-workflow UI. Any logged-in user may read
// it (the auth middleware already enforces a session).
func (p *Plugin) handleConfig(w http.ResponseWriter, r *http.Request) {
	cfg := p.getConfiguration()
	p.writeJSON(w, http.StatusOK, map[string]interface{}{
		"approval_enabled": cfg.EnableApproval,
	})
}

// --- Approval workflow handlers ---

// decodeRange decodes a {"from":<ms>,"to":<ms>} JSON body and validates the
// range. It writes a 400 and returns ok=false on a malformed body or range.
func (p *Plugin) decodeRange(w http.ResponseWriter, r *http.Request) (from, to int64, ok bool) {
	var body struct {
		From int64 `json:"from"`
		To   int64 `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		p.writeError(w, http.StatusBadRequest, "invalid request body")
		return 0, 0, false
	}
	if body.From <= 0 || body.To <= 0 {
		p.writeError(w, http.StatusBadRequest, "'from' and 'to' are required")
		return 0, 0, false
	}
	if body.To < body.From {
		p.writeError(w, http.StatusBadRequest, "'to' must not be before 'from'")
		return 0, 0, false
	}
	return body.From, body.To, true
}

// approvalEnabled reports whether the approval workflow is on. When off it
// writes a 404 so the workflow endpoints are indistinguishable from unrouted
// paths, and returns false.
func (p *Plugin) approvalEnabled(w http.ResponseWriter) bool {
	if !p.getConfiguration().EnableApproval {
		p.writeError(w, http.StatusNotFound, "not found")
		return false
	}
	return true
}

// transition runs a SetStatusRange for the given user/range/status pair and
// writes the {"updated":<n>} response (or maps the store error).
func (p *Plugin) transition(w http.ResponseWriter, userID string, from, to int64, fromStatus, toStatus string) {
	n, err := p.store.SetStatusRange(userID, from, to, fromStatus, toStatus)
	if p.writeStoreError(w, err) {
		return
	}
	p.writeJSON(w, http.StatusOK, map[string]interface{}{"updated": n})
}

func (p *Plugin) handleTimesheetSubmit(w http.ResponseWriter, r *http.Request) {
	if !p.approvalEnabled(w) {
		return
	}
	userID := r.Header.Get("Mattermost-User-ID")
	from, to, ok := p.decodeRange(w, r)
	if !ok {
		return
	}
	p.transition(w, userID, from, to, store.StatusOpen, store.StatusSubmitted)
}

func (p *Plugin) handleTimesheetWithdraw(w http.ResponseWriter, r *http.Request) {
	if !p.approvalEnabled(w) {
		return
	}
	userID := r.Header.Get("Mattermost-User-ID")
	from, to, ok := p.decodeRange(w, r)
	if !ok {
		return
	}
	p.transition(w, userID, from, to, store.StatusSubmitted, store.StatusOpen)
}

// decodeAdminRange decodes an admin workflow body {"user_id","from","to"} after
// verifying admin permission and that the workflow is enabled. Returns ok=false
// (response already written) on any failure.
func (p *Plugin) decodeAdminRange(w http.ResponseWriter, r *http.Request) (targetUser string, from, to int64, ok bool) {
	if !p.approvalEnabled(w) {
		return "", 0, 0, false
	}
	caller := r.Header.Get("Mattermost-User-ID")
	if !p.requireAdmin(w, caller) {
		return "", 0, 0, false
	}
	var body struct {
		UserID string `json:"user_id"`
		From   int64  `json:"from"`
		To     int64  `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		p.writeError(w, http.StatusBadRequest, "invalid request body")
		return "", 0, 0, false
	}
	if body.UserID == "" {
		p.writeError(w, http.StatusBadRequest, "user_id is required")
		return "", 0, 0, false
	}
	if body.From <= 0 || body.To <= 0 {
		p.writeError(w, http.StatusBadRequest, "'from' and 'to' are required")
		return "", 0, 0, false
	}
	if body.To < body.From {
		p.writeError(w, http.StatusBadRequest, "'to' must not be before 'from'")
		return "", 0, 0, false
	}
	return body.UserID, body.From, body.To, true
}

func (p *Plugin) handleAdminApprove(w http.ResponseWriter, r *http.Request) {
	targetUser, from, to, ok := p.decodeAdminRange(w, r)
	if !ok {
		return
	}
	p.transition(w, targetUser, from, to, store.StatusSubmitted, store.StatusApproved)
}

func (p *Plugin) handleAdminReject(w http.ResponseWriter, r *http.Request) {
	targetUser, from, to, ok := p.decodeAdminRange(w, r)
	if !ok {
		return
	}
	p.transition(w, targetUser, from, to, store.StatusSubmitted, store.StatusOpen)
}

func (p *Plugin) handleAdminReopen(w http.ResponseWriter, r *http.Request) {
	targetUser, from, to, ok := p.decodeAdminRange(w, r)
	if !ok {
		return
	}
	p.transition(w, targetUser, from, to, store.StatusApproved, store.StatusOpen)
}

// --- Admin handlers ---

func (p *Plugin) requireAdmin(w http.ResponseWriter, userID string) bool {
	if !p.users.HasPermissionTo(userID, model.PermissionManageSystem) {
		p.writeError(w, http.StatusForbidden, "forbidden")
		return false
	}
	return true
}

func (p *Plugin) handleAdminEntries(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if !p.requireAdmin(w, userID) {
		return
	}
	from, to, ok := p.parseRange(w, r)
	if !ok {
		return
	}
	filterUser := r.URL.Query().Get("user_id")

	entries, err := p.store.ListAll(filterUser, from, to)
	if p.writeStoreError(w, err) {
		return
	}
	if entries == nil {
		entries = []*store.TimeEntry{}
	}

	// Resolve usernames once per user so the admin table can show names, not IDs.
	// Bounded so a very wide range can't trigger an unbounded number of per-user
	// API calls; beyond the cap we fall back to the raw id.
	const maxUserLookups = 500
	usernames := map[string]string{}
	for _, e := range entries {
		if _, ok := usernames[e.UserID]; ok {
			continue
		}
		if len(usernames) >= maxUserLookups {
			usernames[e.UserID] = e.UserID
			continue
		}
		if u, uerr := p.users.Get(e.UserID); uerr == nil {
			usernames[e.UserID] = u.Username
		} else {
			usernames[e.UserID] = e.UserID
		}
	}

	p.writeJSON(w, http.StatusOK, map[string]interface{}{"entries": entries, "usernames": usernames})
}

func (p *Plugin) handleAdminExport(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if !p.requireAdmin(w, userID) {
		return
	}
	from, to, ok := p.parseRange(w, r)
	if !ok {
		return
	}
	filterUser := r.URL.Query().Get("user_id")

	entries, err := p.store.ListAll(filterUser, from, to)
	if p.writeStoreError(w, err) {
		return
	}

	// Cache per-user lookups for timezone and username resolution.
	userCache := map[string]*model.User{}
	lookup := func(uid string) *model.User {
		if u, found := userCache[uid]; found {
			return u
		}
		u, uerr := p.users.Get(uid)
		if uerr != nil {
			u = nil
		}
		userCache[uid] = u
		return u
	}

	locFor := func(uid string) *time.Location {
		if u := lookup(uid); u != nil {
			return locationForUser(u)
		}
		return time.UTC
	}
	nameFor := func(uid string) string {
		if u := lookup(uid); u != nil {
			return u.Username
		}
		return uid
	}

	p.writeCSV(w, entries, true, locFor, nameFor)
}

// --- CSV writing ---

// csvSafe neutralizes spreadsheet formula injection: a cell whose first
// character is one of = + - @ (or a leading tab/CR) is prefixed with a single
// quote so Excel/Sheets/LibreOffice treat it as text, not a formula.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

// writeCSV streams entries as a CSV attachment. When admin is true a leading
// username column is added; locFor resolves the timezone for a given row's user
// and nameFor (admin only) resolves the username.
func (p *Plugin) writeCSV(
	w http.ResponseWriter,
	entries []*store.TimeEntry,
	admin bool,
	locFor func(userID string) *time.Location,
	nameFor func(userID string) string,
) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="timesheet.csv"`)
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{"date", "start", "end", "break_minutes", "net_hours", "project", "description"}
	if admin {
		header = append([]string{"username"}, header...)
	}
	if err := cw.Write(header); err != nil {
		p.client.Log.Error("Failed to write CSV header", "error", err)
		return
	}

	now := model.GetMillis()
	for _, e := range entries {
		loc := locFor(e.UserID)

		date := time.UnixMilli(e.StartAt).In(loc).Format("2006-01-02")
		start := time.UnixMilli(e.StartAt).In(loc).Format(time.RFC3339)
		// For a still-running entry leave end/break/net blank so the exported
		// row never shows totals that don't correspond to a clock-out.
		end := ""
		breakMinutes := ""
		netHours := ""
		if e.EndAt != 0 {
			end = time.UnixMilli(e.EndAt).In(loc).Format(time.RFC3339)
			breakMinutes = strconv.FormatInt(totalBreakSeconds(e, now)/60, 10)
			netHours = strconv.FormatFloat(float64(e.NetSeconds(now))/3600, 'f', 2, 64)
		}

		record := []string{date, start, end, breakMinutes, netHours, csvSafe(e.Project), csvSafe(e.Description)}
		if admin {
			record = append([]string{csvSafe(nameFor(e.UserID))}, record...)
		}
		if err := cw.Write(record); err != nil {
			p.client.Log.Error("Failed to write CSV row", "error", err)
			return
		}
	}
}
