# loggerhandler

1. Descrizione generale della libreria

loggerhandler è una libreria che fornisce un semplice gestore di "span" di log: ogni span accumula record di log (Debug/Info/Warn/Error) e invia comandi a un `LoggerHandler` centrale che formatta i record, si occupa della scrittura su writer (console/file), gestisce timeout degli span e mantiene metriche tramite un'interfaccia astratta `MeterInterface`.

La libreria è pensata per separare la raccolta locale dei log (nei singoli `SpanLogger`) dall'inserimento effettivo nel sistema di I/O e di metriche (gestito da `LoggerHandler`), permettendo anche di configurare writer di file con rotazione tramite `lumberjack` e di sostituire il provider di metriche tramite `MeterInterface`.

---

2. Tipi

2.1 WriterConfigs

2.1.1 Campi
- `consoleLogging bool` — abilita il logging su console
- `fileLocation string` — percorso del file di log (vuoto = no file)
- `fileMaxSize int` — massimo MB prima della rotazione
- `fileMaxBackups int` — numero di file di backup da mantenere
- `compress bool` — abilita la compressione dei backup
- `multi io.Writer` — writer combinato (stdout + file)
- `textHandler *slog.TextHandler` — handler testuale di slog

2.1.2 Metodi e funzioni
- 2.1.2.1 `NewLogConfigs(consoleLogging bool, fileLocation string, fileMaxSize int, fileMaxBackups int, compress bool) *WriterConfigs` — NewLogConfigs crea e inizializza una struttura WriterConfigs: imposta i campi, crea l'io.MultiWriter e inizializza il TextHandler.
- 2.1.2.2 `GetTextHandler() *slog.TextHandler` — GetTextHandler restituisce il TextHandler interno (può essere nil).
- 2.1.2.3 `GetMultiWriter() io.Writer` — GetMultiWriter restituisce l'io.Writer combinato.
- 2.1.2.4 `IsConsoleLoggingEnabled() bool` — IsConsoleLoggingEnabled indica se il logging su console è abilitato.
- 2.1.2.5 `GetFileLocation() string` — GetFileLocation restituisce il percorso del file di log configurato.
- 2.1.2.6 `GetFileMaxSize() int` — GetFileMaxSize restituisce la dimensione massima configurata per i file di log (MB).
- 2.1.2.7 `GetFileMaxBackups() int` — GetFileMaxBackups restituisce il numero di backup configurati.
- 2.1.2.8 `IsCompressEnabled() bool` — IsCompressEnabled indica se la compressione dei backup è abilitata.

---

2.2 OpType

2.2.1 Campi / Valori
- `OpLog` — comando normale di logging (accumulo di record)
- `OpReleaseSuccess` — rilascio dello span con successo
- `OpReleaseFailure` — rilascio dello span per errore
- `OpTimeout` — rilascio dello span per timeout

2.2.2 Metodi
- (nessuno direttamente; `LogCommand.TypeString()` fornisce la rappresentazione testuale)

---

2.3 LogCommand

2.3.1 Campi
- `Op OpType` — tipo di operazione
- `SpanID string` — identificatore dello span relativo
- `Records []slog.Record` — slice di record da processare
- `Err error` — errore opzionale (usato per `OpReleaseFailure` e `OpTimeout`)

2.3.2 Metodi e funzioni
- 2.3.2.1 `TypeString() string` — TypeString restituisce una rappresentazione testuale del tipo di operazione (es. "Log", "ReleaseFailure", ...).

---

2.4 SpanLogger

2.4.1 Campi
- `id string` — identificatore dello span
- `timeDuration time.Duration` — durata/timeout dello span
- `tags []string` — tag associati
- `bufferSize int` — dimensione dichiarata del buffer
- `buffer []slog.Record` — buffer interno dei record
- `loggerHandler *LoggerHandler` — riferimento al `LoggerHandler` che elabora i comandi
- `logLevel slog.Level` — livello minimo che causa l'invio immediato

