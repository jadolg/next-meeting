package notify

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"next-meeting/calendar"

	"github.com/gen2brain/beeep"
)

func TestMarkAndHasBeenNotified(t *testing.T) {
	_ = Clear()
	defer Clear()

	m := &calendar.MeetingInfo{
		Summary: "Test Meeting",
		Start:   time.Now().Add(1 * time.Hour),
		End:     time.Now().Add(2 * time.Hour),
	}

	if HasBeenNotified(m) {
		t.Fatalf("expected not notified initially")
	}

	if err := MarkNotified(m); err != nil {
		t.Fatalf("MarkNotified failed: %v", err)
	}

	if !HasBeenNotified(m) {
		t.Fatalf("expected notified after MarkNotified")
	}

	if _, err := os.Stat(getNotifyFilePath(m)); err != nil {
		t.Fatalf("expected notify file to exist: %v", err)
	}
}

func TestShouldNotifyBehavior(t *testing.T) {
	_ = Clear()
	defer Clear()

	now := time.Now()
	m := &calendar.MeetingInfo{
		Summary: "Soon Meeting",
		Start:   now.Add(30 * time.Second),
		End:     now.Add(90 * time.Second),
	}
	status := &calendar.MeetingStatus{NextMeeting: m}

	if ShouldNotify(status, 1*time.Minute) == nil {
		t.Fatalf("expected ShouldNotify to return meeting for 1m threshold")
	}

	if ShouldNotify(status, 10*time.Second) != nil {
		t.Fatalf("expected ShouldNotify to return nil for 10s threshold")
	}

	if err := MarkNotified(m); err != nil {
		t.Fatalf("MarkNotified failed: %v", err)
	}
	if ShouldNotify(status, 1*time.Minute) != nil {
		t.Fatalf("expected ShouldNotify to return nil after marking notified")
	}
}

func TestEnsureDefaultIconCreatesFile(t *testing.T) {
	p := filepath.Join(os.TempDir(), "next-meeting-icon.png")
	_ = os.Remove(p)

	icon := ensureDefaultIcon()
	if icon == "" {
		t.Fatalf("expected icon path, got empty string")
	}

	if _, err := os.Stat(icon); err != nil {
		t.Fatalf("expected icon file to exist: %v", err)
	}

	_ = os.Remove(icon)
}

func TestSendNotification(t *testing.T) {
	beeep.AppName = "Next Meeting Test"
	if err := beeep.Notify("title", "body", ensureDefaultIcon()); err != nil {
		t.Fatalf("failed to send notification: %v", err)
	}

}
