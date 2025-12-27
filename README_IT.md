# Logger Handler

Indice:

- [1 Descrizione generale della libreria](#descrizione-generale-della-libreria)
- [2 Tipi](#tipi)
  - [2.1 Tipo 1 (WriterConfigs)](#writerconfigs)
    - [2.1.1 Campi](#writerconfigs)
    - [2.1.2 Metodi e funzioni](#writerconfigs)
      - [2.1.2.1 `NewLogConfigs` (costruttore)](#writerconfigs)
      - [2.1.2.2 `toMulti`](#writerconfigs)
      - [2.1.2.3 `initTextHandler`](#writerconfigs)
      - [2.1.2.4 `GetTextHandler`](#writerconfigs)
      - [2.1.2.5 `GetMultiWriter`](#writerconfigs)
      - [2.1.2.6 `IsConsoleLoggingEnabled`](#writerconfigs)
      - [2.1.2.7 `GetFileLocation`](#writerconfigs)
      - [2.1.2.8 `GetFileMaxSize`](#writerconfigs)
      - [2.1.2.9 `GetFileMaxBackups`](#writerconfigs)
      - [2.1.2.10 `IsCompressEnabled`](#writerconfigs)
  - [2.2 Tipo 2 (MeterInterface + Int64CounterLike + Int64UpDownCounterLike)](#meterinterface)
    - [2.2.1 Campi / Descrizione](#meterinterface)
    - [2.2.2 Metodi e funzioni](#meterinterface)
      - [2.2.2.1 `Int64Counter` (MeterInterface)](#meterinterface)
      - [2.2.2.2 `Int64UpDownCounter` (MeterInterface)](#meterinterface)
      - [2.2.2.3 `Add` (Int64CounterLike)](#meterinterface)
      - [2.2.2.4 `Add` (Int64UpDownCounterLike)](#meterinterface)
  - [2.3 Tipo 3 (LogCommand e OpType)](#logcommand)
    - [2.3.1 Campi](#logcommand)
    - [2.3.2 Metodi](#logcommand)
      - [2.3.2.1 `TypeString`](#logcommand)
  - [2.4 Tipo 4 (LoggerHandler)](#loggerhandler)
    - [2.4.1 Campi](#loggerhandler)
    - [2.4.2 Metodi e funzioni](#loggerhandler)
      - [2.4.2.1 `NewLoggerHandler` (costruttore)](#loggerhandler)
      - [2.4.2.2 `initTempHandler`](#loggerhandler)
      - [2.4.2.3 `initMetrics`](#loggerhandler)
      - [2.4.2.4 `GetMeter`](#loggerhandler)
      - [2.4.2.5 `GetLogHandler`](#loggerhandler)
      - [2.4.2.6 `GetErrorHandler`](#loggerhandler)
      - [2.4.2.7 `GetSpans`](#loggerhandler)
      - [2.4.2.8 `GetTotalCounter`](#loggerhandler)
      - [2.4.2.9 `GetSuccessCounter`](#loggerhandler)
      - [2.4.2.10 `GetFailureCounter`](#loggerhandler)
      - [2.4.2.11 `GetDiscardedCounter`](#loggerhandler)
      - [2.4.2.12 `GetInvalidSpanCounter`](#loggerhandler)
      - [2.4.2.13 `GetActiveSpansGauge`](#loggerhandler)
      - [2.4.2.14 `AddSpan`](#loggerhandler)
      - [2.4.2.15 `RemoveSpan`](#loggerhandler)
      - [2.4.2.16 `AppendCommand`](#loggerhandler)
      - [2.4.2.17 `processCommand`, `createStrLog`, `writeToHandler`](#loggerhandler)
      - [2.4.2.18 `processOpLog`, `processOpSuccess`, `processOpFailure`, `processOpTimeout`](#loggerhandler)
      - [2.4.2.19 `checkSpanExists`](#loggerhandler)
      - [2.4.2.20 `Close`](#loggerhandler)
  - [2.5 Tipo 5 (SpanLogger)](#spanlogger)
    - [2.5.1 Campi](#spanlogger)
    - [2.5.2 Metodi e funzioni](#spanlogger)
      - [2.5.2.1 `NewSpanLogger` (costruttore)](#spanlogger)
      - [2.5.2.2 `GetID`, `GetDuration`, `GetTags`, `GetBufferSize`, `GetLoggerHandler`, `GetLogLevel`](#spanlogger)
      - [2.5.2.3 `sendLogCmd`, `Debug`, `Info`, `Warn`, `Error`, `ReleaseSuccess`, `Timeout`](#spanlogger)
- [3 Esempi](#esempi)

---

<a name="descrizione-generale-della-libreria"></a>
1 Descrizione generale della libreria

Logger Handler è una piccola libreria Go che fornisce un meccanismo di "span" per raggruppare record di log,
invio asincrono dei log, gestione di timeout per gli span e integrazione opzionale con un `MeterInterface`
(per creare metriche OpenTelemetry-like). La libreria non obbliga una dipendenza diretta da OTel: usa
interfacce (placeholder) per i contatori.

<a name="tipi"></a>
2 Tipi

2.1 Tipo 1 (WriterConfigs)
<a name="writerconfigs"></a>

2.1.1 Campi

- `consoleLogging bool` — Se true, si usa il logging su console
- `fileLocation string` — Percorso del file di log. Se vuoto, non si usa il file logging
- `fileMaxSize int` — MB prima di fare la rotazione
- `fileMaxBackups int` — Numero di file di backup da mantenere
- `compress bool` — Se true, i file di log di backup vengono compressi in gzip
- `multi io.Writer` — writer combinato (stdout + file se configurati)
- `textHandler *slog.TextHandler` — handler di slog per scrittura testuale

2.1.2 Metodi e funzioni

- `NewLogConfigs(consoleLogging bool, fileLocation string, fileMaxSize int, fileMaxBackups int, compress bool) *WriterConfigs` — Costruttore
  - Commento: "NewLogConfigs crea e inizializza una struttura WriterConfigs. Cosa fa: imposta i campi in base ai parametri, crea l'io.MultiWriter e init del TextHandler."

- `toMulti()` — costruisce l'io.MultiWriter in base alla configurazione.
  - Commento: "toMulti costruisce l'io.MultiWriter in base alla configurazione. Cosa fa: aggiunge stdout e/o un writer su file (lumberjack) al MultiWriter."

- `initTextHandler()` — inizializza un TextHandler di slog usando il MultiWriter.
  - Commento: "initTextHandler inizializza un TextHandler di slog usando il MultiWriter."

- `GetTextHandler() *slog.TextHandler`
  - Commento: "GetTextHandler restituisce il TextHandler interno."

- `GetMultiWriter() io.Writer`
  - Commento: "GetMultiWriter restituisce l'io.Writer combinato."

- `IsConsoleLoggingEnabled() bool`
  - Commento: "IsConsoleLoggingEnabled indica se il logging su console è abilitato."

- `GetFileLocation() string`
  - Commento: "GetFileLocation restituisce il percorso del file di log configurato."

- `GetFileMaxSize() int`
  - Commento: "GetFileMaxSize restituisce la dimensione massima configurata per i file di log."

- `GetFileMaxBackups() int`
  - Commento: "GetFileMaxBackups restituisce il numero di backup configurati."

- `IsCompressEnabled() bool`
  - Commento: "IsCompressEnabled indica se la compressione dei backup è abilitata."

2.2 Tipo 2 (MeterInterface + Int64CounterLike + Int64UpDownCounterLike)
<a name="meterinterface"></a>

2.2.1 Campi / Descrizione

- `MeterInterface` è un'interfaccia che espone metodi per ottenere strumenti metrici (Int64Counter e Int64UpDownCounter).
  - Commento: "MeterInterface espone i metodi usati per creare strumenti metrici. Cosa fa: fornisce un'astrazione minima sulle factory per creare contatori e contatori up/down in modo che il pacchetto non dipenda direttamente da una specifica implementazione di OpenTelemetry."

- `Int64CounterLike` — interfaccia con metodo `Add(value int64, opts ...metric.AddOption)`
  - Commento: "Int64CounterLike è un tipo placeholder che espone il metodo Add per contatori. Cosa fa: permette al codice di chiamare Add senza dipendere dall'implementazione reale."

- `Int64UpDownCounterLike` — interfaccia con metodo `Add(value int64, opts ...metric.AddOption)`
  - Commento: "Int64UpDownCounterLike è un tipo placeholder che espone il metodo Add per contatori up/down. Cosa fa: permette di incrementare o decrementare una metrica istantanea."

2.2.2 Metodi e funzioni

- `Int64Counter(name string, opts ...metric.InstrumentOption) (Int64CounterLike, error)` — crea o restituisce un contatore di tipo Int64.
  - Commento: "Int64Counter crea o restituisce un contatore di tipo Int64. Parametri: name, opts. Ritorna: Int64CounterLike (implementazione specifica) e errore."

- `Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (Int64UpDownCounterLike, error)` — crea o restituisce un contatore up/down.
  - Commento: "Int64UpDownCounter crea o restituisce un contatore up/down di tipo Int64."

- `Add(value int64, opts ...metric.AddOption)` — metodo su Int64CounterLike e Int64UpDownCounterLike per modificare la metrica.
  - Commento: "Add incrementa il contatore del valore specificato." (sul counter) / "Add aggiunge (o sottrae se value è negativo) il valore specificato." (sul up/down)

2.3 Tipo 3 (LogCommand e OpType)
<a name="logcommand"></a>

... (file continues)

