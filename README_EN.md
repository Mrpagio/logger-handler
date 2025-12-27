# Logger Handler

Index:

- [1 General description](#general-description)
- [2 Types](#types)
  - [2.1 Type 1 (WriterConfigs)](#writerconfigs)
    - [2.1.1 Fields](#writerconfigs)
    - [2.1.2 Methods and functions](#writerconfigs)
      - [2.1.2.1 `NewLogConfigs` (constructor)](#writerconfigs)
      - [2.1.2.2 `toMulti`](#writerconfigs)
      - [2.1.2.3 `initTextHandler`](#writerconfigs)
      - [2.1.2.4 `GetTextHandler`](#writerconfigs)
      - [2.1.2.5 `GetMultiWriter`](#writerconfigs)
      - [2.1.2.6 `IsConsoleLoggingEnabled`](#writerconfigs)
      - [2.1.2.7 `GetFileLocation`](#writerconfigs)
      - [2.1.2.8 `GetFileMaxSize`](#writerconfigs)
      - [2.1.2.9 `GetFileMaxBackups`](#writerconfigs)
      - [2.1.2.10 `IsCompressEnabled`](#writerconfigs)
  - [2.2 Type 2 (MeterInterface + Int64CounterLike + Int64UpDownCounterLike)](#meterinterface)
    - [2.2.1 Fields / Description](#meterinterface)
    - [2.2.2 Methods and functions](#meterinterface)
      - [2.2.2.1 `Int64Counter` (MeterInterface)](#meterinterface)
      - [2.2.2.2 `Int64UpDownCounter` (MeterInterface)](#meterinterface)
      - [2.2.2.3 `Add` (Int64CounterLike)](#meterinterface)
      - [2.2.2.4 `Add` (Int64UpDownCounterLike)](#meterinterface)
  - [2.3 Type 3 (LogCommand and OpType)](#logcommand)
    - [2.3.1 Fields](#logcommand)
    - [2.3.2 Methods](#logcommand)
      - [2.3.2.1 `TypeString`](#logcommand)
  - [2.4 Type 4 (LoggerHandler)](#loggerhandler)
    - [2.4.1 Fields](#loggerhandler)
    - [2.4.2 Methods and functions](#loggerhandler)
      - [2.4.2.1 `NewLoggerHandler` (constructor)](#loggerhandler)
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
  - [2.5 Type 5 (SpanLogger)](#spanlogger)
    - [2.5.1 Fields](#spanlogger)
    - [2.5.2 Methods and functions](#spanlogger)
      - [2.5.2.1 `NewSpanLogger` (constructor)](#spanlogger)
      - [2.5.2.2 `GetID`, `GetDuration`, `GetTags`, `GetBufferSize`, `GetLoggerHandler`, `GetLogLevel`](#spanlogger)
      - [2.5.2.3 `sendLogCmd`, `Debug`, `Info`, `Warn`, `Error`, `ReleaseSuccess`, `Timeout`](#spanlogger)
- [3 Examples](#examples)

---

<a name="general-description"></a>
1 General description

Logger Handler is a small Go library that provides a "span" mechanism to group log records, asynchronous log
submission, management of span timeouts, and optional integration with a `MeterInterface` (to create
OpenTelemetry-like metrics). The library does not force a direct dependency on OTel: it uses interfaces
(placeholders) for metric instruments.

<a name="types"></a>
2 Types

2.1 Type 1 (WriterConfigs)
<a name="writerconfigs"></a>

2.1.1 Fields

- `consoleLogging bool` — If true, console logging is enabled
- `fileLocation string` — Path to the log file. If empty, file logging is not used
- `fileMaxSize int` — MB before rotation
- `fileMaxBackups int` — Number of backup files to keep
- `compress bool` — If true, rotated backup files are gzipped
- `multi io.Writer` — combined writer (stdout + file if configured)
- `textHandler *slog.TextHandler` — slog TextHandler for textual output

2.1.2 Methods and functions

- `NewLogConfigs(consoleLogging bool, fileLocation string, fileMaxSize int, fileMaxBackups int, compress bool) *WriterConfigs` — Constructor
  - Comment: "NewLogConfigs creates and initializes a WriterConfigs struct. It sets fields according to the parameters, creates the io.MultiWriter and initializes the TextHandler."

- `toMulti()` — builds the io.MultiWriter according to the configuration.
  - Comment: "toMulti builds the io.MultiWriter based on the configuration. It adds stdout and/or a file writer (lumberjack) to the MultiWriter."

- `initTextHandler()` — initializes a slog TextHandler using the MultiWriter.
  - Comment: "initTextHandler initializes a slog TextHandler using the MultiWriter."

- `GetTextHandler() *slog.TextHandler`
  - Comment: "GetTextHandler returns the internal TextHandler."

- `GetMultiWriter() io.Writer`
  - Comment: "GetMultiWriter returns the combined io.Writer."

- `IsConsoleLoggingEnabled() bool`
  - Comment: "IsConsoleLoggingEnabled indicates whether console logging is enabled."

- `GetFileLocation() string`
  - Comment: "GetFileLocation returns the configured log file path."

- `GetFileMaxSize() int`
  - Comment: "GetFileMaxSize returns the configured max file size in MB."

- `GetFileMaxBackups() int`
  - Comment: "GetFileMaxBackups returns the number of configured backups."

- `IsCompressEnabled() bool`
  - Comment: "IsCompressEnabled indicates whether backup compression is enabled."

2.2 Type 2 (MeterInterface + Int64CounterLike + Int64UpDownCounterLike)
<a name="meterinterface"></a>

2.2.1 Fields / Description

- `MeterInterface` is an interface that exposes methods to obtain metric instruments (Int64Counter and Int64UpDownCounter).
  - Comment: "MeterInterface exposes methods used to create metric instruments. It provides a minimal abstraction over factories so the package does not depend directly on a specific OpenTelemetry implementation."

- `Int64CounterLike` — interface with method `Add(value int64, opts ...metric.AddOption)`
  - Comment: "Int64CounterLike is a placeholder type exposing Add for counters. It allows calling Add without depending on a real implementation."

- `Int64UpDownCounterLike` — interface with method `Add(value int64, opts ...metric.AddOption)`
  - Comment: "Int64UpDownCounterLike is a placeholder type exposing Add for up/down counters. It allows incrementing or decrementing an instantaneous metric."

2.2.2 Methods and functions

- `Int64Counter(name string, opts ...metric.InstrumentOption) (Int64CounterLike, error)` — creates or returns an Int64 counter.
  - Comment: "Int64Counter creates or returns an Int64 counter. Parameters: name, opts. Returns: Int64CounterLike (implementation) and error."

- `Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (Int64UpDownCounterLike, error)` — creates or returns an up/down counter.
  - Comment: "Int64UpDownCounter creates or returns an Int64 up/down counter."

- `Add(value int64, opts ...metric.AddOption)` — method on Int64CounterLike and Int64UpDownCounterLike to modify the metric.
  - Comment: "Add increments the counter by the specified value." (on counter) / "Add adds (or subtracts if value is negative) the specified value." (on up/down)

2.3 Type 3 (LogCommand and OpType)
<a name="logcommand"></a>

2.3.1 Fields

- `LogCommand`:
  - `Op OpType` — operation type
  - `SpanID string` — span identifier
  - `Records []slog.Record` — accumulated records
  - `Err error` — optional error (used for OpReleaseFailure)

2.3.2 Methods

- `TypeString() string` — returns a textual representation of the operation type.
  - Comment: "TypeString returns a textual representation of the operation type."

2.4 Type 4 (LoggerHandler)
<a name="loggerhandler"></a>

2.4.1 Fields

- `logWriter *WriterConfigs` — writer for normal logs
- `errWriter *WriterConfigs` — writer for error logs
- `tmpHandler *slog.JSONHandler` — temporary handler to build JSON strings
- `strBuilder strings.Builder` — builder for the log string
- `meter MeterInterface` — interface for metrics (may be nil)
- `spans map[string]*SpanLogger` — map of active spans
- `timeouts map[string]*time.Timer` — timers for spans with timeout
- `chTimers chan string` — channel for timer notifications
- `channel chan LogCommand` — command channel
- `wg *sync.WaitGroup`, `closeOnce *sync.Once`, `mu *sync.Mutex` — synchronization primitives
- `timersWg *sync.WaitGroup` — synchronization for timer callbacks
- `closing int32` — atomic flag for closing
- metrics (all optional): `totalCounter`, `successCounter`, `failureCounter`, `discardedCounter`, `invalidSpanCounter`, `activeSpansGauge`

2.4.2 Methods and functions

- `NewLoggerHandler(logConfig *WriterConfigs, errConfig *WriterConfigs, meter MeterInterface, bufferSize int) *LoggerHandler` — creates and initializes a new LoggerHandler.
  - Comment: "NewLoggerHandler creates and initializes a new LoggerHandler. It allocates data structures, initializes the temporary handler and metrics, and starts goroutines that process commands and timer notifications. Params: logConfig, errConfig, meter, bufferSize. Returns: pointer to a fully initialized LoggerHandler."

- `initTempHandler()` — initializes a temporary JSON handler used to build textual representations.
  - Comment: "initTempHandler initializes a temporary JSON handler used to build textual representations of records before writing them."

- `initMetrics()` — creates required metrics through the MeterInterface.
  - Comment: "initMetrics creates required metrics through the MeterInterface. It calls the meter methods to obtain counters and instruments. Returns: error if metric creation fails."

- `GetMeter()` — returns the associated MeterInterface instance.
  - Comment: "GetMeter returns the MeterInterface instance associated with the LoggerHandler."

- `GetLogHandler()` — returns the log writer configuration.
  - Comment: "GetLogHandler returns the WriterConfigs for normal logs."

- `GetErrorHandler()` — returns the error writer configuration.
  - Comment: "GetErrorHandler returns the WriterConfigs for error logs."

- `GetSpans()` — returns a shallow copy of the span map.
  - Comment: "GetSpans returns a shallow copy of the internal span map. It does a concurrent-safe copy to avoid external races."

- `GetTotalCounter()`, `GetSuccessCounter()`, `GetFailureCounter()`, `GetDiscardedCounter()`, `GetInvalidSpanCounter()`, `GetActiveSpansGauge()` — getters for metrics (may be nil).

- `AddSpan(duration time.Duration, tags []string, bufferSize int, level slog.Level) *SpanLogger` — creates and registers a new SpanLogger.
  - Comment: "AddSpan creates and registers a new SpanLogger. It generates a unique span ID, creates the SpanLogger, and registers a timer for timeout (if requested)."
  - Note: updates to `activeSpansGauge` happen only if `lh.meter` is not nil and the instrument exists (nil-robustness).

- `RemoveSpan(id string)` — removes the span and stops its timer.
  - Comment: "RemoveSpan removes the span with the given ID and stops its associated timer."
  - Note: decrement of `activeSpansGauge` is conditioned on `if lh.meter != nil` and additional nil checks on the instrument.

- `AppendCommand(cmd LogCommand)` — attempts to add a LogCommand to the internal channel.
  - Comment: "AppendCommand tries to add a LogCommand to the internal channel. It does a non-blocking send; if the channel is full it increments the discarded commands counter."
  - Note: increment of `discardedCounter` is protected by `if lh.meter != nil`.

- `processCommand`, `createStrLog`, `writeToHandler` — internal functions that process commands, build textual representations and write to the appropriate writers.
  - Comments: present in code; e.g. `createStrLog`: "builds the textual representation of the records contained in a LogCommand and places it in the temporary string builder."

- `processOpLog`, `processOpSuccess`, `processOpFailure`, `processOpTimeout` — handle various release/operation types.
  - Comments: e.g. `processOpSuccess`: "handles closing a span with success (OpReleaseSuccess). It updates metrics, removes the span and writes the log if present." 
  - Important note: all modifications to counters (`totalCounter`, `successCounter`, `failureCounter`) are performed inside `if lh.meter != nil { ... }` to handle the case where the `meter` passed is `nil`. Relevant comments are kept inside those blocks.

- `checkSpanExists(cmd LogCommand) (LogCommand, bool)` — verifies that the SpanID in a LogCommand exists in the span map.
  - Comment: "checkSpanExists verifies that the SpanID in the LogCommand exists in the internal span map. If not, it transforms the LogCommand into an OpReleaseFailure and adds an error record." 
  - Note: increment of `invalidSpanCounter` is conditioned on `if lh.meter != nil`.

- `Close()` — stops all timers, closes channels and waits for goroutines to finish.
  - Comment: "Close stops all timers, closes channels and waits for goroutine termination."

2.5 Type 5 (SpanLogger)
<a name="spanlogger"></a>

2.5.1 Fields

- `id string`
- `timeDuration time.Duration`
- `tags []string`
- `bufferSize int`
- `buffer []slog.Record`
- `loggerHandler *LoggerHandler`
- `logLevel slog.Level`

2.5.2 Methods and functions

- `NewSpanLogger(id string, duration time.Duration, tags []string, bufferSize int, loggerHandler *LoggerHandler, level slog.Level) *SpanLogger` — constructor.
  - Comment: "NewSpanLogger creates a new SpanLogger. It constructs and returns a new SpanLogger with provided parameters."

- Getters: `GetID`, `GetDuration`, `GetTags`, `GetBufferSize`, `GetLoggerHandler`, `GetLogLevel` — comments in code explain the returns.

- `sendLogCmd(op OpType, err error)` — builds and sends a LogCommand to the LoggerHandler. (used internally by level methods)
  - Comment: "sendLogCmd builds and sends a LogCommand to the LoggerHandler. It creates a LogCommand with current records, sends it via AppendCommand and clears the buffer."

- Levels: `Debug`, `Info`, `Warn`, `Error` — add a record to the buffer and send the command if the level requires it.
  - Comments: present in code (e.g. `Debug`: "adds a debug record to the buffer and, if the level requires it, sends the command.")

- `ReleaseSuccess()` — sends a release success command (OpReleaseSuccess).
  - Comment: "ReleaseSuccess sends an OpReleaseSuccess command."

- `Timeout()` — generates a timeout record, appends it to the buffer and sends OpTimeout.
  - Comment: "Timeout generates a timeout error record, appends it to the buffer and sends OpTimeout."

3 Examples
<a name="examples"></a>

The `example/` folder contains a minimal example showing usage without real OpenTelemetry dependencies: `example/main.go`.

Brief example description:

- Creates `WriterConfigs` for logs and errors (console enabled).
- Creates a minimal `fakeMeter` that implements `MeterInterface` returning fake counters.
- Creates a `LoggerHandler` with `NewLoggerHandler(..., meter, ...)` and then creates a span (`AddSpan`).
- Sends some logs (`Info`, `Debug`) and closes the span with `ReleaseSuccess()`.

Addition: `meter == nil` example

- Path: `example/meter_nil/main.go`
- Purpose: demonstrates behavior when passing `nil` as `meter` to `NewLoggerHandler`. Metrics are optional and the code is robust to a nil meter: no calls to methods on a nil meter are made without first checking `if lh.meter != nil`.
- What the example prints when run:
  - It prints `Meter is nil? true` and that returned counters are nil.
  - It performs normal span operations (Info + ReleaseSuccess) and writes logs to console, without updating metrics.

Quick commands

Run tests (direct):

```bash
# from module root
go test -v ./...
```

Run tests (script that saves output into test/output):

```bash
bash test/run_all_tests.sh
```

Run the main example:

```bash
go run ./example
```

Run the `meter == nil` example:

```bash
go run ./example/meter_nil
```

---

Notes

- All parts of the code that modify cumulative metrics (`totalCounter`, `successCounter`, `failureCounter`, `discardedCounter`, `invalidSpanCounter`) and the instantaneous indicator `activeSpansGauge` are protected by `if lh.meter != nil { ... }` to avoid panics when the meter is nil. The comments in code related to these checks were kept adjacent to the operations.
- Test suites were not modified.

If you want, I can:
- Add more complete examples in `example/` (timeout, nil meter, etc.)
- Produce a more compact or alternative README version

