package loggerhandler

import (
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"
)

type LoggerHandler struct {
	handler slog.Handler
	meter   MeterInterface
	spans   map[string]*SpanLogger

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

func NewLoggerHandler(handler slog.Handler, meter MeterInterface, bufferSize int) *LoggerHandler {
	lh := &LoggerHandler{
		handler:   handler,
		meter:     meter,
		spans:     make(map[string]*SpanLogger),
		channel:   make(chan LogCommand, bufferSize),
		wg:        &sync.WaitGroup{},
		closeOnce: &sync.Once{},
		mu:        &sync.Mutex{},
	}

	err := lh.initMetrics()
	if err != nil {
		panic("Errore nell'inizializzazione delle metriche: " + err.Error())
	}

	lh.wg.Add(1)
	go func() {
		defer lh.wg.Done()
		for cmd := range lh.channel {
			lh.processLogCommand(cmd)
		}
	}()

	return lh
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

func (lh *LoggerHandler) GetHandler() slog.Handler {
	return lh.handler
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

func (lh *LoggerHandler) AppendLogCommand(cmd LogCommand) {
	lh.channel <- cmd
}

func (lh *LoggerHandler) processLogCommand(cmd LogCommand) {

}
