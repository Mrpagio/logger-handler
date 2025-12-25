package loggerhandler

import "log/slog"

type OpType int

const (
	// OpType rappresenta il tipo di operazione associata a un LogCommand.
	// OpLog: comando di logging normale (accumulo di record)
	// OpReleaseSuccess: rilascio dello span con successo
	// OpReleaseFailure: rilascio dello span per errore
	// OpTimeout: rilascio dello span per timeout
	OpLog OpType = iota
	OpReleaseSuccess
	OpReleaseFailure
	OpTimeout
)

// LogCommand è la struttura che descrive un'operazione da eseguire sul LoggerHandler.
// Campi:
//   - Op: tipo di operazione (OpType)
//   - SpanID: identificatore dello span a cui il comando si riferisce
//   - Records: slice di slog.Record accumulati nello span
//   - Err: errore opzionale (usato per OpReleaseFailure e OpTimeout)
type LogCommand struct {
	Op      OpType
	SpanID  string
	Records []slog.Record
	Err     error // Usato solo per OpReleaseFailure
}

// TypeString restituisce una rappresentazione testuale del tipo di operazione.
// Parametri: nessuno
// Ritorna: stringa descrittiva del tipo di operazione
func (lc *LogCommand) TypeString() string {
	switch lc.Op {
	case OpLog:
		return "Log"
	case OpReleaseSuccess:
		return "ReleaseSuccess"
	case OpReleaseFailure:
		return "ReleaseFailure"
	case OpTimeout:
		return "Timeout"
	default:
		return "Unknown"
	}
}
