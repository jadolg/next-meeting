package calendar

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// MeetingInfo contains information about a calendar event
type MeetingInfo struct {
	Summary   string
	Start     time.Time
	End       time.Time
	IsAllDay  bool
	Location  string
	Attendees int
}

// MeetingStatus represents the current meeting status
type MeetingStatus struct {
	CurrentMeeting *MeetingInfo
	NextMeeting    *MeetingInfo
}

// Service wraps the Google Calendar API service
type Service struct {
	svc *calendar.Service
}

// NewService creates a new Calendar service
func NewService(ctx context.Context, client *http.Client) (*Service, error) {
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create calendar service: %w", err)
	}
	return &Service{svc: svc}, nil
}

// GetTodayEvents fetches all events for today from the primary calendar
func (s *Service) GetTodayEvents(ctx context.Context) ([]*MeetingInfo, error) {
	now := time.Now()

	// Query events from now onwards, limited to today
	year, month, day := now.Date()
	tomorrow := time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())

	timeMin := now.Add(-2 * time.Hour).Format(time.RFC3339) // Include events that may have started recently
	timeMax := tomorrow.Format(time.RFC3339)

	events, err := s.svc.Events.List("primary").
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(timeMin).
		TimeMax(timeMax).
		OrderBy("startTime").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve events: %w", err)
	}

	var result []*MeetingInfo

	for _, item := range events.Items {
		// Skip all-day events
		if item.Start.DateTime == "" {
			continue
		}

		start, err := time.Parse(time.RFC3339, item.Start.DateTime)
		if err != nil {
			continue
		}

		end, err := time.Parse(time.RFC3339, item.End.DateTime)
		if err != nil {
			continue
		}

		meeting := &MeetingInfo{
			Summary:   item.Summary,
			Start:     start,
			End:       end,
			Location:  item.Location,
			Attendees: len(item.Attendees),
		}

		result = append(result, meeting)
	}

	return result, nil
}

// GetMeetingStatus calculates current and next meetings from a list of events
func GetMeetingStatus(events []*MeetingInfo) *MeetingStatus {
	now := time.Now()
	status := &MeetingStatus{}

	for _, meeting := range events {
		// Check if this is a current meeting (happening now)
		if now.After(meeting.Start) && now.Before(meeting.End) {
			if status.CurrentMeeting == nil {
				status.CurrentMeeting = meeting
			}
		} else if now.Before(meeting.Start) {
			// This is a future meeting
			if status.NextMeeting == nil {
				status.NextMeeting = meeting
			}
		}

		// If we have both current and next meeting, we can stop
		if status.CurrentMeeting != nil && status.NextMeeting != nil {
			break
		}
	}

	return status
}

// FormatDuration returns a human-readable duration string
func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "overdue"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
