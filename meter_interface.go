package loggerhandler

import (
	"go.opentelemetry.io/otel/metric"
)

// MeterInterface espone i metodi usati per creare strumenti metrici.
// Cosa fa: fornisce un'astrazione minima sulle factory per creare contatori
//
//	e contatori up/down in modo che il pacchetto non dipenda direttamente
//	da una specifica implementazione di OpenTelemetry.
//
// Parametri: nessuno
// Ritorna: nessuna (è un'interfaccia)
type MeterInterface interface {
	// Int64Counter crea o restituisce un contatore di tipo Int64.
	// Parametri:
	//  - name: nome della metrica
	//  - opts: opzioni aggiuntive per l'istanza della metrica
	// Ritorna: Int64CounterLike (implementazione specifica) e errore
	Int64Counter(name string, opts ...metric.InstrumentOption) (Int64CounterLike, error)
	// Int64UpDownCounter crea o restituisce un contatore up/down di tipo Int64.
	// Parametri:
	//  - name: nome della metrica
	//  - opts: opzioni aggiuntive per l'istanza della metrica
	// Ritorna: Int64UpDownCounterLike (implementazione specifica) e errore
	Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (Int64UpDownCounterLike, error)
}

// Int64CounterLike è un tipo placeholder che espone il metodo Add per contatori.
// Cosa fa: permette al codice di chiamare Add senza dipendere dall'implementazione reale.
// Parametri: nessuno
// Ritorna: nessuno (è un'interfaccia)
type Int64CounterLike interface {
	// Add incrementa il contatore del valore specificato.
	// Parametri:
	//  - value: valore da aggiungere (int64)
	//  - opts: opzioni addizionali per l'operazione
	// Ritorna: nulla
	Add(value int64, opts ...metric.AddOption)
}

// Int64UpDownCounterLike è un tipo placeholder che espone il metodo Add per contatori up/down.
// Cosa fa: permette di incrementare o decrementare una metrica istantanea.
// Parametri: nessuno
// Ritorna: nessuno (è un'interfaccia)
type Int64UpDownCounterLike interface {
	// Add aggiunge (o sottrae se value è negativo) il valore specificato.
	// Parametri:
	//  - value: valore da aggiungere (int64)
	//  - opts: opzioni addizionali per l'operazione
	// Ritorna: nulla
	Add(value int64, opts ...metric.AddOption)
}
