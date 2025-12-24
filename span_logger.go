package loggerhandler

import (
	"fmt"
	"log/slog"
	"time"
)

type SpanLogger struct {
	id            string
	timeDuration  time.Duration
	tags          []string
	bufferSize    int
	buffer        []slog.Record
	loggerHandler *LoggerHandler
	logLevel      slog.Level
}

func NewSpanLogger(id string, duration time.Duration, tags []string, bufferSize int, loggerHandler *LoggerHandler, level slog.Level) *SpanLogger {
	return &SpanLogger{
		id:            id,
		timeDuration:  duration,
		tags:          tags,
		bufferSize:    bufferSize,
		loggerHandler: loggerHandler,
		logLevel:      level,
	}
}

func (sl *SpanLogger) GetID() string {
	return sl.id
}

func (sl *SpanLogger) GetDuration() time.Duration {
	return sl.timeDuration
}

func (sl *SpanLogger) GetTags() []string {
	return sl.tags
}

func (sl *SpanLogger) GetBufferSize() int {
	return sl.bufferSize
}

func (sl *SpanLogger) GetLoggerHandler() *LoggerHandler {
	return sl.loggerHandler
}

func (sl *SpanLogger) GetLogLevel() slog.Level {
	return sl.logLevel
}

func (sl *SpanLogger) sendLogCmd(op OpType, err error) {
	lc := LogCommand{
		Op:      op,
		SpanID:  sl.id,
		Records: sl.buffer,
		Err:     err,
	}
	// aggiungo il comando alla coda del LoggerHandler
	sl.loggerHandler.AppendLogCommand(lc)
	// svuoto il buffer
	sl.buffer = []slog.Record{}
}

func (sl *SpanLogger) Debug(msg string, attrs ...slog.Attr) {
	lvl := slog.LevelDebug

	// Creo il record
	record := slog.NewRecord(time.Now(), lvl, msg, 0)
	record.AddAttrs(attrs...)

	// Aggiungo il record al buffer
	sl.buffer = append(sl.buffer, record)

	if lvl >= sl.logLevel {
		sl.sendLogCmd(OpLog, nil)
	}
}

func (sl *SpanLogger) Info(msg string, attrs ...slog.Attr) {
	lvl := slog.LevelInfo

	// Creo il record
	record := slog.NewRecord(time.Now(), lvl, msg, 0)
	record.AddAttrs(attrs...)

	// Aggiungo il record al buffer
	sl.buffer = append(sl.buffer, record)

	if lvl >= sl.logLevel {
		sl.sendLogCmd(OpLog, nil)
	}
}

func (sl *SpanLogger) Warn(msg string, attrs ...slog.Attr) {
	lvl := slog.LevelWarn

	// Creo il record
	record := slog.NewRecord(time.Now(), lvl, msg, 0)
	record.AddAttrs(attrs...)

	// Aggiungo il record al buffer
	sl.buffer = append(sl.buffer, record)

	if lvl >= sl.logLevel {
		sl.sendLogCmd(OpLog, nil)
	}
}

func (sl *SpanLogger) Error(msg string, attrs ...slog.Attr) {
	lvl := slog.LevelError

	// Creo il record
	record := slog.NewRecord(time.Now(), lvl, msg, 0)
	record.AddAttrs(attrs...)

	// Aggiungo il record al buffer
	sl.buffer = append(sl.buffer, record)

	sl.sendLogCmd(OpReleaseFailure, fmt.Errorf("%s", msg))
}

func (sl *SpanLogger) ReleaseSuccess() {
	sl.sendLogCmd(OpReleaseSuccess, nil)
}
