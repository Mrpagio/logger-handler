package loggerhandler

import (
	"go.opentelemetry.io/otel/metric"
)

// MeterInterface espone i metodi usati per creare strumenti metrici.
// Questo file contiene solo le interfacce/stub minime per permettere
// all'utente di trasformare il progetto in una libreria importabile
// come github.com/Mrpagio/logger-handler.

type MeterInterface interface {
	Int64Counter(name string, opts ...metric.InstrumentOption) (Int64CounterLike, error)
	Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (Int64UpDownCounterLike, error)
}

// Tipi placeholder minimi. Possono essere estesi dall'utente con metodi concreti
// se si vuole esporre funzionalità reali dei contatori.

type Int64CounterLike interface{
	Add(value int64, opts ...metric.AddOption)
}

type Int64UpDownCounterLike interface{
	Add(value int64, opts ...metric.AddOption)
}

