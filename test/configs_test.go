package loggerhandler_test

import (
	"log/slog"
	"testing"

	loggerhandler "github.com/Mrpagio/logger-handler"
)

func TestNewLogConfigsAndGetters(t *testing.T) {
	wc := loggerhandler.NewLogConfigs(true, "", 10, 2, true)
	if wc == nil {
		t.Fatal("NewLogConfigs returned nil")
	}
	if wc.GetMultiWriter() == nil {
		t.Fatal("GetMultiWriter returned nil")
	}
	if wc.GetTextHandler() == nil {
		t.Fatal("GetTextHandler returned nil")
	}
	if !wc.IsConsoleLoggingEnabled() {
		t.Fatal("IsConsoleLoggingEnabled expected true")
	}
	if wc.GetFileLocation() != "" {
		t.Fatal("GetFileLocation expected empty string")
	}
	if wc.GetFileMaxSize() != 10 {
		t.Fatalf("GetFileMaxSize expected 10, got %d", wc.GetFileMaxSize())
	}
	if wc.GetFileMaxBackups() != 2 {
		t.Fatalf("GetFileMaxBackups expected 2, got %d", wc.GetFileMaxBackups())
	}
	if !wc.IsCompressEnabled() {
		t.Fatal("IsCompressEnabled expected true")
	}

	// ensure TextHandler can be used to log
	handler := wc.GetTextHandler()
	logger := slog.New(handler)
	logger.Info("test", "k", "v")
}
