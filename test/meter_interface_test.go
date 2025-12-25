package loggerhandler_test

import (
	"go.opentelemetry.io/otel/metric"
	"testing"

	loggerhandler "github.com/Mrpagio/logger-handler"
)

// testCounter e testUpDownCounter sono implementazioni di test per le interfacce
type testCounterMI struct{ called bool }

func (t *testCounterMI) Add(v int64, _ ...metric.AddOption) { t.called = true }

type testUpDownCounterMI struct{ called bool }

func (t *testUpDownCounterMI) Add(v int64, _ ...metric.AddOption) { t.called = true }

// fakeMeterMI implementa MeterInterface usando i tipi di test sopra
type fakeMeterMI struct{}

func (f *fakeMeterMI) Int64Counter(name string, opts ...metric.InstrumentOption) (loggerhandler.Int64CounterLike, error) {
	return &testCounterMI{}, nil
}
func (f *fakeMeterMI) Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (loggerhandler.Int64UpDownCounterLike, error) {
	return &testUpDownCounterMI{}, nil
}

func TestMeterInterfaceFakeImplementations(t *testing.T) {
	m := &fakeMeterMI{}
	c, err := m.Int64Counter("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil counter")
	}
	c.Add(1)

	u, err := m.Int64UpDownCounter("test2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u == nil {
		t.Fatal("expected non-nil updowncounter")
	}
	u.Add(-1)
}
