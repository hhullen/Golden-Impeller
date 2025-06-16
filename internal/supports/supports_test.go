package supports

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseDate(t *testing.T) {
	t.Parallel()

	t.Run("now", func(t *testing.T) {
		t.Parallel()
		tm, err := ParseDate("now")
		require.NotNil(t, tm)
		require.Nil(t, err)
	})

	t.Run("-months", func(t *testing.T) {
		t.Parallel()
		tm, err := ParseDate("-13")

		tb := time.Now().AddDate(0, 12, 0)

		require.True(t, tm.Before(tb))
		require.NotNil(t, tm)
		require.Nil(t, err)
	})

	t.Run("parse formats", func(t *testing.T) {
		t.Parallel()

		for _, tf := range dateFormats {
			tm, err := ParseDate(tf)
			require.NotNil(t, tm)
			require.Nil(t, err)

		}
	})

	t.Run("incorrect value", func(t *testing.T) {
		t.Parallel()

		tm, err := ParseDate("10--5")
		require.NotNil(t, err)
		require.Equal(t, tm, time.Time{})
	})

	t.Run("incorrect date", func(t *testing.T) {
		t.Parallel()

		tm, err := ParseDate("1998-20.02")
		require.NotNil(t, err)
		require.Equal(t, tm, time.Time{})
	})
}

func TestWaitFor(t *testing.T) {
	t.Parallel()

	t.Run("ctx done", func(t *testing.T) {
		t.Parallel()

		start := time.Now()
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)
		WaitFor(ctx, time.Second*10)

		require.True(t, start.Before(time.Now().Add(time.Millisecond*200)))
	})

	t.Run("ctx done", func(t *testing.T) {
		t.Parallel()

		start := time.Now()
		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
		WaitFor(ctx, time.Millisecond*100)

		require.True(t, start.Before(time.Now().Add(time.Millisecond*200)))
	})
}

func TestCast(t *testing.T) {
	t.Parallel()

	t.Run("CastToFloat64", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			require.Equal(t, CastToFloat64(5.22), float64(5.22))
		})

		require.NotPanics(t, func() {
			require.Equal(t, CastToFloat64(522), float64(522))
		})

		require.Panics(t, func() {
			CastToFloat64("text")
		})
	})

	t.Run("CastToInt64", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			require.Equal(t, CastToInt64(5.22), int64(5))
		})

		require.NotPanics(t, func() {
			require.Equal(t, CastToInt64(359), int64(359))
		})

		require.Panics(t, func() {
			CastToInt64("text")
		})
	})
}

func TestCloseIfMaybeClosed(t *testing.T) {
	t.Parallel()

	t.Run("close closed", func(t *testing.T) {
		t.Parallel()
		ch := make(chan int)
		close(ch)
		err := CloseIfMaybeClosed(ch)
		require.NotNil(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ch := make(chan int)

		err := CloseIfMaybeClosed(ch)
		require.Nil(t, err)
	})
}

func TestSendIfMaybeClosed(t *testing.T) {
	t.Parallel()

	t.Run("send to closed", func(t *testing.T) {
		t.Parallel()
		ch := make(chan int)
		close(ch)
		err := SendIfMaybeClosed(ch, 1)
		require.NotNil(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ch := make(chan int, 1)

		err := SendIfMaybeClosed(ch, 1)
		require.Nil(t, err)
	})
}

func TestSendOrSkipIfMaybeClosed(t *testing.T) {
	t.Parallel()

	t.Run("send to closed", func(t *testing.T) {
		t.Parallel()
		ch := make(chan int)
		close(ch)
		err := SendOrSkipIfMaybeClosed(ch, 1)
		require.NotNil(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ch := make(chan int)

		err := SendOrSkipIfMaybeClosed(ch, 1)
		require.Nil(t, err)
	})
}
