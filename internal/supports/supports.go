package supports

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

func CloseIfMaybeClosed[Type any](ch chan Type) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v", p)
		}
	}()

	select {
	case <-ch:
	default:
	}

	close(ch)

	return
}

func SendOrSkipIfMaybeClosed[Type any](ch chan<- Type, v Type) (err error) {
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

func SendIfMaybeClosed[Type any](ch chan<- Type, v Type) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v", p)
		}
	}()

	ch <- v

	return
}

func IsInContainer() bool {
	return os.Getenv("RUNNING_IN_CONTAINER") == "true"
}

func ReadSecret(path string) string {
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		panic(err)
	}

	secret := ""
	fmt.Fscan(f, &secret)

	return string(secret)
}

func MakeKVMessagesJSON(kvs ...any) (bytes []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("failed WriteInTopicKV: %v", p)
		}
	}()

	msgs := map[string]any{}
	for i := 0; i < len(kvs)-1; i += 2 {
		key := fmt.Sprint(kvs[i])
		value := kvs[i+1]
		msgs[key] = value
	}

	bytes, err = json.Marshal(msgs)
	return
}
