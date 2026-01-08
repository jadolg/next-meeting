package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"next-meeting/auth"
	"next-meeting/calendar"
)

func main() {
	ctx := context.Background()

	// Handle --clear flag
	if len(os.Args) > 1 && os.Args[1] == "--clear" {
		if err := auth.ClearToken(); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing credentials: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Credentials cleared")
		return
	}

	// Handle --login flag
	if len(os.Args) > 1 && os.Args[1] == "--login" {
		if err := auth.Login(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error during login: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Logged in successfully")
		return
	}

	// Check if logged in
	if !auth.IsLoggedIn(ctx) {
		fmt.Println("🔒 Not logged in. Run with --login to authenticate.")
		os.Exit(1)
	}

	// Get authenticated client
	client, err := auth.GetClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting authenticated client: %v\n", err)
		os.Exit(1)
	}

	// Create calendar service
	calSvc, err := calendar.NewService(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating calendar service: %v\n", err)
		os.Exit(1)
	}

	// Get meeting status
	status, err := calSvc.GetMeetingStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting meeting status: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()

	// Build single-line output
	var parts []string

	// Current meeting (if any)
	if status.CurrentMeeting != nil {
		remaining := status.CurrentMeeting.End.Sub(now)
		if remaining < time.Minute {
			parts = append(parts, fmt.Sprintf("🔴 %s finishing now", status.CurrentMeeting.Summary))
		} else {
			parts = append(parts, fmt.Sprintf("🔴 %s (%s left)", status.CurrentMeeting.Summary, calendar.FormatDuration(remaining)))
		}
	}

	// Next meeting (if any)
	if status.NextMeeting != nil {
		startsIn := status.NextMeeting.Start.Sub(now)
		if startsIn < time.Minute {
			parts = append(parts, fmt.Sprintf("🕐 %s starting now", status.NextMeeting.Summary))
		} else {
			parts = append(parts, fmt.Sprintf("🕐 %s in %s", status.NextMeeting.Summary, calendar.FormatDuration(startsIn)))
		}
	}

	// Output
	if len(parts) == 0 {
		fmt.Println("📭 No meetings")
	} else {
		fmt.Println(strings.Join(parts, " │ "))
	}
}
