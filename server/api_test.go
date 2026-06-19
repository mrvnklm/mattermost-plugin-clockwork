package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"go.uber.org/mock/gomock"

	"github.com/mrvnklm/mattermost-plugin-clockwork/server/store"
	"github.com/mrvnklm/mattermost-plugin-clockwork/server/store/mocks"
)

// fakeUsers is a hand-rolled userClient stub for handler tests. It records
// permission answers per user and returns canned model.User lookups.
type fakeUsers struct {
	admins map[string]bool
	users  map[string]*model.User
}

func (f *fakeUsers) Get(userID string) (*model.User, error) {
	if u, ok := f.users[userID]; ok {
		return u, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeUsers) HasPermissionTo(userID string, _ *model.Permission) bool {
	return f.admins[userID]
}

// newTestPlugin wires a Plugin with a mock store and fake user client, plus a
// no-op log so writeStoreError's error path does not panic.
func newTestPlugin(t *testing.T, st store.Store, users userClient, cfg *configuration) *Plugin {
	t.Helper()
	p := &Plugin{
		store: st,
		users: users,
	}
	p.configuration = cfg
	p.router = p.initRouter()
	return p
}

// do issues a request through the plugin router with the given caller header.
func do(p *Plugin, method, path, userID, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if userID != "" {
		r.Header.Set("Mattermost-User-ID", userID)
	}
	w := httptest.NewRecorder()
	p.ServeHTTP(nil, w, r)
	return w
}

// --- csvSafe ---

func TestCsvSafe(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"normal", "normal"},
		{"=cmd", "'=cmd"},
		{"+1", "'+1"},
		{"-1", "'-1"},
		{"@x", "'@x"},
		{"\tx", "'\tx"},
		{"\rx", "'\rx"},
		{"a=b", "a=b"}, // only leading char matters
	}
	for _, tt := range tests {
		if got := csvSafe(tt.in); got != tt.want {
			t.Errorf("csvSafe(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// --- parseRange ---

func TestParseRange(t *testing.T) {
	p := &Plugin{}

	t.Run("explicit valid range", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/x?from=100&to=200", nil)
		w := httptest.NewRecorder()
		from, to, ok := p.parseRange(w, r)
		if !ok || from != 100 || to != 200 {
			t.Fatalf("got from=%d to=%d ok=%v", from, to, ok)
		}
	})

	t.Run("defaults when absent", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		w := httptest.NewRecorder()
		from, to, ok := p.parseRange(w, r)
		if !ok || to <= from {
			t.Fatalf("expected a valid default window, got from=%d to=%d ok=%v", from, to, ok)
		}
		// The built-in default is a 7-day lookback.
		if gotDays := (to - from) / (24 * 60 * 60 * 1000); gotDays != 7 {
			t.Fatalf("default window = %d days, want 7", gotDays)
		}
	})

	t.Run("DefaultReportDays config overrides the default window", func(t *testing.T) {
		cp := &Plugin{}
		cp.configuration = &configuration{DefaultReportDays: 30}
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		w := httptest.NewRecorder()
		from, to, ok := cp.parseRange(w, r)
		if !ok {
			t.Fatalf("ok=false (code=%d)", w.Code)
		}
		if gotDays := (to - from) / (24 * 60 * 60 * 1000); gotDays != 30 {
			t.Fatalf("configured window = %d days, want 30", gotDays)
		}
	})

	t.Run("malformed from", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/x?from=abc", nil)
		w := httptest.NewRecorder()
		if _, _, ok := p.parseRange(w, r); ok {
			t.Fatal("expected ok=false")
		}
		if w.Code != http.StatusBadRequest {
			t.Fatalf("code = %d, want 400", w.Code)
		}
	})

	t.Run("to before from", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/x?from=200&to=100", nil)
		w := httptest.NewRecorder()
		if _, _, ok := p.parseRange(w, r); ok {
			t.Fatal("expected ok=false")
		}
		if w.Code != http.StatusBadRequest {
			t.Fatalf("code = %d, want 400", w.Code)
		}
	})
}

// --- writeStoreError mapping ---

