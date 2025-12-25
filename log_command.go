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
