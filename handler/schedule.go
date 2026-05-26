package handler

import (
	"audyn/config"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// parseHHMM parses a "HH:MM" string and returns (hour, minute, error).
func parseHHMM(s string) (int, int, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time %q: expected HH:MM", s)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, 0, fmt.Errorf("invalid hour in %q", s)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("invalid minute in %q", s)
	}
	return h, m, nil
}

// isInWindow reports whether t falls inside the [startH:startM, endH:endM) window
func isInWindow(t time.Time, startH, startM, endH, endM int) bool {
	nowMins := t.Hour()*60 + t.Minute()
	startMins := startH*60 + startM
	endMins := endH*60 + endM

	switch {
	case startMins == endMins:
		return false // zero-length window → never allowed
	case startMins < endMins:
		// Normal window e.g. 02:00–06:00
		return nowMins >= startMins && nowMins < endMins
	default:
		// Overnight window e.g. 22:00–06:00
		return nowMins >= startMins || nowMins < endMins
	}
}

func WaitForDownloadWindow(cfg config.Config) {
	if cfg.ScheduleStart == "" || cfg.ScheduleEnd == "" {
		return
	}

	startH, startM, err := parseHHMM(cfg.ScheduleStart)
	if err != nil {
		slog.Warn("AUDYN_SCHEDULE_START invalid, ignoring schedule", "err", err)
		return
	}
	endH, endM, err := parseHHMM(cfg.ScheduleEnd)
	if err != nil {
		slog.Warn("AUDYN_SCHEDULE_END invalid, ignoring schedule", "err", err)
		return
	}

	now := time.Now()
	if isInWindow(now, startH, startM, endH, endM) {
		return // already inside the window
	}

	// Calculate how long until the window opens next.
	next := time.Date(now.Year(), now.Month(), now.Day(), startH, startM, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	wait := next.Sub(now)

	slog.Info("Download window not open, waiting",
		"window_start", cfg.ScheduleStart,
		"window_end", cfg.ScheduleEnd,
		"resumes_at", next.Format("2006-01-02 15:04:05"),
		"wait", wait.Round(time.Second),
	)
	time.Sleep(wait)
}
