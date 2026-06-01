package store

import "testing"

// These tests cover the pure-logic methods of TimeEntry (NetSeconds, IsRunning,
// OnBreak, Validate). They require no database.
//
// The SQL-backed Store methods (StartTimer, StopTimer, breaks, List, Create,
// Update, Delete, migrations) are intentionally not unit-tested here: they
// require a real Postgres or MySQL instance and belong in an integration test
// suite running against a live Mattermost database.

func TestTimeEntry_IsRunning(t *testing.T) {
	tests := []struct {
		name  string
		entry TimeEntry
		want  bool
	}{
		{name: "no end is running", entry: TimeEntry{EndAt: 0}, want: true},
		{name: "with end is not running", entry: TimeEntry{EndAt: 1000}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.IsRunning(); got != tt.want {
				t.Errorf("IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeEntry_OnBreak(t *testing.T) {
	tests := []struct {
		name  string
		entry TimeEntry
		want  bool
	}{
		{name: "break started is on break", entry: TimeEntry{BreakStartedAt: 5000}, want: true},
		{name: "no break started is not on break", entry: TimeEntry{BreakStartedAt: 0}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.OnBreak(); got != tt.want {
				t.Errorf("OnBreak() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeEntry_NetSeconds(t *testing.T) {
	tests := []struct {
		name  string
		entry TimeEntry
		now   int64
		want  int64
	}{
		{
			name:  "closed entry, no breaks",
			entry: TimeEntry{StartAt: 0, EndAt: 60_000},
			now:   0,
			want:  60,
		},
		{
			name:  "closed entry with completed breaks",
			entry: TimeEntry{StartAt: 0, EndAt: 60_000, BreakSeconds: 20},
			now:   0,
			want:  40,
		},
		{
			name:  "running entry uses now as end",
			entry: TimeEntry{StartAt: 0, EndAt: 0},
			now:   30_000,
			want:  30,
		},
		{
			name:  "running entry with active break",
			entry: TimeEntry{StartAt: 0, EndAt: 0, BreakStartedAt: 10_000},
			now:   30_000,
			want:  10, // gross 30s - active break (30-10=20s) = 10s
		},
		{
			name:  "running entry with completed and active break",
			entry: TimeEntry{StartAt: 0, EndAt: 0, BreakSeconds: 5, BreakStartedAt: 20_000},
			now:   30_000,
			want:  15, // gross 30 - (5 completed + 10 active) = 15
		},
		{
			name:  "clamps at zero when breaks exceed gross",
			entry: TimeEntry{StartAt: 0, EndAt: 60_000, BreakSeconds: 120},
			now:   0,
			want:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.NetSeconds(tt.now); got != tt.want {
				t.Errorf("NetSeconds(%d) = %d, want %d", tt.now, got, tt.want)
			}
		})
	}
}

func TestTimeEntry_Validate(t *testing.T) {
	tests := []struct {
		name    string
		entry   TimeEntry
		wantErr bool
	}{
		{
			name:    "valid running entry",
			entry:   TimeEntry{UserID: "user1", StartAt: 1000},
			wantErr: false,
		},
		{
			name:    "valid closed entry",
			entry:   TimeEntry{UserID: "user1", StartAt: 1000, EndAt: 2000},
			wantErr: false,
		},
		{
			name:    "missing user_id",
			entry:   TimeEntry{StartAt: 1000},
			wantErr: true,
		},
		{
			name:    "missing start_at",
			entry:   TimeEntry{UserID: "user1", StartAt: 0},
			wantErr: true,
		},
		{
			name:    "end before start",
			entry:   TimeEntry{UserID: "user1", StartAt: 2000, EndAt: 1000},
			wantErr: true,
		},
		{
			name:    "negative break seconds",
			entry:   TimeEntry{UserID: "user1", StartAt: 1000, BreakSeconds: -5},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
