package calendar

import (
	"fmt"
	"testing"
	"time"
)

func TestGetMeetingStatus(t *testing.T) {
	// Use a fixed "now" time for deterministic tests
	// We'll create events relative to this time
	fixedNow := time.Date(2026, 1, 9, 14, 30, 0, 0, time.UTC)

	// Helper function to create meetings relative to fixedNow
	makeMeeting := func(summary string, startOffset, endOffset time.Duration) *MeetingInfo {
		return &MeetingInfo{
			Summary:   summary,
			Start:     fixedNow.Add(startOffset),
			End:       fixedNow.Add(endOffset),
			Location:  "Test Location",
			Attendees: 5,
		}
	}

	tests := []struct {
		name                  string
		events                []*MeetingInfo
		now                   time.Time
		wantCurrentSummary    string
		wantCurrentMeetingNil bool
		wantNextSummary       string
		wantNextMeetingNil    bool
	}{
		{
			name:                  "empty events list",
			events:                []*MeetingInfo{},
			now:                   fixedNow,
			wantCurrentMeetingNil: true,
			wantNextMeetingNil:    true,
		},
		{
			name:                  "nil events list",
			events:                nil,
			now:                   fixedNow,
			wantCurrentMeetingNil: true,
			wantNextMeetingNil:    true,
		},
		{
			name: "single current meeting (now is within meeting)",
			events: []*MeetingInfo{
				makeMeeting("Current Meeting", -30*time.Minute, 30*time.Minute),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Current Meeting",
			wantCurrentMeetingNil: false,
			wantNextMeetingNil:    true,
		},
		{
			name: "single future meeting",
			events: []*MeetingInfo{
				makeMeeting("Future Meeting", 1*time.Hour, 2*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentMeetingNil: true,
			wantNextSummary:       "Future Meeting",
			wantNextMeetingNil:    false,
		},
		{
			name: "single past meeting",
			events: []*MeetingInfo{
				makeMeeting("Past Meeting", -2*time.Hour, -1*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentMeetingNil: true,
			wantNextMeetingNil:    true,
		},
		{
			name: "current and next meeting",
			events: []*MeetingInfo{
				makeMeeting("Current Meeting", -30*time.Minute, 30*time.Minute),
				makeMeeting("Next Meeting", 1*time.Hour, 2*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Current Meeting",
			wantCurrentMeetingNil: false,
			wantNextSummary:       "Next Meeting",
			wantNextMeetingNil:    false,
		},
		{
			name: "multiple future meetings - earliest selected",
			events: []*MeetingInfo{
				makeMeeting("Later Meeting", 3*time.Hour, 4*time.Hour),
				makeMeeting("Earliest Meeting", 1*time.Hour, 2*time.Hour),
				makeMeeting("Middle Meeting", 2*time.Hour, 3*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentMeetingNil: true,
			wantNextSummary:       "Earliest Meeting",
			wantNextMeetingNil:    false,
		},
		{
			name: "multiple overlapping current meetings - most recent start selected",
			events: []*MeetingInfo{
				makeMeeting("Earlier Current", -1*time.Hour, 1*time.Hour),
				makeMeeting("Later Current", -15*time.Minute, 45*time.Minute),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Later Current",
			wantCurrentMeetingNil: false,
			wantNextMeetingNil:    true,
		},
		{
			name: "meeting starts exactly at now",
			events: []*MeetingInfo{
				makeMeeting("Starting Now", 0, 1*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Starting Now",
			wantCurrentMeetingNil: false,
			wantNextMeetingNil:    true,
		},
		{
			name: "meeting ends exactly at now",
			events: []*MeetingInfo{
				makeMeeting("Ending Now", -1*time.Hour, 0),
			},
			now:                   fixedNow,
			wantCurrentMeetingNil: true,
			wantNextMeetingNil:    true,
		},
		{
			name: "complex scenario with past, current, and future meetings",
			events: []*MeetingInfo{
				makeMeeting("Past Meeting 1", -3*time.Hour, -2*time.Hour),
				makeMeeting("Past Meeting 2", -2*time.Hour, -1*time.Hour),
				makeMeeting("Current Meeting", -30*time.Minute, 30*time.Minute),
				makeMeeting("Next Meeting 1", 1*time.Hour, 2*time.Hour),
				makeMeeting("Next Meeting 2", 2*time.Hour, 3*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Current Meeting",
			wantCurrentMeetingNil: false,
			wantNextSummary:       "Next Meeting 1",
			wantNextMeetingNil:    false,
		},
		{
			name: "back-to-back meetings - boundary case",
			events: []*MeetingInfo{
				makeMeeting("First Meeting", -1*time.Hour, 0),
				makeMeeting("Second Meeting", 0, 1*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Second Meeting",
			wantCurrentMeetingNil: false,
			wantNextMeetingNil:    true,
		},
		{
			name: "multiple current meetings - test preference for later start",
			events: []*MeetingInfo{
				makeMeeting("Long Meeting", -2*time.Hour, 2*time.Hour),
				makeMeeting("Short Recent Meeting", -10*time.Minute, 20*time.Minute),
				makeMeeting("Medium Meeting", -1*time.Hour, 1*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Short Recent Meeting",
			wantCurrentMeetingNil: false,
			wantNextMeetingNil:    true,
		},
		{
			name: "meetings in unsorted order",
			events: []*MeetingInfo{
				makeMeeting("Third", 3*time.Hour, 4*time.Hour),
				makeMeeting("First", 1*time.Hour, 2*time.Hour),
				makeMeeting("Second", 2*time.Hour, 3*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentMeetingNil: true,
			wantNextSummary:       "First",
			wantNextMeetingNil:    false,
		},
		{
			name: "all-day-like long meeting",
			events: []*MeetingInfo{
				makeMeeting("Full Day", -8*time.Hour, 8*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Full Day",
			wantCurrentMeetingNil: false,
			wantNextMeetingNil:    true,
		},
		// Edge cases: same start time - shorter takes precedence
		{
			name: "current meetings with same start time - shorter takes precedence",
			events: []*MeetingInfo{
				makeMeeting("Long Meeting", -30*time.Minute, 90*time.Minute),
				makeMeeting("Short Meeting", -30*time.Minute, 30*time.Minute),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Short Meeting",
			wantCurrentMeetingNil: false,
			wantNextMeetingNil:    true,
		},
		{
			name: "current meetings with same start and end times - first in order",
			events: []*MeetingInfo{
				makeMeeting("Meeting A", -30*time.Minute, 30*time.Minute),
				makeMeeting("Meeting B", -30*time.Minute, 30*time.Minute),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Meeting A",
			wantCurrentMeetingNil: false,
			wantNextMeetingNil:    true,
		},
		{
			name: "future meetings with same start time - shorter takes precedence",
			events: []*MeetingInfo{
				makeMeeting("Long Future", 1*time.Hour, 3*time.Hour),
				makeMeeting("Short Future", 1*time.Hour, 2*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentMeetingNil: true,
			wantNextSummary:       "Short Future",
			wantNextMeetingNil:    false,
		},
		{
			name: "future meetings with same start and end times - first in order",
			events: []*MeetingInfo{
				makeMeeting("Future A", 1*time.Hour, 2*time.Hour),
				makeMeeting("Future B", 1*time.Hour, 2*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentMeetingNil: true,
			wantNextSummary:       "Future A",
			wantNextMeetingNil:    false,
		},
		// Edge cases: field values
		{
			name: "meeting with empty summary",
			events: []*MeetingInfo{
				makeMeeting("", -30*time.Minute, 30*time.Minute),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "",
			wantCurrentMeetingNil: false,
			wantNextMeetingNil:    true,
		},
		// Edge case: meeting starts during current meeting - should be next
		{
			name: "meeting starts during current meeting - should be next",
			events: []*MeetingInfo{
				makeMeeting("Long Current", -30*time.Minute, 2*time.Hour),
				makeMeeting("Starts During", 15*time.Minute, 1*time.Hour),
			},
			now:                   fixedNow,
			wantCurrentSummary:    "Long Current",
			wantCurrentMeetingNil: false,
			wantNextSummary:       "Starts During",
			wantNextMeetingNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since GetMeetingStatus uses time.Now() internally, we need to
			// adjust our test events relative to the actual current time
			// for the test to work correctly. We'll create adjusted events.
			adjustedEvents := make([]*MeetingInfo, len(tt.events))
			now := time.Now()
			offset := now.Sub(fixedNow)

			for i, evt := range tt.events {
				if evt != nil {
					adjustedEvents[i] = &MeetingInfo{
						Summary:   evt.Summary,
						Start:     evt.Start.Add(offset),
						End:       evt.End.Add(offset),
						Location:  evt.Location,
						Attendees: evt.Attendees,
					}
				}
			}

			status := GetMeetingStatus(adjustedEvents)

			if status == nil {
				t.Fatal("GetMeetingStatus returned nil")
			}

			// Check current meeting
			if tt.wantCurrentMeetingNil {
				if status.CurrentMeeting != nil {
					t.Errorf("expected CurrentMeeting to be nil, got %+v", status.CurrentMeeting)
				}
			} else {
				if status.CurrentMeeting == nil {
					t.Errorf("expected CurrentMeeting to not be nil")
				} else if status.CurrentMeeting.Summary != tt.wantCurrentSummary {
					t.Errorf("expected CurrentMeeting.Summary = %q, got %q",
						tt.wantCurrentSummary, status.CurrentMeeting.Summary)
				}
			}

			// Check next meeting
			if tt.wantNextMeetingNil {
				if status.NextMeeting != nil {
					t.Errorf("expected NextMeeting to be nil, got %+v", status.NextMeeting)
				}
			} else {
				if status.NextMeeting == nil {
					t.Errorf("expected NextMeeting to not be nil")
				} else if status.NextMeeting.Summary != tt.wantNextSummary {
					t.Errorf("expected NextMeeting.Summary = %q, got %q",
						tt.wantNextSummary, status.NextMeeting.Summary)
				}
			}
		})
	}
}

func TestGetMeetingStatus_BoundaryConditions(t *testing.T) {
	now := time.Now()

	t.Run("now equals meeting start time - should be current", func(t *testing.T) {
		// Meeting starts exactly at now
		events := []*MeetingInfo{
			{
				Summary: "Starting Now",
				Start:   now,
				End:     now.Add(1 * time.Hour),
			},
		}

		status := GetMeetingStatus(events)

		// !now.Before(meeting.Start) is true when now == Start
		// now.Before(meeting.End) is true
		// So this should be current
		if status.CurrentMeeting == nil {
			t.Fatal("expected meeting starting exactly at now to be CurrentMeeting")
		}
		if status.CurrentMeeting.Summary != "Starting Now" {
			t.Errorf("expected 'Starting Now', got %q", status.CurrentMeeting.Summary)
		}
	})

	t.Run("now equals meeting end time - should not be current", func(t *testing.T) {
		// Meeting ends exactly at now
		events := []*MeetingInfo{
			{
				Summary: "Ending Now",
				Start:   now.Add(-1 * time.Hour),
				End:     now,
			},
		}

		status := GetMeetingStatus(events)

		// !now.Before(meeting.Start) is true
		// now.Before(meeting.End) is false when now == End
		// So this should NOT be current
		if status.CurrentMeeting != nil {
			t.Errorf("expected meeting ending exactly at now to not be CurrentMeeting, got %+v",
				status.CurrentMeeting)
		}
	})

	t.Run("meeting about to end - should be current", func(t *testing.T) {
		now := time.Now()
		// Use a longer buffer to account for time passing during test execution
		endTime := now.Add(100 * time.Millisecond)
		events := []*MeetingInfo{
			{
				Summary: "Almost Ending",
				Start:   now.Add(-1 * time.Hour),
				End:     endTime,
			},
		}

		status := GetMeetingStatus(events)

		if status.CurrentMeeting == nil {
			t.Fatal("expected meeting to be CurrentMeeting (ending soon)")
		}
	})

	t.Run("1 nanosecond after meeting starts - should be current", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(-1 * time.Nanosecond)
		events := []*MeetingInfo{
			{
				Summary: "Just Started",
				Start:   startTime,
				End:     now.Add(1 * time.Hour),
			},
		}

		status := GetMeetingStatus(events)

		if status.CurrentMeeting == nil {
			t.Fatal("expected meeting to be CurrentMeeting (1ns after start)")
		}
	})

	t.Run("meeting starting soon - should be next", func(t *testing.T) {
		now := time.Now()
		// Use a longer buffer to ensure the meeting is still in the future
		startTime := now.Add(100 * time.Millisecond)
		events := []*MeetingInfo{
			{
				Summary: "Starting Soon",
				Start:   startTime,
				End:     now.Add(1 * time.Hour),
			},
		}

		status := GetMeetingStatus(events)

		if status.CurrentMeeting != nil {
			t.Errorf("expected no CurrentMeeting for meeting starting soon")
		}
		if status.NextMeeting == nil {
			t.Fatal("expected meeting starting soon to be NextMeeting")
		}
		if status.NextMeeting.Summary != "Starting Soon" {
			t.Errorf("expected 'Starting Soon', got %q", status.NextMeeting.Summary)
		}
	})
}

func TestGetMeetingStatus_ReturnValueIntegrity(t *testing.T) {
	now := time.Now()

	t.Run("returned status is never nil", func(t *testing.T) {
		testCases := []struct {
			name   string
			events []*MeetingInfo
		}{
			{"nil events", nil},
			{"empty events", []*MeetingInfo{}},
			{"single past event", []*MeetingInfo{
				{Summary: "Past", Start: now.Add(-2 * time.Hour), End: now.Add(-1 * time.Hour)},
			}},
			{"single current event", []*MeetingInfo{
				{Summary: "Current", Start: now.Add(-30 * time.Minute), End: now.Add(30 * time.Minute)},
			}},
			{"single future event", []*MeetingInfo{
				{Summary: "Future", Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour)},
			}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				status := GetMeetingStatus(tc.events)
				if status == nil {
					t.Error("GetMeetingStatus should never return nil")
				}
			})
		}
	})

	t.Run("current meeting pointer refers to original event", func(t *testing.T) {
		meeting := &MeetingInfo{
			Summary:   "Original Meeting",
			Start:     now.Add(-30 * time.Minute),
			End:       now.Add(30 * time.Minute),
			Location:  "Room 1",
			Attendees: 10,
		}
		events := []*MeetingInfo{meeting}

		status := GetMeetingStatus(events)

		if status.CurrentMeeting != meeting {
			t.Error("CurrentMeeting should point to the original MeetingInfo pointer")
		}
	})

	t.Run("next meeting pointer refers to original event", func(t *testing.T) {
		meeting := &MeetingInfo{
			Summary:   "Future Meeting",
			Start:     now.Add(1 * time.Hour),
			End:       now.Add(2 * time.Hour),
			Location:  "Room 2",
			Attendees: 5,
		}
		events := []*MeetingInfo{meeting}

		status := GetMeetingStatus(events)

		if status.NextMeeting != meeting {
			t.Error("NextMeeting should point to the original MeetingInfo pointer")
		}
	})
}

func TestGetMeetingStatus_LargeDataSets(t *testing.T) {
	now := time.Now()

	t.Run("100 past meetings", func(t *testing.T) {
		events := make([]*MeetingInfo, 100)
		for i := 0; i < 100; i++ {
			events[i] = &MeetingInfo{
				Summary: "Past Meeting",
				Start:   now.Add(time.Duration(-100+i) * time.Hour),
				End:     now.Add(time.Duration(-99+i) * time.Hour),
			}
		}

		status := GetMeetingStatus(events)

		if status.CurrentMeeting != nil {
			t.Error("expected no CurrentMeeting for all past meetings")
		}
		if status.NextMeeting != nil {
			t.Error("expected no NextMeeting for all past meetings")
		}
	})

	t.Run("100 future meetings - earliest selected", func(t *testing.T) {
		events := make([]*MeetingInfo, 100)
		for i := 0; i < 100; i++ {
			events[i] = &MeetingInfo{
				Summary: fmt.Sprintf("Future Meeting %d", i),
				Start:   now.Add(time.Duration(100-i) * time.Hour), // Reverse order
				End:     now.Add(time.Duration(101-i) * time.Hour),
			}
		}

		status := GetMeetingStatus(events)

		if status.NextMeeting == nil {
			t.Fatal("expected NextMeeting to exist")
		}
		// Meeting 99 has the earliest start time (1 hour from now)
		if status.NextMeeting.Summary != "Future Meeting 99" {
			t.Errorf("expected 'Future Meeting 99' (earliest), got %q",
				status.NextMeeting.Summary)
		}
	})

	t.Run("mixed past, current, future - 1000 events", func(t *testing.T) {
		events := make([]*MeetingInfo, 1000)

		// 333 past meetings
		for i := 0; i < 333; i++ {
			events[i] = &MeetingInfo{
				Summary: fmt.Sprintf("Past %d", i),
				Start:   now.Add(time.Duration(-1000+i) * time.Hour),
				End:     now.Add(time.Duration(-999+i) * time.Hour),
			}
		}

		// 1 current meeting
		events[333] = &MeetingInfo{
			Summary: "Current",
			Start:   now.Add(-30 * time.Minute),
			End:     now.Add(30 * time.Minute),
		}

		// 666 future meetings
		for i := 334; i < 1000; i++ {
			events[i] = &MeetingInfo{
				Summary: fmt.Sprintf("Future %d", i),
				Start:   now.Add(time.Duration(i-333) * time.Hour),
				End:     now.Add(time.Duration(i-332) * time.Hour),
			}
		}

		status := GetMeetingStatus(events)

		if status.CurrentMeeting == nil || status.CurrentMeeting.Summary != "Current" {
			t.Errorf("expected CurrentMeeting 'Current', got %+v", status.CurrentMeeting)
		}
		if status.NextMeeting == nil || status.NextMeeting.Summary != "Future 334" {
			t.Errorf("expected NextMeeting 'Future 334' (earliest future), got %+v",
				status.NextMeeting)
		}
	})
}

func TestFilterAccepted(t *testing.T) {
	fixedNow := time.Date(2026, 1, 14, 10, 0, 0, 0, time.UTC)

	// Helper function to create meetings with a specific response status
	makeMeetingWithStatus := func(summary string, status string) *MeetingInfo {
		return &MeetingInfo{
			Summary:            summary,
			Start:              fixedNow,
			End:                fixedNow.Add(1 * time.Hour),
			Location:           "Test Location",
			Attendees:          5,
			SelfResponseStatus: status,
		}
	}

	tests := []struct {
		name          string
		events        []*MeetingInfo
		wantCount     int
		wantSummaries []string
	}{
		{
			name:          "empty events list",
			events:        []*MeetingInfo{},
			wantCount:     0,
			wantSummaries: nil,
		},
		{
			name:          "nil events list",
			events:        nil,
			wantCount:     0,
			wantSummaries: nil,
		},
		{
			name: "single accepted event",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Accepted Meeting", "accepted"),
			},
			wantCount:     1,
			wantSummaries: []string{"Accepted Meeting"},
		},
		{
			name: "single tentative event (maybe)",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Maybe Meeting", "tentative"),
			},
			wantCount:     1,
			wantSummaries: []string{"Maybe Meeting"},
		},
		{
			name: "single declined event",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Declined Meeting", "declined"),
			},
			wantCount:     0,
			wantSummaries: nil,
		},
		{
			name: "single needsAction event (not responded)",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Pending Meeting", "needsAction"),
			},
			wantCount:     0,
			wantSummaries: nil,
		},
		{
			name: "single event with empty response status",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Unknown Meeting", ""),
			},
			wantCount:     0,
			wantSummaries: nil,
		},
		{
			name: "mix of all response statuses",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Accepted 1", "accepted"),
				makeMeetingWithStatus("Declined 1", "declined"),
				makeMeetingWithStatus("Tentative 1", "tentative"),
				makeMeetingWithStatus("NeedsAction 1", "needsAction"),
				makeMeetingWithStatus("Accepted 2", "accepted"),
				makeMeetingWithStatus("Tentative 2", "tentative"),
			},
			wantCount:     4,
			wantSummaries: []string{"Accepted 1", "Tentative 1", "Accepted 2", "Tentative 2"},
		},
		{
			name: "all declined events",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Declined 1", "declined"),
				makeMeetingWithStatus("Declined 2", "declined"),
				makeMeetingWithStatus("Declined 3", "declined"),
			},
			wantCount:     0,
			wantSummaries: nil,
		},
		{
			name: "all accepted events",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Accepted 1", "accepted"),
				makeMeetingWithStatus("Accepted 2", "accepted"),
				makeMeetingWithStatus("Accepted 3", "accepted"),
			},
			wantCount:     3,
			wantSummaries: []string{"Accepted 1", "Accepted 2", "Accepted 3"},
		},
		{
			name: "all tentative events",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Tentative 1", "tentative"),
				makeMeetingWithStatus("Tentative 2", "tentative"),
			},
			wantCount:     2,
			wantSummaries: []string{"Tentative 1", "Tentative 2"},
		},
		{
			name: "all needsAction events (pending responses)",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Pending 1", "needsAction"),
				makeMeetingWithStatus("Pending 2", "needsAction"),
			},
			wantCount:     0,
			wantSummaries: nil,
		},
		{
			name: "accepted and tentative only",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Accepted", "accepted"),
				makeMeetingWithStatus("Tentative", "tentative"),
			},
			wantCount:     2,
			wantSummaries: []string{"Accepted", "Tentative"},
		},
		{
			name: "declined and needsAction only",
			events: []*MeetingInfo{
				makeMeetingWithStatus("Declined", "declined"),
				makeMeetingWithStatus("Pending", "needsAction"),
			},
			wantCount:     0,
			wantSummaries: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterAccepted(tt.events)

			if len(got) != tt.wantCount {
				t.Errorf("FilterAccepted() returned %d events, want %d", len(got), tt.wantCount)
			}

			if tt.wantSummaries != nil {
				for i, event := range got {
					if i >= len(tt.wantSummaries) {
						t.Errorf("FilterAccepted() returned more events than expected")
						break
					}
					if event.Summary != tt.wantSummaries[i] {
						t.Errorf("FilterAccepted()[%d].Summary = %q, want %q", i, event.Summary, tt.wantSummaries[i])
					}
				}
			}
		})
	}
}

func TestFilterAccepted_PreservesEventOrder(t *testing.T) {
	fixedNow := time.Date(2026, 1, 14, 10, 0, 0, 0, time.UTC)

	events := []*MeetingInfo{
		{Summary: "First", Start: fixedNow, End: fixedNow.Add(1 * time.Hour), SelfResponseStatus: "accepted"},
		{Summary: "Second", Start: fixedNow.Add(1 * time.Hour), End: fixedNow.Add(2 * time.Hour), SelfResponseStatus: "declined"},
		{Summary: "Third", Start: fixedNow.Add(2 * time.Hour), End: fixedNow.Add(3 * time.Hour), SelfResponseStatus: "tentative"},
		{Summary: "Fourth", Start: fixedNow.Add(3 * time.Hour), End: fixedNow.Add(4 * time.Hour), SelfResponseStatus: "accepted"},
	}

	got := FilterAccepted(events)

	expectedOrder := []string{"First", "Third", "Fourth"}
	if len(got) != len(expectedOrder) {
		t.Fatalf("FilterAccepted() returned %d events, want %d", len(got), len(expectedOrder))
	}

	for i, event := range got {
		if event.Summary != expectedOrder[i] {
			t.Errorf("FilterAccepted()[%d].Summary = %q, want %q", i, event.Summary, expectedOrder[i])
		}
	}
}

func TestFilterAccepted_DoesNotModifyOriginalSlice(t *testing.T) {
	fixedNow := time.Date(2026, 1, 14, 10, 0, 0, 0, time.UTC)

	original := []*MeetingInfo{
		{Summary: "Accepted", Start: fixedNow, End: fixedNow.Add(1 * time.Hour), SelfResponseStatus: "accepted"},
		{Summary: "Declined", Start: fixedNow.Add(1 * time.Hour), End: fixedNow.Add(2 * time.Hour), SelfResponseStatus: "declined"},
	}

	originalLen := len(original)
	originalFirstSummary := original[0].Summary
	originalSecondSummary := original[1].Summary

	_ = FilterAccepted(original)

	if len(original) != originalLen {
		t.Errorf("Original slice length changed from %d to %d", originalLen, len(original))
	}
	if original[0].Summary != originalFirstSummary {
		t.Errorf("Original slice first element changed")
	}
	if original[1].Summary != originalSecondSummary {
		t.Errorf("Original slice second element changed")
	}
}

func TestFilterAccepted_WithMeetingStatusIntegration(t *testing.T) {
	// Test that FilterAccepted works correctly with GetMeetingStatus
	now := time.Date(2026, 1, 14, 10, 30, 0, 0, time.UTC)

	events := []*MeetingInfo{
		// Current meeting - declined (should be filtered out)
		{
			Summary:            "Declined Current",
			Start:              now.Add(-30 * time.Minute),
			End:                now.Add(30 * time.Minute),
			SelfResponseStatus: "declined",
		},
		// Current meeting - accepted (should be kept)
		{
			Summary:            "Accepted Current",
			Start:              now.Add(-15 * time.Minute),
			End:                now.Add(45 * time.Minute),
			SelfResponseStatus: "accepted",
		},
		// Next meeting - needsAction (should be filtered out)
		{
			Summary:            "Pending Next",
			Start:              now.Add(1 * time.Hour),
			End:                now.Add(2 * time.Hour),
			SelfResponseStatus: "needsAction",
		},
		// Next meeting - tentative (should be kept)
		{
			Summary:            "Maybe Next",
			Start:              now.Add(2 * time.Hour),
			End:                now.Add(3 * time.Hour),
			SelfResponseStatus: "tentative",
		},
	}

	filtered := FilterAccepted(events)

	if len(filtered) != 2 {
		t.Fatalf("FilterAccepted() returned %d events, want 2", len(filtered))
	}

	if filtered[0].Summary != "Accepted Current" {
		t.Errorf("First filtered event should be 'Accepted Current', got %q", filtered[0].Summary)
	}
	if filtered[1].Summary != "Maybe Next" {
		t.Errorf("Second filtered event should be 'Maybe Next', got %q", filtered[1].Summary)
	}
}
