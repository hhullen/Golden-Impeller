package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	t.Parallel()

	t.Run("NewLogger", func(t *testing.T) {
		t.Parallel()

		l := NewLogger(os.Stdout, "test")
		require.NotNil(t, l)
	})

	t.Run("send", func(t *testing.T) {
		t.Parallel()

		l := NewLogger(os.Stdout, "test")

		require.NotPanics(t, func() {
			l.Infof("%s", "Hello MF!")
		})

		require.NotPanics(t, func() {
			l.Errorf("%s", "Hello MF!")
		})

		require.NotPanics(t, func() {
			l.Fatalf("%s", "Hello MF!")
		})

		require.NotNil(t, l)
	})

	t.Run("send to stopped", func(t *testing.T) {
		t.Parallel()

		l := NewLogger(os.Stdout, "test")
		l.Stop()

		require.Panics(t, func() {
			l.Infof("%s", "Hello MF!")
		})

		require.Panics(t, func() {
			l.Errorf("%s", "Hello MF!")
		})

		require.Panics(t, func() {
			l.Fatalf("%s", "Hello MF!")
		})

		require.NotNil(t, l)
	})
}