2.4.2 Metodi e funzioni
- 2.4.2.1 `NewSpanLogger(id string, duration time.Duration, tags []string, bufferSize int, loggerHandler *LoggerHandler, level slog.Level) *SpanLogger` — NewSpanLogger crea un nuovo SpanLogger: costruisce e ritorna un oggetto SpanLogger con i parametri forniti.
- 2.4.2.2 `GetID() string` — GetID restituisce l'id dello span.
- 2.4.2.3 `GetDuration() time.Duration` — GetDuration restituisce la durata (timeout) dello span.
- 2.4.2.4 `GetTags() []string` — GetTags restituisce i tag associati allo span.
- 2.4.2.5 `GetBufferSize() int` — GetBufferSize restituisce la dimensione del buffer interno dichiarata.
- 2.4.2.6 `GetLoggerHandler() *LoggerHandler` — GetLoggerHandler restituisce il riferimento al LoggerHandler associato.
- 2.4.2.7 `GetLogLevel() slog.Level` — GetLogLevel restituisce il livello di logging configurato per lo span.
- 2.4.2.8 `Debug(msg string, attrs ...slog.Attr)` — Debug aggiunge un record di debug al buffer e, se il livello lo richiede, invia il comando (OpLog).
- 2.4.2.9 `Info(msg string, attrs ...slog.Attr)` — Info aggiunge un record di info al buffer e, se il livello lo richiede, invia il comando (OpLog).
- 2.4.2.10 `Warn(msg string, attrs ...slog.Attr)` — Warn aggiunge un record di warning al buffer e, se il livello lo richiede, invia il comando (OpLog).
- 2.4.2.11 `Error(msg string, attrs ...slog.Attr)` — Error aggiunge un record di errore al buffer e invia immediatamente un OpReleaseFailure.
- 2.4.2.12 `ReleaseSuccess()` — ReleaseSuccess invia un comando di rilascio con successo (OpReleaseSuccess).
- 2.4.2.13 `Timeout()` — Timeout genera un record di timeout, lo aggiunge al buffer e invia OpTimeout.

---

2.5 MeterInterface

2.5.1 Campi / Scopo
- Interfaccia che astrae la creazione di strumenti metrici (contatori) per evitare dipendenze dirette da una specifica libreria OpenTelemetry.

2.5.2 Metodi
- 2.5.2.1 `Int64Counter(name string, opts ...metric.InstrumentOption) (Int64CounterLike, error)` — Int64Counter crea o restituisce un contatore di tipo Int64.
- 2.5.2.2 `Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (Int64UpDownCounterLike, error)` — Int64UpDownCounter crea o restituisce un contatore up/down di tipo Int64.

---

2.6 Int64CounterLike

2.6.1 Scopo
- Interfaccia placeholder che espone `Add` per contatori cumulativi.

2.6.2 Metodi
- 2.6.2.1 `Add(value int64, opts ...metric.AddOption)` — Add incrementa il contatore del valore specificato.

---

2.7 Int64UpDownCounterLike

2.7.1 Scopo
- Interfaccia placeholder che espone `Add` per contatori up/down.

2.7.2 Metodi
- 2.7.2.1 `Add(value int64, opts ...metric.AddOption)` — Add aggiunge (o sottrae se value è negativo) il valore specificato.

---

2.8 LoggerHandler

2.8.1 Campi (selezione)
- `logWriter *WriterConfigs`, `errWriter *WriterConfigs` — writer per log normali ed errori
- `tmpHandler *slog.JSONHandler` — handler temporaneo per formattare record
- `meter MeterInterface` — astrazione per creare metriche
- `spans map[string]*SpanLogger` — mappa degli span registrati
- `timeouts map[string]*time.Timer` — timer per span con timeout
- `chTimers chan string` — canale che riceve notifiche di timer scaduti
- `channel chan LogCommand` — canale di comandi in ingresso
- `wg *sync.WaitGroup`, `mu *sync.Mutex`, `closing int32` — sincronizzazione e stato
- metriche: `totalCounter, successCounter, failureCounter, discardedCounter, invalidSpanCounter, activeSpansGauge`

