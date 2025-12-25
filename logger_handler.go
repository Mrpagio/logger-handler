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
	// Contatori cumulativi
	totalCounter Int64CounterLike
	// Contatori di successo
	successCounter Int64CounterLike
	// Contatori di fallimento
	failureCounter Int64CounterLike
	// Contatori di LogCommand scartati perchè il buffer era pieno
	discardedCounter Int64CounterLike
	// Contatori di LogCommand con Span scaduti o non presenti
	invalidSpanCounter Int64CounterLike

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
	// Copia superficiale protetta dal mutex per evitare race esterne
	lh.mu.Lock()
	defer lh.mu.Unlock()
	cp := make(map[string]*SpanLogger, len(lh.spans))
	for k, v := range lh.spans {
		cp[k] = v
	}
	return cp
}

func (lh *LoggerHandler) GetTotalCounter() Int64CounterLike {
	return lh.totalCounter
}

func (lh *LoggerHandler) GetSuccessCounter() Int64CounterLike {
	return lh.successCounter
}

func (lh *LoggerHandler) GetFailureCounter() Int64CounterLike {
	return lh.failureCounter
}

func (lh *LoggerHandler) GetDiscardedCounter() Int64CounterLike {
	return lh.discardedCounter
}

func (lh *LoggerHandler) GetInvalidSpanCounter() Int64CounterLike {
	return lh.invalidSpanCounter
}

func (lh *LoggerHandler) GetActiveSpansGauge() Int64UpDownCounterLike {
	return lh.activeSpansGauge
}

func (lh *LoggerHandler) AddSpan(duration time.Duration, tags []string, bufferSize int, level slog.Level) *SpanLogger {
	var spanID string
	done := false
	for !done {
		spanID, done = lh.generateSpanID()
	}

	span := NewSpanLogger(spanID, duration, tags, bufferSize, lh, level)

	// Aggiorna lo span nella mappa in modo concorrente-sicuro
	lh.mu.Lock()
	lh.spans[spanID] = span
	lh.mu.Unlock()

	// Aggiorno il contatori metrici se esistono (robustezza nil)
	if lh.totalCounter != nil {
		// non modifichiamo qui il totale, serve solo come esempio se volessimo
	}

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
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if _, exists := lh.spans[id]; exists {
		return false
	}
	// L'id è univoco
	// Aggiungilo alla mappa degli span
	lh.spans[id] = nil // Placeholder, lo span verrà creato successivamente

	// incrementa il contatore degli span attivi (robustezza nil)
	if lh.activeSpansGauge != nil {
		lh.activeSpansGauge.Add(1)
	}
	return true
}

func (lh *LoggerHandler) RemoveSpan(id string) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if _, exists := lh.spans[id]; exists {
		delete(lh.spans, id)

		if lh.activeSpansGauge != nil {
			lh.activeSpansGauge.Add(-1)
		}
	}
}

func (lh *LoggerHandler) AppendCommand(cmd LogCommand) {
	// Controllo se il canale è pieno
	select {
	case lh.channel <- cmd:
		// Comando aggiunto con successo
		return
	default:
		// Canale pieno, scarto il comando
		lh.discardedCounter.Add(1)
		return
	}
}

func (lh *LoggerHandler) processCommand(cmd LogCommand) {
	// Controllo che lo SpanID esista
	cmd = lh.checkSpanExists(cmd)

	// Creo la stringa di log nello string builder
	lh.createStrLog(cmd)

	// Scrivo il log in base al tipo di operazione
	lh.writeToHandler(cmd)
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

func (lh *LoggerHandler) writeToHandler(cmd LogCommand) {
	var err error
	switch cmd.Op {
	case OpLog:
		err = lh.processOpLog(cmd.SpanID)
		if err != nil {
			return
		}
	case OpReleaseSuccess:
		err = lh.processOpSuccess(cmd.SpanID)
		if err != nil {
			return
		}
	case OpReleaseFailure:
		err = lh.processOpFailure(cmd.SpanID)
		if err != nil {
			return
		}
	case OpTimeout:
		err = lh.processOpTimeout(cmd.SpanID)
		if err != nil {
			return
		}
	}
	return
}

func (lh *LoggerHandler) processOpLog(spanId string) error {
	// Scrivo sul log handler la stringa presente nello string builder
	_, err := fmt.Fprintln(lh.logWriter.GetMultiWriter(), lh.strBuilder.String())
	if err != nil {
		return err
	}
	return nil
}

func (lh *LoggerHandler) processOpFailure(spanId string) error {
	// Aggiorno il contatore dei fallimenti
	lh.failureCounter.Add(1)
	// Aggiorno il contatore totale
	lh.totalCounter.Add(1)

	// Rimuovo gli span completati con successo dalla mappa
	lh.RemoveSpan(spanId)

	_, err := fmt.Fprintln(lh.errWriter.GetMultiWriter(), lh.strBuilder.String())
	if err != nil {
		return err
	}

	return nil
}

func (lh *LoggerHandler) processOpSuccess(spanId string) error {
	// Incremento il contatore dei successi
	lh.successCounter.Add(1)
	// Incremento il contatore totale
	lh.totalCounter.Add(1)

	// Rimuovo gli span completati con successo dalla mappa
	lh.RemoveSpan(spanId)

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

func (lh *LoggerHandler) processOpTimeout(spanId string) error {
	// Aggiorno il contatore dei fallimenti
	lh.failureCounter.Add(1)
	// Aggiorno il contatore totale
	lh.totalCounter.Add(1)

	// Rimuovo gli span completati con successo dalla mappa
	lh.RemoveSpan(spanId)

	_, err := fmt.Fprintln(lh.errWriter.GetMultiWriter(), lh.strBuilder.String())
	if err != nil {
		return err
	}

	return nil
}

func (lh *LoggerHandler) checkSpanExists(cmd LogCommand) LogCommand {
	lh.mu.Lock()
	_, exists := lh.spans[cmd.SpanID]
	lh.mu.Unlock()

	if !exists {
		// Lo SpanID non esiste
		// Incremento il contatore degli span non validi (robustezza nil)
		if lh.invalidSpanCounter != nil {
			lh.invalidSpanCounter.Add(1)
		}
		// Creo un record di errore
		errRecord := slog.NewRecord(time.Now(), slog.LevelError, "SpanID non trovato: "+cmd.SpanID, 0)
		// Aggiungo il record al log
		cmd.Records = append(cmd.Records, errRecord)
		// Forzo l'operazione a essere un failure
		cmd.Op = OpReleaseFailure
	}
	return cmd
}

func (lh *LoggerHandler) Close() {
	lh.closeOnce.Do(func() {
		close(lh.channel)
		lh.wg.Wait()
	})
}
