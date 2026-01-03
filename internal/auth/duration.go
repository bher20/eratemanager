package auth

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// ParseExpirationDuration parses a duration string or custom date and returns an expiration time.
// Supported formats:
//   - "never" or "" (empty) - returns nil (no expiration)
//   - "30d" - 30 days from now
//   - "7d" - 7 days from now
//   - "24h" - 24 hours from now
//   - "1h" - 1 hour from now
//   - Any valid Go duration like "30m", "2h30m", etc.
//   - "mm/dd/yyyy" - Custom date (e.g., "12/25/2026")
//   - "mm/dd/yyyy HH:MM" - Custom date with time (e.g., "12/25/2026 14:30")
//
// Examples:
//   ParseExpirationDuration("never") -> nil
//   ParseExpirationDuration("30d") -> time.Now().Add(30 * 24 * time.Hour)
//   ParseExpirationDuration("12/25/2026") -> Dec 25, 2026 at 00:00:00 UTC
//   ParseExpirationDuration("12/25/2026 14:30") -> Dec 25, 2026 at 14:30:00 UTC
func ParseExpirationDuration(expiresIn string) (*time.Time, error) {
	if expiresIn == "" || expiresIn == "never" {
		return nil, nil
	}

	// Try to parse as standard Go duration first
	if dur, err := time.ParseDuration(expiresIn); err == nil {
		t := time.Now().Add(dur)
		return &t, nil
	}

	// Try to parse custom date formats: mm/dd/yyyy or mm/dd/yyyy HH:MM
	dateFormats := []string{
		"01/02/2006 15:04", // mm/dd/yyyy HH:MM
		"01/02/2006",       // mm/dd/yyyy
	}

	for _, format := range dateFormats {
		if t, err := time.Parse(format, expiresIn); err == nil {
			// Ensure the date is in the future
			if t.Before(time.Now()) {
				return nil, fmt.Errorf("expiration date must be in the future: %s", expiresIn)
			}
			return &t, nil
		}
	}

	// Try to parse custom formats like "30d", "7d", etc.
	re := regexp.MustCompile(`^(\d+)([dwh])$`)
	matches := re.FindStringSubmatch(expiresIn)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid expiration format: %s (use 'never', '30d', '7d', '24h', '12/25/2026', or any Go duration like '30m')", expiresIn)
	}

	num, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid number in expiration: %s", expiresIn)
	}

	var dur time.Duration
	unit := matches[2]
	switch unit {
	case "d":
		dur = time.Duration(num) * 24 * time.Hour
	case "w":
		dur = time.Duration(num) * 7 * 24 * time.Hour
	case "h":
		dur = time.Duration(num) * time.Hour
	default:
		return nil, fmt.Errorf("unknown unit in expiration: %s", unit)
	}

	t := time.Now().Add(dur)
	return &t, nil
}