2.8.2 Metodi e funzioni
- 2.8.2.1 `NewLoggerHandler(logConfig *WriterConfigs, errConfig *WriterConfigs, meter MeterInterface, bufferSize int) *LoggerHandler` — NewLoggerHandler crea e inizializza un nuovo LoggerHandler: alloca le strutture dati, inizializza handler temporaneo e metriche e avvia le goroutine che processano comandi e notifiche di timeout.
- 2.8.2.2 `GetMeter() MeterInterface` — GetMeter restituisce l'istanza di MeterInterface associata al LoggerHandler.
- 2.8.2.3 `GetLogHandler() *WriterConfigs` — GetLogHandler restituisce la configurazione del writer dei log normali.
- 2.8.2.4 `GetErrorHandler() *WriterConfigs` — GetErrorHandler restituisce la configurazione del writer degli errori.
- 2.8.2.5 `GetSpans() map[string]*SpanLogger` — GetSpans restituisce una copia superficiale della mappa degli SpanLogger, protetta dal mutex.
- 2.8.2.6 `AddSpan(duration time.Duration, tags []string, bufferSize int, level slog.Level) *SpanLogger` — AddSpan crea e registra un nuovo SpanLogger: genera un spanID univoco, crea lo SpanLogger e registra il timer per il timeout se richiesto.
- 2.8.2.7 `RemoveSpan(id string)` — RemoveSpan rimuove lo span con l'id fornito e ferma il timer associato.
- 2.8.2.8 `AppendCommand(cmd LogCommand)` — AppendCommand prova ad aggiungere un LogCommand al canale interno in modo non bloccante; se il canale è pieno incrementa il contatore dei comandi scartati.
- 2.8.2.9 `processCommand(cmd LogCommand)` — processCommand processa un singolo LogCommand: verifica che lo span esista, costruisce la stringa di log e scrive sul writer appropriato.
- 2.8.2.10 `Close()` — Close ferma tutti i timer, chiude i canali e aspetta la terminazione delle goroutine.

---

3. Esempi

3.1 Note generali
- Gli esempi nella sezione utilizzano un `fakeMeter` minimale che implementa l'interfaccia `MeterInterface` per permettere l'esecuzione senza configurare un provider reale di OpenTelemetry.
- Esiste anche un esempio reale eseguibile in `example/main.go` nella repository.

3.2 Basic usage (snippet)

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
	logCfg := loggerhandler.NewLogConfigs(true, "", 10, 3, false)
	errCfg := loggerhandler.NewLogConfigs(true, "", 10, 3, false)
	meter := &fakeMeter{}
	lh := loggerhandler.NewLoggerHandler(logCfg, errCfg, meter, 100)

	span := lh.AddSpan(0, []string{"example"}, 10, slog.LevelDebug)
	span.Info("starting span")
	span.Debug("debug message")
	span.ReleaseSuccess()
	// attendere un breve periodo per consentire l'elaborazione dei comandi
	time.Sleep(100 * time.Millisecond)
}
```

3.3 Using timeouts (snippet)

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

	span := lh.AddSpan(50*time.Millisecond, []string{"timeout-example"}, 10, slog.LevelInfo)
	span.Info("doing work")
	// attendere più a lungo del timeout per vedere il comportamento
	time.Sleep(150 * time.Millisecond)
}
```

---

Se vuoi che aggiunga esempi aggiuntivi (ad esempio: scrittura su file, integrazione con un reale provider OpenTelemetry per le metriche, o snippet per testare la rotazione dei file con `lumberjack`), li aggiungo volentieri.
