package loggerhandler

import (
	"errors"
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

// NewSpanLogger crea un nuovo SpanLogger.
// Cosa fa: costruisce e ritorna un nuovo oggetto SpanLogger con i parametri forniti.
// Parametri:
//   - id: identificatore univoco dello span
//   - duration: durata (timeout) dello span
//   - tags: lista di tag associati
//   - bufferSize: dimensione del buffer interno (non usata direttamente qui ma conservata)
//   - loggerHandler: riferimento al LoggerHandler che gestisce lo span
//   - level: livello minimo di log che provoca l'invio immediato
//
// Ritorna: puntatore a SpanLogger
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

// GetID restituisce l'id dello span.
// Parametri: nessuno
// Ritorna: string (id dello span)
func (sl *SpanLogger) GetID() string {
	return sl.id
}

// GetDuration restituisce la durata (timeout) dello span.
// Parametri: nessuno
// Ritorna: time.Duration
func (sl *SpanLogger) GetDuration() time.Duration {
	return sl.timeDuration
}

// GetTags restituisce i tag associati allo span.
// Parametri: nessuno
// Ritorna: slice di stringhe contenente i tag
func (sl *SpanLogger) GetTags() []string {
	return sl.tags
}

// GetBufferSize restituisce la dimensione del buffer interno dichiarata.
// Parametri: nessuno
// Ritorna: int
func (sl *SpanLogger) GetBufferSize() int {
	return sl.bufferSize
}

// GetLoggerHandler restituisce il riferimento al LoggerHandler associato.
// Parametri: nessuno
// Ritorna: *LoggerHandler
func (sl *SpanLogger) GetLoggerHandler() *LoggerHandler {
	return sl.loggerHandler
}

// GetLogLevel restituisce il livello di logging configurato per lo span.
// Parametri: nessuno
// Ritorna: slog.Level
func (sl *SpanLogger) GetLogLevel() slog.Level {
	return sl.logLevel
}

// sendLogCmd costruisce e invia un LogCommand al LoggerHandler.
// Cosa fa: crea un LogCommand con i record correnti, lo invia ad AppendCommand e pulisce il buffer.
// Parametri:
//   - op: tipo di operazione (OpLog, OpReleaseSuccess, OpReleaseFailure, OpTimeout)
//   - err: errore opzionale (usato per OpReleaseFailure/OpTimeout)
//
// Ritorna: nulla
func (sl *SpanLogger) sendLogCmd(op OpType, err error) {
	lc := LogCommand{
		Op:      op,
		SpanID:  sl.id,
		Records: sl.buffer,
		Err:     err,
	}
	// aggiungo il comando alla coda del LoggerHandler
	sl.loggerHandler.AppendCommand(lc)
	// svuoto il buffer
	sl.buffer = []slog.Record{}
}

// Debug aggiunge un record di debug al buffer e, se il livello lo richiede, invia il comando.
// Parametri:
//   - msg: messaggio di log
//   - attrs: attributi opzionali
//
// Ritorna: nulla
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

// Info aggiunge un record di info al buffer e, se il livello lo richiede, invia il comando.
// Parametri:
//   - msg: messaggio di log
//   - attrs: attributi opzionali
//
// Ritorna: nulla
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

// Warn aggiunge un record di warning al buffer e, se il livello lo richiede, invia il comando.
// Parametri:
//   - msg: messaggio di log
//   - attrs: attributi opzionali
//
// Ritorna: nulla
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

// Error aggiunge un record di errore al buffer e invia immediatamente un OpReleaseFailure.
// Parametri:
//   - msg: messaggio di errore
//   - attrs: attributi opzionali (non usati qui ma accettati per compatibilità)
//
// Ritorna: nulla
func (sl *SpanLogger) Error(msg string, attrs ...slog.Attr) {
	lvl := slog.LevelError

	// Creo il record
	record := slog.NewRecord(time.Now(), lvl, msg, 0)
	record.AddAttrs(attrs...)

	// Aggiungo il record al buffer
	sl.buffer = append(sl.buffer, record)

	sl.sendLogCmd(OpReleaseFailure, fmt.Errorf("%s", msg))
}

// ReleaseSuccess invia un comando di rilascio con successo (OpReleaseSuccess).
// Parametri: nessuno
// Ritorna: nulla
func (sl *SpanLogger) ReleaseSuccess() {
	sl.sendLogCmd(OpReleaseSuccess, nil)
}

// Timeout genera un record di timeout, lo aggiunge al buffer e invia OpTimeout.
// Cosa fa: crea un record di errore relativo al timeout e invia il comando di timeout.
// Parametri: nessuno
// Ritorna: nulla
func (sl *SpanLogger) Timeout() {
	// Creo un record di timeout
	record := slog.NewRecord(time.Now(), slog.LevelError, "Span timeout reached", 0)
	// Aggiungo il record al buffer
	sl.buffer = append(sl.buffer, record)
	// Invio il comando di timeout
	sl.sendLogCmd(OpTimeout, errors.New("Span timeout reached"))
}
