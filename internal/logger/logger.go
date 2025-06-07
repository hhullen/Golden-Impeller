package logger

import (
	"io"
	"log"
	"sync"
)

const (
	infoTag  = "INFO "
	errorTag = "ERROR "
	fatalTag = "FATAL "
)

type message struct {
	template string
	args     []any
}

type Logger struct {
	l       log.Logger
	infoCh  chan message
	errCh   chan message
	fatalCh chan message

	wg sync.WaitGroup
}

func NewLogger(out io.Writer, prefix string) *Logger {
	l := &Logger{
		infoCh:  make(chan message, 1000),
		errCh:   make(chan message, 1000),
		fatalCh: make(chan message, 1000),
	}

	l.l.SetOutput(out)
	l.l.SetPrefix(prefix + " ")
	l.l.SetFlags(log.Ltime + log.Ldate)

	go l.listenCh(l.infoCh, l.l.Printf)
	go l.listenCh(l.errCh, l.l.Printf)
	go l.listenCh(l.fatalCh, l.l.Printf)

	return l
}

func (l *Logger) Stop() {
	l.wg.Wait()
	close(l.infoCh)
	close(l.errCh)
	close(l.fatalCh)
}

func (l *Logger) Infof(template string, args ...any) {
	l.wg.Add(1)
	l.infoCh <- message{template: infoTag + template, args: args}
}

func (l *Logger) Errorf(template string, args ...any) {
	l.wg.Add(1)
	l.errCh <- message{template: errorTag + template, args: args}
}

func (l *Logger) Fatalf(template string, args ...any) {
	l.wg.Add(1)
	l.fatalCh <- message{template: fatalTag + template, args: args}
}

func (l *Logger) listenCh(ch chan message, fn func(string, ...any)) {
	for msg := range ch {
		fn(msg.template, msg.args...)
		l.wg.Done()
	}
}
