package logger

import (
	"log"
)

type message struct {
	template string
	args     []any
}

type Logger struct {
	infoCh  chan message
	errCh   chan message
	fatalCh chan message
}

func NewLogger() *Logger {

	l := &Logger{
		infoCh:  make(chan message, 1000),
		errCh:   make(chan message, 1000),
		fatalCh: make(chan message, 1000),
	}

	go listenCh(l.infoCh, log.Printf)
	go listenCh(l.errCh, log.Fatalf)
	go listenCh(l.fatalCh, log.Panicf)

	return l
}

func (l *Logger) Stop() {
	close(l.infoCh)
	close(l.errCh)
	close(l.fatalCh)
}

func (l *Logger) Infof(template string, args ...any) {
	l.infoCh <- message{template: template, args: args}
}

func (l *Logger) Errorf(template string, args ...any) {
	l.errCh <- message{template: template, args: args}
}

func (l *Logger) Fatalf(template string, args ...any) {
	l.fatalCh <- message{template: template, args: args}
}

func listenCh(ch chan message, fn func(string, ...any)) {
	for msg := range ch {
		fn(msg.template, msg.args...)
	}
}