func TestWriteStoreError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		want     int
		wantTrue bool
	}{
		{"nil", nil, http.StatusOK, false},
		{"already running", store.ErrAlreadyRunning, http.StatusConflict, true},
		{"not running", store.ErrNotRunning, http.StatusConflict, true},
		{"already on break", store.ErrAlreadyOnBreak, http.StatusConflict, true},
		{"not on break", store.ErrNotOnBreak, http.StatusConflict, true},
		{"locked", store.ErrLocked, http.StatusForbidden, true},
		{"not found", store.ErrNotFound, http.StatusNotFound, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{}
			w := httptest.NewRecorder()
			got := p.writeStoreError(w, tt.err)
			if got != tt.wantTrue {
				t.Fatalf("writeStoreError returned %v, want %v", got, tt.wantTrue)
			}
			if tt.wantTrue && w.Code != tt.want {
				t.Fatalf("code = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

// --- auth middleware ---

func TestAuthMiddleware_RejectsAnonymous(t *testing.T) {
	p := newTestPlugin(t, nil, &fakeUsers{}, &configuration{})
	w := do(p, http.MethodGet, "/api/v1/config", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", w.Code)
	}
}

// --- config endpoint ---

func TestHandleConfig(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		p := newTestPlugin(t, nil, &fakeUsers{}, &configuration{EnableApproval: enabled})
		w := do(p, http.MethodGet, "/api/v1/config", "u1", "")
		if w.Code != http.StatusOK {
			t.Fatalf("code = %d, want 200", w.Code)
		}
		var body struct {
			ApprovalEnabled bool `json:"approval_enabled"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.ApprovalEnabled != enabled {
			t.Fatalf("approval_enabled = %v, want %v", body.ApprovalEnabled, enabled)
		}
	}
}

// --- workflow gating: 404 when disabled ---

func TestWorkflowEndpoints_404WhenDisabled(t *testing.T) {
	p := newTestPlugin(t, nil, &fakeUsers{admins: map[string]bool{"admin": true}}, &configuration{EnableApproval: false})

	cases := []struct {
		method, path, body, user string
	}{
		{http.MethodPost, "/api/v1/timesheet/submit", `{"from":1,"to":2}`, "u1"},
		{http.MethodPost, "/api/v1/timesheet/withdraw", `{"from":1,"to":2}`, "u1"},
		{http.MethodPost, "/api/v1/admin/approve", `{"user_id":"u1","from":1,"to":2}`, "admin"},
		{http.MethodPost, "/api/v1/admin/reject", `{"user_id":"u1","from":1,"to":2}`, "admin"},
		{http.MethodPost, "/api/v1/admin/reopen", `{"user_id":"u1","from":1,"to":2}`, "admin"},
	}
	for _, c := range cases {
		w := do(p, c.method, c.path, c.user, c.body)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s %s: code = %d, want 404", c.method, c.path, w.Code)
		}
	}
}

// --- user workflow transitions (mock store) ---

func TestTimesheetSubmit_OpenToSubmitted(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mocks.NewMockStore(ctrl)
	st.EXPECT().
		SetStatusRange("u1", int64(100), int64(200), store.StatusOpen, store.StatusSubmitted).
		Return(3, nil)

	p := newTestPlugin(t, st, &fakeUsers{}, &configuration{EnableApproval: true})
	w := do(p, http.MethodPost, "/api/v1/timesheet/submit", "u1", `{"from":100,"to":200}`)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Updated int `json:"updated"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Updated != 3 {
		t.Fatalf("updated = %d, want 3", body.Updated)
	}
}

func TestTimesheetWithdraw_SubmittedToOpen(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mocks.NewMockStore(ctrl)
	st.EXPECT().
		SetStatusRange("u1", int64(1), int64(2), store.StatusSubmitted, store.StatusOpen).
		Return(1, nil)

	p := newTestPlugin(t, st, &fakeUsers{}, &configuration{EnableApproval: true})
	w := do(p, http.MethodPost, "/api/v1/timesheet/withdraw", "u1", `{"from":1,"to":2}`)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
}

func TestTimesheetSubmit_BadBody(t *testing.T) {
	p := newTestPlugin(t, nil, &fakeUsers{}, &configuration{EnableApproval: true})
	w := do(p, http.MethodPost, "/api/v1/timesheet/submit", "u1", `{"from":0,"to":0}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", w.Code)
	}
}

// --- admin workflow transitions + gating ---

func TestAdminApprove_SubmittedToApproved(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mocks.NewMockStore(ctrl)
	st.EXPECT().
		SetStatusRange("target", int64(10), int64(20), store.StatusSubmitted, store.StatusApproved).
		Return(2, nil)

	p := newTestPlugin(t, st, &fakeUsers{admins: map[string]bool{"admin": true}}, &configuration{EnableApproval: true})
	w := do(p, http.MethodPost, "/api/v1/admin/approve", "admin", `{"user_id":"target","from":10,"to":20}`)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminReject_SubmittedToOpen(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mocks.NewMockStore(ctrl)
	st.EXPECT().
		SetStatusRange("target", int64(10), int64(20), store.StatusSubmitted, store.StatusOpen).
		Return(1, nil)

	p := newTestPlugin(t, st, &fakeUsers{admins: map[string]bool{"admin": true}}, &configuration{EnableApproval: true})
	w := do(p, http.MethodPost, "/api/v1/admin/reject", "admin", `{"user_id":"target","from":10,"to":20}`)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
}

func TestAdminReopen_ApprovedToOpen(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mocks.NewMockStore(ctrl)
	st.EXPECT().
		SetStatusRange("target", int64(10), int64(20), store.StatusApproved, store.StatusOpen).
		Return(5, nil)

	p := newTestPlugin(t, st, &fakeUsers{admins: map[string]bool{"admin": true}}, &configuration{EnableApproval: true})
	w := do(p, http.MethodPost, "/api/v1/admin/reopen", "admin", `{"user_id":"target","from":10,"to":20}`)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
}

func TestAdminApprove_NonAdminForbidden(t *testing.T) {
	// No store calls expected; a non-admin must be rejected before reaching it.
	ctrl := gomock.NewController(t)
	st := mocks.NewMockStore(ctrl)

	p := newTestPlugin(t, st, &fakeUsers{admins: map[string]bool{}}, &configuration{EnableApproval: true})
	w := do(p, http.MethodPost, "/api/v1/admin/approve", "regular", `{"user_id":"target","from":10,"to":20}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403", w.Code)
	}
}

func TestAdminApprove_MissingUserID(t *testing.T) {
	p := newTestPlugin(t, nil, &fakeUsers{admins: map[string]bool{"admin": true}}, &configuration{EnableApproval: true})
	w := do(p, http.MethodPost, "/api/v1/admin/approve", "admin", `{"from":10,"to":20}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", w.Code)
	}
}

// --- ownership: PUT another user's entry yields 404 ---

func TestUpdateEntry_OwnershipHidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mocks.NewMockStore(ctrl)
	st.EXPECT().Get("e1").Return(&store.TimeEntry{ID: "e1", UserID: "owner", StartAt: 1, EndAt: 2}, nil)

	p := newTestPlugin(t, st, &fakeUsers{}, &configuration{})
	w := do(p, http.MethodPut, "/api/v1/entries/e1", "intruder", `{"end_at":2}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", w.Code)
	}
}

// --- admin entries requires admin ---

func TestAdminEntries_NonAdminForbidden(t *testing.T) {
	p := newTestPlugin(t, nil, &fakeUsers{admins: map[string]bool{}}, &configuration{})
	w := do(p, http.MethodGet, "/api/v1/admin/entries?from=1&to=2", "regular", "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403", w.Code)
	}
}

// --- admin entries passes user_id filter through to the store ---

func TestAdminEntries_UserFilterPassthrough(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mocks.NewMockStore(ctrl)
	st.EXPECT().ListAll("target", int64(1), int64(2)).Return([]*store.TimeEntry{
		{ID: "e1", UserID: "target", StartAt: 1, EndAt: 2, Status: store.StatusOpen},
	}, nil)

	users := &fakeUsers{
		admins: map[string]bool{"admin": true},
		users:  map[string]*model.User{"target": {Id: "target", Username: "bob"}},
	}
	p := newTestPlugin(t, st, users, &configuration{})
	w := do(p, http.MethodGet, "/api/v1/admin/entries?from=1&to=2&user_id=target", "admin", "")
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("bob")) {
		t.Fatalf("expected username map to contain resolved username, got %s", w.Body.String())
	}
}
