package logger

import (
	"io"
	"log"
	"sync"
)

type message struct {
	template string
	args     []any
}

type Logger struct {
	log.Logger
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

	l.SetOutput(out)
	l.SetPrefix(prefix + " ")
	l.SetFlags(log.Ltime + log.Ldate)

	go l.listenCh(l.infoCh, l.Printf)
	go l.listenCh(l.errCh, l.Printf)
	go l.listenCh(l.fatalCh, l.Panicf)

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
	l.infoCh <- message{template: template, args: args}
}

func (l *Logger) Errorf(template string, args ...any) {
	l.wg.Add(1)
	l.errCh <- message{template: template, args: args}
}

func (l *Logger) Fatalf(template string, args ...any) {
	l.wg.Add(1)
	l.fatalCh <- message{template: template, args: args}
}

func (l *Logger) listenCh(ch chan message, fn func(string, ...any)) {
	for msg := range ch {
		fn(msg.template, msg.args...)
		l.wg.Done()
	}
}
