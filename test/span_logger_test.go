package loggerhandler_test

import (
	"log/slog"
	"testing"
	"time"

	loggerhandler "github.com/Mrpagio/logger-handler"
	"go.opentelemetry.io/otel/metric"
)

func TestSpanLoggerBasicAndCommands(t *testing.T) {
	logCfg := loggerhandler.NewLogConfigs(true, "", 1, 1, false)
	errCfg := loggerhandler.NewLogConfigs(true, "", 1, 1, false)
	fm := &testFakeMeter{}
	lh := loggerhandler.NewLoggerHandler(logCfg, errCfg, fm, 10)
	defer lh.Close()

	// span with immediate send level (Debug) - use AddSpan to register it
	span := lh.AddSpan(0, []string{"t"}, 5, slog.LevelDebug)
	if span.GetID() == "" {
		t.Fatalf("expected non-empty id")
	}
	if span.GetDuration() != 0 {
		t.Fatalf("expected duration 0")
	}
	// trigger Debug -> should send OpLog (via handler channel)
	span.Debug("dbg")
	// give a small time for append to happen
	time.Sleep(10 * time.Millisecond)
	// we can inspect exported GetSpans to ensure span still present
	spans := lh.GetSpans()
	if _, ok := spans[span.GetID()]; !ok {
		t.Fatalf("span %s expected to be present", span.GetID())
	}

	// Error should send OpReleaseFailure (this will also remove span)
	span.Error("err")
	// give time to process
	time.Sleep(10 * time.Millisecond)

	// ReleaseSuccess (on a fresh span)
	sp2 := lh.AddSpan(0, nil, 5, slog.LevelDebug)
	sp2.ReleaseSuccess()
	// Timeout invocation should also produce OpTimeout
	sp3 := lh.AddSpan(0, nil, 5, slog.LevelDebug)
	sp3.Timeout()

	// ensure methods don't block; test that info at higher level doesn't send if level is Error
	sp4 := lh.AddSpan(0, nil, 5, slog.LevelError)
	sp4.Info("info")
	// ok if no panic and no deadlock
}

// test-only fake meter used by this file
type testFakeCounter struct{}

func (c *testFakeCounter) Add(v int64, _ ...metric.AddOption) {}

type testFakeUpDownCounter struct{}

func (c *testFakeUpDownCounter) Add(v int64, _ ...metric.AddOption) {}

type testFakeMeter struct{}

func (f *testFakeMeter) Int64Counter(name string, opts ...metric.InstrumentOption) (loggerhandler.Int64CounterLike, error) {
	return &testFakeCounter{}, nil
}
func (f *testFakeMeter) Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (loggerhandler.Int64UpDownCounterLike, error) {
	return &testFakeUpDownCounter{}, nil
}

// provide a simple alias used above
var _ = time.Now // keep import used

// alias type used above
type testFakeMeterAlias = testFakeMeter

// create instance type expected in test
var _ = testFakeMeterAlias{}
