package main

import (
	"log/slog"
	"time"

	loggerhandler "github.com/Mrpagio/logger-handler"
	"go.opentelemetry.io/otel/metric"
)

// Fake counter e meter di esempio (minimale)
type fakeCounter struct{}

func (f *fakeCounter) Add(v int64, _ ...metric.AddOption) {}

type fakeUpDownCounter struct{}

func (f *fakeUpDownCounter) Add(v int64, _ ...metric.AddOption) {}

type fakeMeter struct{}

func (f *fakeMeter) Int64Counter(name string, opts ...metric.InstrumentOption) (loggerhandler.Int64CounterLike, error) {
	return &fakeCounter{}, nil
}
func (f *fakeMeter) Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (loggerhandler.Int64UpDownCounterLike, error) {
	return &fakeUpDownCounter{}, nil
}

func main() {
	// Configuro writer: console abilitato, nessun file
	logCfg := loggerhandler.NewLogConfigs(true, "", 10, 3, false)
	errCfg := loggerhandler.NewLogConfigs(true, "", 10, 3, false)

	meter := &fakeMeter{}
	lh := loggerhandler.NewLoggerHandler(logCfg, errCfg, meter, 100)

	// Creo uno span senza timeout
	span := lh.AddSpan(0, []string{"example"}, 10, slog.LevelDebug)

	span.Info("starting span")
	span.Debug("debug message")

	// Segnalo successo. Non chiamare RemoveSpan() subito: il LoggerHandler
	// processerà il comando e rimuoverà lo span internamente.
	span.ReleaseSuccess()

	// Attendere brevemente per permettere l'elaborazione dei comandi
	time.Sleep(50 * time.Millisecond)
}
