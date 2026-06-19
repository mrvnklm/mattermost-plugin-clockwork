package command

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"

	"github.com/mrvnklm/mattermost-plugin-clockwork/server/store"
)

// Handler holds the dependencies needed to execute slash commands.
type Handler struct {
	client *pluginapi.Client
	store  store.Store
}

// Command is the interface the plugin uses to dispatch slash commands.
type Command interface {
	Handle(args *model.CommandArgs) (*model.CommandResponse, error)
}

const trackCommandTrigger = "track"

// NewCommandHandler registers the /track slash command and returns a handler.
func NewCommandHandler(client *pluginapi.Client, st store.Store) Command {
	if err := client.SlashCommand.Register(&model.Command{
		Trigger:          trackCommandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Track your work time",
		AutoCompleteHint: "[in|out|break|status]",
		AutocompleteData: trackAutocompleteData(),
	}); err != nil {
		client.Log.Error("Failed to register command", "error", err)
	}

	return &Handler{
		client: client,
		store:  st,
	}
}

func trackAutocompleteData() *model.AutocompleteData {
	track := model.NewAutocompleteData(trackCommandTrigger, "[subcommand]", "Track your work time")

	in := model.NewAutocompleteData("in", "[description]", "Clock in and start the timer")
	track.AddCommand(in)

	out := model.NewAutocompleteData("out", "", "Clock out and stop the timer")
	track.AddCommand(out)

	brk := model.NewAutocompleteData("break", "", "Toggle a break on your running timer")
	track.AddCommand(brk)

	status := model.NewAutocompleteData("status", "", "Show your current timer status")
	track.AddCommand(status)

	return track
}

// Handle parses the subcommand from the slash command text and dispatches it.
func (c *Handler) Handle(args *model.CommandArgs) (*model.CommandResponse, error) {
	fields := strings.Fields(args.Command)
	if len(fields) < 2 {
		return c.ephemeral("Usage: `/track [in|out|break|status]`"), nil
	}

	subcommand := strings.ToLower(fields[1])
	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args.Command), fields[0]))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, fields[1]))

	switch subcommand {
	case "in":
		return c.handleIn(args, rest), nil
	case "out":
		return c.handleOut(args), nil
	case "break":
		return c.handleBreak(args), nil
	case "status":
		return c.handleStatus(args), nil
	default:
		return c.ephemeral(fmt.Sprintf("Unknown subcommand: `%s`. Try `/track [in|out|break|status]`.", subcommand)), nil
	}
}

func (c *Handler) handleIn(args *model.CommandArgs, description string) *model.CommandResponse {
	_, err := c.store.StartTimer(args.UserId, "", description, model.GetMillis())
	if err != nil {
		return c.ephemeral(friendlyError(err))
	}
	if description != "" {
		return c.ephemeral(fmt.Sprintf("Clocked in. Tracking: %s", description))
	}
	return c.ephemeral("Clocked in. Your timer is running.")
}

func (c *Handler) handleOut(args *model.CommandArgs) *model.CommandResponse {
	now := model.GetMillis()
	entry, err := c.store.StopTimer(args.UserId, now)
	if err != nil {
		return c.ephemeral(friendlyError(err))
	}
	hours := float64(entry.NetSeconds(now)) / 3600
	return c.ephemeral(fmt.Sprintf("Clocked out. Net time worked: %.2f hours.", hours))
}

func (c *Handler) handleBreak(args *model.CommandArgs) *model.CommandResponse {
	running, err := c.store.GetRunning(args.UserId)
	if err != nil {
		return c.ephemeral(friendlyError(err))
	}
	if running == nil {
		return c.ephemeral(friendlyError(store.ErrNotRunning))
	}

	if running.OnBreak() {
		if _, err := c.store.StopBreak(args.UserId, model.GetMillis()); err != nil {
			return c.ephemeral(friendlyError(err))
		}
		return c.ephemeral("Break ended. Back on the clock.")
	}

	if _, err := c.store.StartBreak(args.UserId, model.GetMillis()); err != nil {
		return c.ephemeral(friendlyError(err))
	}
	return c.ephemeral("Break started. Enjoy it!")
}

func (c *Handler) handleStatus(args *model.CommandArgs) *model.CommandResponse {
	running, err := c.store.GetRunning(args.UserId)
	if err != nil {
		return c.ephemeral(friendlyError(err))
	}
	if running == nil {
		return c.ephemeral("You're not clocked in.")
	}

	hours := float64(running.NetSeconds(model.GetMillis())) / 3600
	state := "running"
	if running.OnBreak() {
		state = "on break"
	}
	msg := fmt.Sprintf("Timer is %s. Net time worked: %.2f hours.", state, hours)
	if running.Description != "" {
		msg += fmt.Sprintf("\nTracking: %s", running.Description)
	}
	return c.ephemeral(msg)
}

func (c *Handler) ephemeral(text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}

// friendlyError maps store sentinel errors to user-facing English messages.
func friendlyError(err error) string {
	switch {
	case errors.Is(err, store.ErrAlreadyRunning):
		return "You're already clocked in."
	case errors.Is(err, store.ErrNotRunning):
		return "You're not clocked in."
	case errors.Is(err, store.ErrAlreadyOnBreak):
		return "You're already on a break."
	case errors.Is(err, store.ErrNotOnBreak):
		return "You're not currently on a break."
	default:
		return "Something went wrong. Please try again."
	}
}
