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
