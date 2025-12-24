package loggerhandler

import "log/slog"

type OpType int

const (
	OpLog OpType = iota
	OpReleaseSuccess
	OpReleaseFailure
	OpTimeout
)

type LogCommand struct {
	Op      OpType
	SpanID  string
	Records []slog.Record
	Err     error // Usato solo per OpReleaseFailure
}
