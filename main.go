package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"next-meeting/auth"
	"next-meeting/cache"
	"next-meeting/calendar"
)

func main() {
	clearCredentials := flag.Bool("clear", false, "Clear credentials")
	clearCache := flag.Bool("clear-cache", false, "Clear the calendar cache")
	login := flag.Bool("login", false, "Login to Google Calendar")
	flag.Parse()

	ctx := context.Background()

	// Handle --clear-cache flag
	if *clearCache {
		if err := cache.Clear(); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Cache cleared (%s)\n", cache.GetPath())
		return
	}

	// Handle --clear flag
	if *clearCredentials {
		if err := auth.ClearToken(); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing credentials: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Credentials cleared")
		return
	}

	// Handle --login flag
	if *login {
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

	// Try to read events from cache first
	events := cache.Read()

	// If no valid cache, fetch from API
	if events == nil {
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

		// Get events from API
		events, err = calSvc.GetTodayEvents(ctx)
		if err != nil {
			if isNetworkError(err) {
				fmt.Println("📡 Calendar Offline")
				return
			}
			fmt.Fprintf(os.Stderr, "Error getting events: %v\n", err)
			os.Exit(1)
		}

		// Cache the events
		if err := cache.Write(events); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cache results: %v\n", err)
		}
	}

	// Calculate current/next meeting status from events (always fresh calculation)
	status := calendar.GetMeetingStatus(events)

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

// isNetworkError checks if an error is related to network connectivity issues
func isNetworkError(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	return false
}
