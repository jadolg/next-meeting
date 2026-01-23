package notify

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	"next-meeting/calendar"

	"github.com/gen2brain/beeep"
)

const notifyDir = "next-meeting-notify"

//go:embed icon.png
var defaultIconBytes []byte

var (
	iconOnce        sync.Once
	defaultIconPath string
)

func getNotifyDir() string {
	return filepath.Join(os.TempDir(), notifyDir)
}

func ensureDefaultIcon() string {
	iconOnce.Do(func() {
		p := filepath.Join(os.TempDir(), "next-meeting-icon.png")
		if _, err := os.Stat(p); err == nil {
			defaultIconPath = p
			return
		}
		if len(defaultIconBytes) == 0 {
			defaultIconPath = ""
			return
		}

		// Decode the embedded bytes and re-encode as truecolor PNG (NRGBA)
		img, _, err := image.Decode(bytes.NewReader(defaultIconBytes))
		if err != nil {
			// fallback: write raw bytes
			if err := os.WriteFile(p, defaultIconBytes, 0644); err != nil {
				defaultIconPath = ""
				return
			}
			defaultIconPath = p
			return
		}

		dst := image.NewNRGBA(img.Bounds())
		draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Src)

		f, err := os.Create(p)
		if err != nil {
			defaultIconPath = ""
			return
		}
		defer f.Close()
		if err := png.Encode(f, dst); err != nil {
			defaultIconPath = ""
			return
		}
		defaultIconPath = p
	})
	return defaultIconPath
}

func getNotificationID(meeting *calendar.MeetingInfo) string {
	data := fmt.Sprintf("%s|%s|%s", meeting.Summary, meeting.Start.Format(time.RFC3339), meeting.End.Format(time.RFC3339))
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

func getNotifyFilePath(meeting *calendar.MeetingInfo) string {
	return filepath.Join(getNotifyDir(), getNotificationID(meeting))
}

func HasBeenNotified(meeting *calendar.MeetingInfo) bool {
	_, err := os.Stat(getNotifyFilePath(meeting))
	return err == nil
}

func MarkNotified(meeting *calendar.MeetingInfo) error {
	dir := getNotifyDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create notify directory: %w", err)
	}

	filePath := getNotifyFilePath(meeting)
	return os.WriteFile(filePath, []byte(meeting.Summary), 0600)
}

func SendNotification(meeting *calendar.MeetingInfo, startsIn time.Duration) error {
	beeep.AppName = "Next Meeting"
	title := meeting.Summary
	var body string
	if startsIn < time.Minute {
		body = fmt.Sprintf("Upcoming meeting — starting now")
	} else {
		body = fmt.Sprintf("Upcoming meeting — in %s", calendar.FormatDuration(startsIn))
	}

	icon := ensureDefaultIcon()
	if err := beeep.Notify(title, body, icon); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}

func ShouldNotify(status *calendar.MeetingStatus, threshold time.Duration) *calendar.MeetingInfo {
	if status.NextMeeting == nil {
		return nil
	}

	startsIn := time.Until(status.NextMeeting.Start)
	if startsIn <= 0 {
		return nil
	}

	if startsIn > threshold {
		return nil
	}

	if HasBeenNotified(status.NextMeeting) {
		return nil
	}

	return status.NextMeeting
}

func Clear() error {
	return os.RemoveAll(getNotifyDir())
}

func CleanOldNotifications() {
	dir := getNotifyDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}
