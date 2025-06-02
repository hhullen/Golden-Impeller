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

func CastToFloat64(n any) float64 {
	if f, ok := n.(float64); ok {
		return f
	}

	if i, ok := n.(int); ok {
		return float64(i)
	}
	panic(fmt.Sprintf("impossible cast to number: %v", n))
}

func CastToInt64(n any) int64 {
	if i, ok := n.(int); ok {
		return int64(i)
	}

	if f, ok := n.(float64); ok {
		return int64(f)
	}
	panic(fmt.Sprintf("impossible cast to number: %v", n))
}

func CloseIfMaybeClosed[Type any](ch chan<- Type) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v", p)
		}
	}()
	close(ch)

	return
}

func SendIfMaybeClosed[Type any](ch chan<- Type, v Type) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v", p)
		}
	}()

	select {
	case ch <- v:
	default:
	}

	return
}
