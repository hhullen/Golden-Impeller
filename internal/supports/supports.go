package supports

import (
	"context"
	"fmt"
	"strings"
	"time"
)

var (
	dateFormats = []string{
		"2006-01-02",
		"2006/01/02",
		"2006.01.02",
		"02-01-2006",
		"02.01.2006",
		"02/01/2006",
	}
)

func ParseDate(s string) (time.Time, error) {
	if strings.ToLower(s) == "now" {
		return time.Now(), nil
	}

	var monthsAgo int
	_, err := fmt.Sscanf(s, "%d", &monthsAgo)
	if err == nil && monthsAgo < 0 {
		return time.Now().AddDate(0, monthsAgo, 0), nil
	}

	for _, f := range dateFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("incorrect date format: %s", s)
}

func WaitFor(ctx context.Context, waitOnPanic time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(waitOnPanic):
	}
}
