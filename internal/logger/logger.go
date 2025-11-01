package logger

import (
	"fmt"
	"io"
	"log"
	"sync"
	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/service/trader"
	"trading_bot/internal/supports"
)

const (
	messagesBuffer = 1000
	LevelLabel     = "Level"
	infoTag        = "INFO"
	errorTag       = "ERROR"
	fatalTag       = "FATAL"
)

type Message struct {
	message string
	args    []any
}

type Logger struct {
	l       log.Logger
	infoCh  chan Message
	errCh   chan Message
	fatalCh chan Message
	history trader.IHistoryWriter

	wg sync.WaitGroup
}

func NewLogger(out io.Writer, prefix string, history trader.IHistoryWriter) *Logger {
	l := &Logger{
		infoCh:  make(chan Message, messagesBuffer),
		errCh:   make(chan Message, messagesBuffer),
		fatalCh: make(chan Message, messagesBuffer),
		history: history,
	}

	writer := l.l.Println

	if supports.IsInContainer() {
		writer = l.writeToHistory
	}

	l.l.SetOutput(out)
	l.l.SetPrefix(prefix + " ")
	l.l.SetFlags(log.Ltime + log.Ldate)

	go l.listenCh(l.infoCh, writer)
	go l.listenCh(l.errCh, writer)
	go l.listenCh(l.fatalCh, writer)

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
	l.infoCh <- Message{message: fmt.Sprintf(template, args...), args: []any{LevelLabel, infoTag}}
}

func (l *Logger) Errorf(template string, args ...any) {
	l.wg.Add(1)
	l.infoCh <- Message{message: fmt.Sprintf(template, args...), args: []any{LevelLabel, errorTag}}
}

func (l *Logger) Fatalf(template string, args ...any) {
	l.wg.Add(1)
	l.infoCh <- Message{message: fmt.Sprintf(template, args...), args: []any{LevelLabel, fatalTag}}
}

func (l *Logger) InfofKV(message string, argsKV ...any) {
	l.wg.Add(1)
	argsKV = append(argsKV, LevelLabel, infoTag)
	l.infoCh <- Message{message: message, args: argsKV}
}

func (l *Logger) ErrorfKV(message string, argsKV ...any) {
	l.wg.Add(1)
	argsKV = append(argsKV, LevelLabel, errorTag)
	l.errCh <- Message{message: message, args: argsKV}
}

func (l *Logger) FatalfKV(message string, argsKV ...any) {
	l.wg.Add(1)
	argsKV = append(argsKV, LevelLabel, fatalTag)
	l.fatalCh <- Message{message: message, args: argsKV}
}

func (l *Logger) listenCh(ch chan Message, fn func(...any)) {
	for msg := range ch {
		bytes, err := supports.MakeKVMessagesJSON(msg.args...)
		if err != nil {
			fn("failed making log", err.Error())
		} else {
			fn(msg.message, string(bytes))
		}
		l.wg.Done()
	}
}

func (l *Logger) writeToHistory(v ...any) {
	var err error
	if len(v) > 1 {
		err = l.history.WriteInTopicKV(ds.TopicLogs, ds.HistoryColMessage, v[0], ds.HistoryColDetails, v[1])
		l.l.Println(v...)
	} else if len(v) > 0 {
		err = l.history.WriteInTopicKV(ds.TopicLogs, ds.HistoryColMessage, v[0])
		l.l.Println(v...)
	}
	if err != nil {
		l.l.Printf("failed writing to topic logs { \"error\": \"%s\"}\n", err.Error())
	}
}
