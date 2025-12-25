package loggerhandler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"
)

type LoggerHandler struct {
	logWriter *WriterConfigs
	errWriter *WriterConfigs

	tmpHandler *slog.JSONHandler
	strBuilder strings.Builder

	meter MeterInterface
	spans map[string]*SpanLogger

	channel   chan LogCommand
	wg        *sync.WaitGroup
	closeOnce *sync.Once
	mu        *sync.Mutex

	// Metriche Otel per Report Temporali
	totalCounter   Int64CounterLike
	successCounter Int64CounterLike
	failureCounter Int64CounterLike

	// Indicatori istantanei
	activeSpansGauge Int64UpDownCounterLike
}

func NewLoggerHandler(logConfig *WriterConfigs, errConfig *WriterConfigs, meter MeterInterface, bufferSize int) *LoggerHandler {
	lh := &LoggerHandler{
		logWriter: logConfig,
		errWriter: errConfig,
		meter:     meter,
		spans:     make(map[string]*SpanLogger),
		channel:   make(chan LogCommand, bufferSize),
		wg:        &sync.WaitGroup{},
		closeOnce: &sync.Once{},
		mu:        &sync.Mutex{},
	}

	// Creo un handler temporaneo per la scrittura su buffer
	err := lh.initTempHandler()
	if err != nil {
		panic("Errore nell'inizializzazione dell'handler temporaneo: " + err.Error())
	}

	// Inizializzo le metriche
	err = lh.initMetrics()
	if err != nil {
		panic("Errore nell'inizializzazione delle metriche: " + err.Error())
	}

	lh.wg.Add(1)
	go func() {
		defer lh.wg.Done()
		for cmd := range lh.channel {
			lh.processCommand(cmd)
		}
	}()

	return lh
}

func (lh *LoggerHandler) initTempHandler() error {
	lh.strBuilder = strings.Builder{}

	lh.tmpHandler = slog.NewJSONHandler(&lh.strBuilder, nil)
	return nil
}

func (lh *LoggerHandler) initMetrics() error {
	var err error
	lh.totalCounter, err = lh.meter.Int64Counter("logger_total_spans", metric.WithDescription("Somma totale degli span creati"))
	if err != nil {
		return err
	}
	lh.successCounter, err = lh.meter.Int64Counter("logger_success_spans", metric.WithDescription("Somma totale degli span completati con successo"))
	if err != nil {
		return err
	}
	lh.failureCounter, err = lh.meter.Int64Counter("logger_failure_spans", metric.WithDescription("Somma totale degli span completati con errore"))
	if err != nil {
		return err
	}
	lh.activeSpansGauge, err = lh.meter.Int64UpDownCounter("logger_active_spans", metric.WithDescription("Contatore degli span attivi"))
	if err != nil {
		return err
	}
	return nil
}

func (lh *LoggerHandler) GetMeter() MeterInterface {
	return lh.meter
}

func (lh *LoggerHandler) GetLogHandler() *WriterConfigs {
	return lh.logWriter
}

func (lh *LoggerHandler) GetErrorHandler() *WriterConfigs {
	return lh.errWriter
}

func (lh *LoggerHandler) GetSpans() map[string]*SpanLogger {
	return lh.spans
}

func (lh *LoggerHandler) AddSpan(duration time.Duration, tags []string, bufferSize int, level slog.Level) *SpanLogger {
	var spanID string
	done := false
	for !done {
		spanID, done = lh.generateSpanID()
	}

	span := NewSpanLogger(spanID, duration, tags, bufferSize, lh, level)

	// Aggiorna lo span nella mappa
	lh.spans[spanID] = span

	// Aggiorno il contatori metrici se esistono

	return span
}

// Crea un id univoco per lo span usando google/uuid v7
func (lh *LoggerHandler) generateSpanID() (string, bool) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", false
	}
	idStr := id.String()
	flag := lh.addSpanToMap(idStr)
	return idStr, flag
}

func (lh *LoggerHandler) addSpanToMap(id string) bool {
	if _, exists := lh.spans[id]; exists {
		return false
	}
	// L'id è univoco
	// Aggiungilo alla mappa degli span
	lh.spans[id] = nil // Placeholder, lo span verrà creato successivamente

	// incrementa il contatore degli span attivi
	lh.activeSpansGauge.Add(1)
	return true
}

