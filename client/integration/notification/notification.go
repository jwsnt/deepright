package notification

import (
	"errors"
	"strings"
)

var ErrEmptyTitle = errors.New("notification title is empty")
var ErrNotificationsNotAllowed = errors.New("Notifications are not allowed for this application")

type Options struct {
	Title   string
	Message string
}

func Supported() bool {
	return supported()
}

func NormalizeOptions(opts Options) (Options, error) {
	opts.Title = strings.TrimSpace(opts.Title)
	opts.Message = strings.TrimSpace(opts.Message)
	if opts.Title == "" {
		return Options{}, ErrEmptyTitle
	}
	return opts, nil
}

func Notify(opts Options) error {
	normalized, err := NormalizeOptions(opts)
	if err != nil {
		return err
	}
	return notify(normalized)
}

func ShouldOpenSettingsForError(err error) bool {
	return errors.Is(err, ErrNotificationsNotAllowed) && settingsSupported()
}

func OpenSettings() error {
	return openSettings()
}
