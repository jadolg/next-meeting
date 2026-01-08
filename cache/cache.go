package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"next-meeting/calendar"
)

const (
	cacheFileName = "next-meeting-cache.json"
	cacheDuration = 30 * time.Minute
)

// CachedData represents the structure stored in the cache file
type CachedData struct {
	Timestamp     time.Time               `json:"timestamp"`
	MeetingStatus *calendar.MeetingStatus `json:"meeting_status"`
}

// GetPath returns the path to the cache file
func GetPath() string {
	return filepath.Join(os.TempDir(), cacheFileName)
}

// Read reads cached meeting status from file.
// Returns nil if cache doesn't exist or is expired.
func Read() *calendar.MeetingStatus {
	data, err := os.ReadFile(GetPath())
	if err != nil {
		return nil
	}

	var cached CachedData
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil
	}

	// Check if cache has expired
	if time.Since(cached.Timestamp) > cacheDuration {
		return nil
	}

	return cached.MeetingStatus
}

// Write writes meeting status to the cache file
func Write(status *calendar.MeetingStatus) error {
	cached := CachedData{
		Timestamp:     time.Now(),
		MeetingStatus: status,
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return os.WriteFile(GetPath(), data, 0600)
}

// Clear deletes the cache file
func Clear() error {
	err := os.Remove(GetPath())
	if os.IsNotExist(err) {
		return nil // Not an error if file doesn't exist
	}
	return err
}