func (lh *LoggerHandler) RemoveSpan(id string) {
	if _, exists := lh.spans[id]; exists {
		delete(lh.spans, id)

		lh.activeSpansGauge.Add(-1)
	}
}

func (lh *LoggerHandler) AppendCommand(cmd LogCommand) {
	lh.channel <- cmd
}

func (lh *LoggerHandler) processCommand(cmd LogCommand) {
	lh.createStrLog(cmd)
	// Scrivo il log in base al tipo di operazione
	lh.writeToHandler(cmd.Op)
}

func (lh *LoggerHandler) createStrLog(cmd LogCommand) {
	// Se non ci sono record, esco
	if len(cmd.Records) == 0 {
		return
	}

	// Al momento non supporta context.Context, ne creo uno fittizio
	ctx := context.TODO()

	// Resetto lo string builder
	lh.strBuilder.Reset()
	// Aggiungo una riga di separazione allo string builder
	lh.strBuilder.WriteString("----- OpType: " + cmd.TypeString() + " ----- Span ID: " + cmd.SpanID + " -----\n")
	lastTimestamp := time.Now()

	// Ciclo sui record
	for _, record := range cmd.Records {
		// Aggiungo il record al log
		lastTimestamp = record.Time
		err := lh.tmpHandler.Handle(ctx, record)
		if err != nil {
			// Gestisco l'errore (al momento lo ignoro)
			continue
		}
	}

	if cmd.Op == OpReleaseFailure && cmd.Err != nil {
		// Creo un record per l'errore
		errRecord := slog.NewRecord(lastTimestamp, slog.LevelError, "Errore nello span: "+cmd.Err.Error(), 0)
		// Aggiungo il record al log
		_ = lh.tmpHandler.Handle(ctx, errRecord)
	}

	// Aggiungo una riga di chiusura allo string builder
	lh.strBuilder.WriteString("--------------------")
}

func (lh *LoggerHandler) writeToHandler(op OpType) {
	var err error
	switch op {
	case OpLog:
		err = lh.processOpLog()
		if err != nil {
			return
		}
	case OpReleaseSuccess:
		err = lh.processOpSuccess()
		if err != nil {
			return
		}
	case OpReleaseFailure:
		err = lh.processOpFailure()
		if err != nil {
			return
		}
	case OpTimeout:
		err = lh.processOpTimeout()
		if err != nil {
			return
		}
	}
	return
}

func (lh *LoggerHandler) processOpLog() error {
	// Scrivo sul log handler la stringa presente nello string builder
	_, err := fmt.Fprintln(lh.logWriter.GetMultiWriter(), lh.strBuilder.String())
	if err != nil {
		return err
	}
	return nil
}

func (lh *LoggerHandler) processOpFailure() error {
	_, err := fmt.Fprintln(lh.errWriter.GetMultiWriter(), lh.strBuilder.String())
	if err != nil {
		return err
	}
	// Aggiorno il contatore dei fallimenti
	lh.failureCounter.Add(1)
	// Aggiorno il contatore totale
	lh.totalCounter.Add(1)
	return nil
}

func (lh *LoggerHandler) processOpSuccess() error {
	// Incremento il contatore dei successi
	lh.successCounter.Add(1)
	// Incremento il contatore totale
	lh.totalCounter.Add(1)
	// Se la lunghezza dello string builder è zero, non scrivo nulla
	if lh.strBuilder.Len() == 0 {
		return nil
	}

	// Scrivo sul log handler la stringa presente nello string builder
	_, err := fmt.Fprintln(lh.logWriter.GetMultiWriter(), lh.strBuilder.String())
	if err != nil {
		return err
	}
	return nil
}

func (lh *LoggerHandler) processOpTimeout() error {
	_, err := fmt.Fprintln(lh.errWriter.GetMultiWriter(), lh.strBuilder.String())
	if err != nil {
		return err
	}
	// Aggiorno il contatore dei fallimenti
	lh.failureCounter.Add(1)
	// Aggiorno il contatore totale
	lh.totalCounter.Add(1)
	return nil
}
