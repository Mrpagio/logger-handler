# loggerhandler

Documentazione rapida dei tipi definiti nella libreria `loggerhandler`.

Indice

- [WriterConfigs](#writerconfigs)
- [OpType](#optype)
- [LogCommand](#logcommand)
- [SpanLogger](#spanlogger)
- [MeterInterface](#meterinterface)
- [Int64CounterLike](#int64counterlike)
- [Int64UpDownCounterLike](#int64updowncounterlike)
- [LoggerHandler](#loggerhandler)
- [Examples](#examples)

---

## WriterConfigs
Descrizione: Configurazioni per i writer dei log (console e file). Gestisce la creazione di un io.MultiWriter e di un `slog.TextHandler` utilizzato per il logging.

Campi principali:
- `consoleLogging bool` — abilita il logging su console
- `fileLocation string` — percorso del file di log (vuoto = no file)
- `fileMaxSize int` — massimo MB prima della rotazione
- `fileMaxBackups int` — numero di backup da mantenere
- `compress bool` — abilita la compressione dei backup
- `multi io.Writer` — writer combinato (stdout + file)
- `textHandler *slog.TextHandler` — handler testuale di slog

Metodi utili:
- `NewLogConfigs(consoleLogging bool, fileLocation string, fileMaxSize int, fileMaxBackups int, compress bool) *WriterConfigs` — costruttore
- `GetTextHandler() *slog.TextHandler`
- `GetMultiWriter() io.Writer`
- `IsConsoleLoggingEnabled() bool`
- `GetFileLocation() string`
- `GetFileMaxSize() int`
- `GetFileMaxBackups() int`
- `IsCompressEnabled() bool`

---

## OpType
Descrizione: tipo enumerato che rappresenta il tipo di operazione di un `LogCommand`.

Valori:
- `OpLog` — comando normale di logging (accumulo di record)
- `OpReleaseSuccess` — rilascio dello span con successo
- `OpReleaseFailure` — rilascio dello span per errore
- `OpTimeout` — rilascio dello span per timeout

---

## LogCommand
Descrizione: struttura che rappresenta un comando inviato al `LoggerHandler` per essere processato.

Campi:
- `Op OpType` — tipo di operazione
- `SpanID string` — identificatore dello span relativo
- `Records []slog.Record` — slice di record da processare
- `Err error` — errore opzionale (usato per `OpReleaseFailure` e `OpTimeout`)

Metodi:
- `TypeString() string` — ritorna una rappresentazione testuale del campo `Op` (es. "Log", "ReleaseFailure", ...)

---

## SpanLogger
Descrizione: logger legato a uno "span" temporale; accumula record e notifica il `LoggerHandler` quando necessario (livelli alti, release, timeout).

Campi principali:
- `id string` — identificatore dello span
- `timeDuration time.Duration` — durata/timeout dello span
- `tags []string` — tag associati
- `bufferSize int` — dimensione dichiarata del buffer
- `buffer []slog.Record` — buffer interno dei record
- `loggerHandler *LoggerHandler` — riferimento al `LoggerHandler` che elabora i comandi
- `logLevel slog.Level` — livello minimo che causa l'invio immediato

Metodi rilevanti:
- `NewSpanLogger(id string, duration time.Duration, tags []string, bufferSize int, loggerHandler *LoggerHandler, level slog.Level) *SpanLogger`
- `GetID() string`, `GetDuration() time.Duration`, `GetTags() []string`, `GetBufferSize() int`, `GetLoggerHandler() *LoggerHandler`, `GetLogLevel() slog.Level`
- `Debug(msg string, attrs ...slog.Attr)`, `Info(...)`, `Warn(...)` — aggiungono record al buffer e inviano `OpLog` se il livello lo richiede
- `Error(msg string, attrs ...slog.Attr)` — aggiunge un record di errore e invia `OpReleaseFailure`
- `ReleaseSuccess()` — invia `OpReleaseSuccess`
- `Timeout()` — aggiunge un record di timeout e invia `OpTimeout`

Comportamento: quando necessario costruisce un `LogCommand` e lo invia al `LoggerHandler` tramite `AppendCommand`.

---

## MeterInterface
Descrizione: interfaccia che astrae la creazione di strumenti metrici (contatori) per evitare dipendenze dirette da una specifica libreria OpenTelemetry.

Metodi:
- `Int64Counter(name string, opts ...metric.InstrumentOption) (Int64CounterLike, error)` — crea o restituisce un contatore cumulativo
- `Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (Int64UpDownCounterLike, error)` — crea o restituisce un contatore up/down (gauge istantaneo)

---

## Int64CounterLike
Descrizione: interfaccia che rappresenta un contatore int64 minimale (placeholder per le implementazioni reali).

Metodi:
- `Add(value int64, opts ...metric.AddOption)` — incrementa il contatore

---

## Int64UpDownCounterLike
Descrizione: interfaccia minima per un contatore up/down (permette incremento/decremento).

Metodi:
- `Add(value int64, opts ...metric.AddOption)` — aggiunge (o sottrae se negativo) il valore

---

## LoggerHandler
Descrizione: componente centrale che raccoglie i `LogCommand` inviati dagli `SpanLogger`, ne costruisce la rappresentazione testuale e scrive sui writer appropriati; gestisce inoltre timer per gli span, metriche e sincronizzazione.

Campi (selezione):
- `logWriter *WriterConfigs`, `errWriter *WriterConfigs` — writer per log normali ed errori
- `tmpHandler *slog.JSONHandler` — handler temporaneo per formattare record
- `meter MeterInterface` — astrazione per creare metriche
- `spans map[string]*SpanLogger` — mappa degli span registrati
- `timeouts map[string]*time.Timer` — timer per span con timeout
- `chTimers chan string` — canale che riceve notifiche di timer scaduti
- `channel chan LogCommand` — canale di comandi in ingresso
- `wg *sync.WaitGroup`, `mu *sync.Mutex`, `closing int32` — sincronizzazione e stato
- metriche: `totalCounter, successCounter, failureCounter, discardedCounter, invalidSpanCounter, activeSpansGauge`

Metodi principali:
- `NewLoggerHandler(logConfig *WriterConfigs, errConfig *WriterConfigs, meter MeterInterface, bufferSize int) *LoggerHandler` — costruttore che avvia le goroutine di processamento
- `AddSpan(duration time.Duration, tags []string, bufferSize int, level slog.Level) *SpanLogger` — crea e registra un nuovo span
- `RemoveSpan(id string)` — rimuove uno span e cancella il timer associato
- `AppendCommand(cmd LogCommand)` — tenta di aggiungere un comando al canale (non bloccante)
- `processCommand(cmd LogCommand)` — elabora un comando: verifica span, costruisce stringa di log e scrive sul writer
- varie getter per metriche e configurazioni

---

## Examples
Ecco alcuni esempi rapidi su come utilizzare la libreria. Gli esempi mostrano un fake meter minimale per poter usare `NewLoggerHandler` senza dipendenze esterne.

### Basic usage
Esempio base che scrive log su console utilizzando un fake meter che implementa l'interfaccia `MeterInterface`.

```go
package main

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/metric"
	loggerhandler "github.com/Mrpagio/logger-handler"
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
```

### Using timeouts
Esempio che mostra l'uso del timeout per uno span (dopo la durata scatta Timeout e viene inviato un comando OpTimeout).

```go
package main

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/metric"
	loggerhandler "github.com/Mrpagio/logger-handler"
}

// Reuso i fake types dall'esempio precedente
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
	logCfg := loggerhandler.NewLogConfigs(true, "", 10, 3, false)
	errCfg := loggerhandler.NewLogConfigs(true, "", 10, 3, false)
	meter := &fakeMeter{}
	lh := loggerhandler.NewLoggerHandler(logCfg, errCfg, meter, 100)

	// Span con timeout di 50ms
	span := lh.AddSpan(50*time.Millisecond, []string{"timeout-example"}, 10, slog.LevelInfo)

	span.Info("doing work")

	// Attendo più a lungo del timeout per vedere il comportamento
	time.Sleep(100 * time.Millisecond)

	// Non è necessario chiamare RemoveSpan() qui: il LoggerHandler rimuove
	// lo span internamente quando elabora l'evento di timeout.
}
```

---

Se vuoi che aggiunga esempi aggiuntivi (ad esempio: scrittura su file, integrazione con un reale provider OpenTelemetry per le metriche, o snippet per testare la rotazione dei file con `lumberjack`), li aggiungo volentieri.
