package loggerhandler_test

import (
	"log/slog"
	"math/rand"
	"sync"
	"testing"
	"time"

	loggerhandler "github.com/Mrpagio/logger-handler"
	"go.opentelemetry.io/otel/metric"
)

// fake implementations for MeterInterface used in tests
type testCounter struct{}

func (c *testCounter) Add(v int64, _ ...metric.AddOption) {}

type testUpDownCounter struct{}

func (c *testUpDownCounter) Add(v int64, _ ...metric.AddOption) {}

// fakeMeter implements MeterInterface for tests
type fakeMeter struct{}

func (f *fakeMeter) Int64Counter(name string, opts ...metric.InstrumentOption) (loggerhandler.Int64CounterLike, error) {
	return &testCounter{}, nil
}
func (f *fakeMeter) Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (loggerhandler.Int64UpDownCounterLike, error) {
	return &testUpDownCounter{}, nil
}

// helper: crea un LoggerHandler di test con fakeMeter
func makeTestHandler(t *testing.T, bufferSize int) *loggerhandler.LoggerHandler {
	t.Helper()
	logCfg := loggerhandler.NewLogConfigs(true, "", 1, 1, false)
	errCfg := loggerhandler.NewLogConfigs(true, "", 1, 1, false)
	fm := &fakeMeter{}
	lh := loggerhandler.NewLoggerHandler(logCfg, errCfg, fm, bufferSize)
	if lh == nil {
		t.Fatal("NewLoggerHandler returned nil")
	}
	return lh
}

// helper: crea uno span con timeout e restituisce lo span
func addSpanHelper(t *testing.T, lh *loggerhandler.LoggerHandler, timeout time.Duration, level slog.Level) *loggerhandler.SpanLogger {
	t.Helper()
	sp := lh.AddSpan(timeout, []string{"tag"}, 5, level)
	if sp == nil {
		t.Fatal("AddSpan returned nil")
	}
	return sp
}

// helper: crea un LoggerHandler di test con fakeMeter ma senza logging su console (silenzioso)
func makeQuietTestHandler(t *testing.T, bufferSize int) *loggerhandler.LoggerHandler {
	t.Helper()
	logCfg := loggerhandler.NewLogConfigs(false, "", 1, 1, false)
	errCfg := loggerhandler.NewLogConfigs(false, "", 1, 1, false)
	fm := &fakeMeter{}
	lh := loggerhandler.NewLoggerHandler(logCfg, errCfg, fm, bufferSize)
	if lh == nil {
		t.Fatal("NewLoggerHandler returned nil")
	}
	return lh
}

// TestNewLoggerHandler verifica che NewLoggerHandler ritorni un'istanza non nil.
func TestNewLoggerHandler(t *testing.T) {
	logCfg := loggerhandler.NewLogConfigs(true, "", 1, 1, false)
	errCfg := loggerhandler.NewLogConfigs(true, "", 1, 1, false)
	fm := &fakeMeter{}
	lh := loggerhandler.NewLoggerHandler(logCfg, errCfg, fm, 10)
	if lh == nil {
		t.Fatal("NewLoggerHandler returned nil")
	}
	lh.Close()
}

// Table-driven: verifica AddSpan e RemoveSpan per diversi timeout/level
func TestAddSpanTable(t *testing.T) {
	cases := []struct {
		name    string
		timeout time.Duration
		level   slog.Level
	}{
		{"no-timeout-debug", 0, slog.LevelDebug},
		{"short-timeout-info", 10 * time.Millisecond, slog.LevelInfo},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			lh := makeTestHandler(t, 10)
			defer lh.Close()
			sp := addSpanHelper(t, lh, c.timeout, c.level)
			id := sp.GetID()
			// presente subito
			if _, ok := lh.GetSpans()[id]; !ok {
				t.Fatalf("span %s not found", id)
			}
			// rimuovo e controllo
			lh.RemoveSpan(id)
			if _, ok := lh.GetSpans()[id]; ok {
				t.Fatalf("span %s still present after RemoveSpan", id)
			}
		})
	}
}

// Test timeout removal table-driven
func TestTimeoutTable(t *testing.T) {
	cases := []struct {
		name    string
		timeout time.Duration
	}{
		{"very-short", 10 * time.Millisecond},
		{"short", 30 * time.Millisecond},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			lh := makeTestHandler(t, 10)
			defer lh.Close()
			sp := addSpanHelper(t, lh, c.timeout, slog.LevelDebug)
			id := sp.GetID()
			// presente subito
			if _, ok := lh.GetSpans()[id]; !ok {
				t.Fatalf("span %s not found immediately after AddSpan", id)
			}
			// aspetta e verifica che venga rimosso
			time.Sleep(200 * time.Millisecond)
			if _, ok := lh.GetSpans()[id]; ok {
				t.Fatalf("span %s still present after timeout", id)
			}
		})
	}
}

// Stress test concurrency dei timer: crea molti span in parallelo con timeout casuali
func TestTimersConcurrencyStress(t *testing.T) {
	lh := makeQuietTestHandler(t, 1000)
	defer lh.Close()

	N := 200
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			d := time.Duration(rand.Intn(50)) * time.Millisecond
			sp := lh.AddSpan(d, []string{"s"}, 5, slog.LevelDebug)
			// some spans we remove early
			if i%10 == 0 {
				lh.RemoveSpan(sp.GetID())
			}
		}()
	}
	wg.Wait()
	// aspetta che i timer scadano
	time.Sleep(500 * time.Millisecond)
	// nessun panic e chiudi
}

// Test specifico: OpRelease inviata e ancora in coda quando arriva Timeout
func TestTimeoutAfterReleaseQueued(t *testing.T) {
	lh := makeTestHandler(t, 10)
	defer lh.Close()

	sp := addSpanHelper(t, lh, 0, slog.LevelDebug)
	id := sp.GetID()

	// Simula: invia OpReleaseSuccess direttamente nel channel (prima)
	rel := loggerhandler.LogCommand{Op: loggerhandler.OpReleaseSuccess, SpanID: id, Records: []slog.Record{}}
	lh.AppendCommand(rel)

	// Ora invia OpTimeout come se il timer fosse scattato subito dopo
	tout := loggerhandler.LogCommand{Op: loggerhandler.OpTimeout, SpanID: id, Records: []slog.Record{}}
	lh.AppendCommand(tout)

	// aspetta che vengano processati
	time.Sleep(100 * time.Millisecond)

	// lo span non deve essere presente
	if _, ok := lh.GetSpans()[id]; ok {
		t.Fatalf("span %s still present after Release+Timeout processing", id)
	}
}
