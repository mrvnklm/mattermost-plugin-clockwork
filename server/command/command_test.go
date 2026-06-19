package command

import (
	"errors"
	"testing"

	"github.com/mrvnklm/mattermost-plugin-clockwork/server/store"
)

func TestFriendlyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"already running", store.ErrAlreadyRunning, "You're already clocked in."},
		{"not running", store.ErrNotRunning, "You're not clocked in."},
		{"already on break", store.ErrAlreadyOnBreak, "You're already on a break."},
		{"not on break", store.ErrNotOnBreak, "You're not currently on a break."},
		{"unknown maps to generic", errors.New("boom"), "Something went wrong. Please try again."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := friendlyError(tt.err); got != tt.want {
				t.Errorf("friendlyError(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}
