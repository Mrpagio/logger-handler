package loggerhandler_test

import (
	"log/slog"
	"testing"

	loggerhandler "github.com/Mrpagio/logger-handler"
)

func TestLogCommandTypeString(t *testing.T) {
	lc := loggerhandler.LogCommand{Op: loggerhandler.OpLog}
	if lc.TypeString() != "Log" {
		t.Fatalf("expected Log, got %s", lc.TypeString())
	}
	lc.Op = loggerhandler.OpReleaseSuccess
	if lc.TypeString() != "ReleaseSuccess" {
		t.Fatalf("expected ReleaseSuccess, got %s", lc.TypeString())
	}
	lc.Op = loggerhandler.OpReleaseFailure
	if lc.TypeString() != "ReleaseFailure" {
		t.Fatalf("expected ReleaseFailure, got %s", lc.TypeString())
	}
	lc.Op = loggerhandler.OpTimeout
	if lc.TypeString() != "Timeout" {
		t.Fatalf("expected Timeout, got %s", lc.TypeString())
	}

	// basic construction
	lc = loggerhandler.LogCommand{Op: loggerhandler.OpLog, SpanID: "s1", Records: []slog.Record{}, Err: nil}
	if lc.SpanID != "s1" {
		t.Fatalf("span id mismatch")
	}
}
