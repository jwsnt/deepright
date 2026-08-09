//go:build darwin

package notification

import (
	"errors"
	"testing"
)

func TestShouldOpenSettingsForErrorOnDarwin(t *testing.T) {
	if !ShouldOpenSettingsForError(ErrNotificationsNotAllowed) {
		t.Fatal("ShouldOpenSettingsForError() = false, want true")
	}
	if ShouldOpenSettingsForError(errors.New("other")) {
		t.Fatal("ShouldOpenSettingsForError() = true, want false for other errors")
	}
}

func TestOpenSettingsUsesNotificationPaneTargets(t *testing.T) {
	oldOpen := notificationDarwinOpenSettingsURLFn
	t.Cleanup(func() {
		notificationDarwinOpenSettingsURLFn = oldOpen
	})

	var got []string
	notificationDarwinOpenSettingsURLFn = func(target string) error {
		got = append(got, target)
		if target == "x-apple.systempreferences:com.apple.preference.notifications" {
			return nil
		}
		return errors.New("unsupported")
	}

	if err := openSettings(); err != nil {
		t.Fatalf("openSettings() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("targets tried = %v, want 2 entries", got)
	}
}
