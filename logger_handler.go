package loggerhandler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
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

	// Gestione timeout
	timeouts map[string]*time.Timer
	// canale che trasporta lo spanID quando scade un timer
	chTimers chan string

	channel   chan LogCommand
	wg        *sync.WaitGroup
	closeOnce *sync.Once
	mu        *sync.Mutex

	// sincronizzazione per i callback dei timer
	timersWg *sync.WaitGroup
	// flag atomico che indica che il logger è in fase di chiusura
	closing int32

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

// NewLoggerHandler crea e inizializza un nuovo LoggerHandler.
// Cosa fa: alloca le strutture dati, inizializza handler temporaneo e metriche,
//
//	e avvia le goroutine che processano comandi e notifiche di timeout.
//
// Parametri:
//   - logConfig: configurazione per il writer dei log normali
//   - errConfig: configurazione per il writer degli errori
//   - meter: implementazione di MeterInterface per creare metriche
//   - bufferSize: dimensione del canale di comandi
//
// Ritorna: puntatore a LoggerHandler completamente inizializzato
func NewLoggerHandler(logConfig *WriterConfigs, errConfig *WriterConfigs, meter MeterInterface, bufferSize int) *LoggerHandler {
	lh := &LoggerHandler{
		logWriter: logConfig,
		errWriter: errConfig,
		meter:     meter,
		spans:     make(map[string]*SpanLogger),
		// mappa dei timer per span
		timeouts: make(map[string]*time.Timer),
		// canale per notifiche di timeout (trasporta lo spanID)
		chTimers:  make(chan string, 128),
		channel:   make(chan LogCommand, bufferSize),
		wg:        &sync.WaitGroup{},
		closeOnce: &sync.Once{},
		mu:        &sync.Mutex{},
		// timersWg per sincronizzare callback dei timer
		timersWg: &sync.WaitGroup{},
		closing:  0,
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

	// Goroutine che ascolta le notifiche dei timer e chiama Timeout sullo Span
	lh.wg.Add(1)
	go func() {
		defer lh.wg.Done()
		for spanID := range lh.chTimers {
			// prendo lo span sotto mutex e lo invoco
			lh.mu.Lock()
			span := lh.spans[spanID]
			lh.mu.Unlock()

			if span == nil {
				// Se non è stato configurato un Meter (può essere nil), non possiamo aggiornare le metriche.
				// Le metriche sono opzionali: aggiorniamo solo se `lh.meter` è presente.
				if lh.meter != nil {
					lh.invalidSpanCounter.Add(1)
				}
				continue
			}

			// invoca il metodo Timeout sullo SpanLogger
			span.Timeout()
		}
	}()

	return lh
}

// initTempHandler inizializza un handler JSON temporaneo usato per costruire
// la rappresentazione testuale dei record prima di scriverli.
// Cosa fa: crea uno strings.Builder e un JSONHandler che lo usa.
// Parametri: nessuno
// Ritorna: errore in caso di fallimento (attualmente sempre nil)
func (lh *LoggerHandler) initTempHandler() error {
	lh.strBuilder = strings.Builder{}

	lh.tmpHandler = slog.NewJSONHandler(&lh.strBuilder, nil)
	return nil
}

// initMetrics crea le metriche richieste tramite il MeterInterface.
// Cosa fa: richiama i metodi del meter per ottenere i contatori e gli indicatori.
// Parametri: nessuno
// Ritorna: errore se la creazione di una metrica fallisce
func (lh *LoggerHandler) initMetrics() error {
	if lh.meter == nil {
		// Nessun meter configurato, esco
		return nil
	}
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

// GetMeter restituisce l'istanza di MeterInterface associata al LoggerHandler.
// Parametri: nessuno
// Ritorna: MeterInterface
func (lh *LoggerHandler) GetMeter() MeterInterface {
	return lh.meter
}

// GetLogHandler restituisce la configurazione del writer dei log normali.
// Parametri: nessuno
// Ritorna: puntatore a WriterConfigs relativo al log principale
func (lh *LoggerHandler) GetLogHandler() *WriterConfigs {
	return lh.logWriter
}

// GetErrorHandler restituisce la configurazione del writer degli errori.
// Parametri: nessuno
// Ritorna: puntatore a WriterConfigs relativo agli errori
func (lh *LoggerHandler) GetErrorHandler() *WriterConfigs {
	return lh.errWriter
}

// GetSpans restituisce una copia superficiale della mappa degli SpanLogger.
// Cosa fa: copia in modo concorrente-sicuro la mappa interna per evitare race esterne.
// Parametri: nessuno
// Ritorna: mappa (copia superficiale) spanID -> *SpanLogger
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

// GetTotalCounter restituisce il contatore totale degli span.
// Parametri: nessuno
// Ritorna: Int64CounterLike (può essere nil)
func (lh *LoggerHandler) GetTotalCounter() Int64CounterLike {
	return lh.totalCounter
}

// GetSuccessCounter restituisce il contatore dei successi.
// Parametri: nessuno
// Ritorna: Int64CounterLike (può essere nil)
func (lh *LoggerHandler) GetSuccessCounter() Int64CounterLike {
	return lh.successCounter
}

// GetFailureCounter restituisce il contatore dei fallimenti.
// Parametri: nessuno
// Ritorna: Int64CounterLike (può essere nil)
func (lh *LoggerHandler) GetFailureCounter() Int64CounterLike {
	return lh.failureCounter
}

// GetDiscardedCounter restituisce il contatore dei comandi scartati.
// Parametri: nessuno
// Ritorna: Int64CounterLike (può essere nil)
func (lh *LoggerHandler) GetDiscardedCounter() Int64CounterLike {
	return lh.discardedCounter
}

// GetInvalidSpanCounter restituisce il contatore degli span non validi.
// Parametri: nessuno
// Ritorna: Int64CounterLike (può essere nil)
func (lh *LoggerHandler) GetInvalidSpanCounter() Int64CounterLike {
	return lh.invalidSpanCounter
}

// GetActiveSpansGauge restituisce il contatore istantaneo degli span attivi.
// Parametri: nessuno
// Ritorna: Int64UpDownCounterLike (può essere nil)
func (lh *LoggerHandler) GetActiveSpansGauge() Int64UpDownCounterLike {
	return lh.activeSpansGauge
}

// AddSpan crea e registra un nuovo SpanLogger.
// Cosa fa: genera un spanID univoco, crea lo SpanLogger, registra il timer per il timeout (se richiesto)
// Parametri:
//   - duration: durata del timeout dello span (0 per nessun timeout)
//   - tags: lista di tag associati allo span
//   - bufferSize: dimensione del buffer interno dello span
//   - level: livello minimo di log che causa l'invio immediato
//
// Ritorna: puntatore al nuovo SpanLogger
func (lh *LoggerHandler) AddSpan(duration time.Duration, tags []string, bufferSize int, level slog.Level) *SpanLogger {
	var spanID string
	done := false
	for !done {
		spanID, done = lh.generateSpanID()
	}

	span := NewSpanLogger(spanID, duration, tags, bufferSize, lh, level)

	// Aggiorna lo span nella mappa in modo concorrente-sicuro e crea il timer associato
	lh.mu.Lock()
	lh.spans[spanID] = span
	// se è richiesto un timeout > 0 ne creo uno e lo memorizzo
	if duration > 0 {
		// incremento il waitgroup dei timer per rappresentare il timer pianificato
		lh.timersWg.Add(1)
		// uso AfterFunc per notificare tramite chTimers quando scade
		idCopy := spanID
		t := time.AfterFunc(duration, func() {
			// assicuriamoci di fare Done al termine del callback
			defer lh.timersWg.Done()
			// se siamo in fase di chiusura, non inviare
			if atomic.LoadInt32(&lh.closing) == 1 {
				return
			}
			// invio non bloccante dello spanID sul canale dei timer
			select {
			case lh.chTimers <- idCopy:
			default:
			}
		})
		lh.timeouts[spanID] = t
	}
	lh.mu.Unlock()

	// Controllo se è presente un Meter; le metriche sono opzionali quindi aggiorniamo
	// solo se `lh.meter` è non-nil.
	if lh.meter != nil {
		// Aggiorno il contatori metrici se esistono (robustezza nil)
		if lh.activeSpansGauge != nil {
			lh.activeSpansGauge.Add(1)
		}
	}

	return span
}

// generateSpanID genera un nuovo UUID v7 e lo registra temporaneamente nelle mappe.
// Parametri: nessuno
// Ritorna: id string e flag bool che indica se l'id è stato inserito con successo
func (lh *LoggerHandler) generateSpanID() (string, bool) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", false
	}
	idStr := id.String()
	flag := lh.addSpanToMaps(idStr)
	return idStr, flag
}

// addSpanToMaps inserisce un id segnaposto nelle mappe interne per evitare collisioni.
// Cosa fa: sotto mutex controlla l'esistenza e inserisce un placeholder nil nello map degli span.
// Parametri: id string
// Ritorna: bool (true se inserito, false se già esistente)
func (lh *LoggerHandler) addSpanToMaps(id string) bool {
	lh.mu.Lock()

	defer lh.mu.Unlock()
	if _, exists := lh.spans[id]; exists {
		return false
	}
	// L'id è univoco
	// Aggiungilo alla mappa degli span
	lh.spans[id] = nil // Placeholder, lo span verrà creato successivamente

	// Controllo se è presente un Meter; le metriche sono opzionali quindi aggiorniamo
	// solo se `lh.meter` è non-nil.
	if lh.meter != nil {
		// incrementa il contatore degli span attivi (robustezza nil)
		if lh.activeSpansGauge != nil {
			lh.activeSpansGauge.Add(1)
		}
	}
	return true
}

// RemoveSpan rimuove lo span con l'id fornito e ferma il timer associato.
// Parametri: id string
// Ritorna: nulla
func (lh *LoggerHandler) RemoveSpan(id string) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if _, exists := lh.spans[id]; exists {
		// fermo ed elimino il timer associato se presente
		if t, ok := lh.timeouts[id]; ok && t != nil {
			// Stop restituisce true se il timer è fermato prima che scatti;
			// in tal caso dobbiamo bilanciare il timersWg chiamando Done()
			stopped := t.Stop()
			if stopped {
				lh.timersWg.Done()
			}
			delete(lh.timeouts, id)
		}
		delete(lh.spans, id)

		// Controllo se è presente un Meter; le metriche sono opzionali quindi aggiorniamo
		// solo se `lh.meter` è non-nil.
		if lh.meter != nil {
			if lh.activeSpansGauge != nil {
				lh.activeSpansGauge.Add(-1)
			}
		}
	}
}

// AppendCommand prova ad aggiungere un LogCommand al canale interno.
// Cosa fa: invio non bloccante sul canale; se pieno incrementa il contatore dei comandi scartati.
// Parametri: cmd LogCommand
// Ritorna: nulla
func (lh *LoggerHandler) AppendCommand(cmd LogCommand) {
	// Controllo se il canale è pieno
	select {
	case lh.channel <- cmd:
		// Comando aggiunto con successo
		return
	default:
		// Canale pieno, scarto il comando
		// Controllo se è presente un Meter; le metriche sono opzionali quindi aggiorniamo
		// solo se `lh.meter` è non-nil.
		if lh.meter != nil {
			lh.discardedCounter.Add(1)
		}
		return
	}
}

// processCommand processa un singolo LogCommand.
// Cosa fa: verifica che lo span esista, costruisce la stringa di log e scrive sul writer appropriato.
// Parametri: cmd LogCommand
// Ritorna: nulla
func (lh *LoggerHandler) processCommand(cmd LogCommand) {
	// Controllo che lo SpanID esista
	cmd, ok := lh.checkSpanExists(cmd)
	if !ok {
		// Lo SpanID non esiste, il comando è stato modificato per essere un failure
		return
	}

	// Creo la stringa di log nello string builder
	lh.createStrLog(cmd)

	// Scrivo il log in base al tipo di operazione
	lh.writeToHandler(cmd)
}

// createStrLog costruisce la rappresentazione testuale dei record contenuti in LogCommand
// e la pone nello string builder temporaneo.
// Cosa fa: formatta i record usando l'handler JSON temporaneo.
// Parametri: cmd LogCommand
// Ritorna: nulla
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

// writeToHandler scrive la stringa costruita nello string builder sul writer appropriato
// in base al tipo di operazione contenuta nel LogCommand.
// Parametri: cmd LogCommand
// Ritorna: nulla
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

// processOpLog gestisce la scrittura di un normale log (OpLog).
// Parametri: _ string (non usato, mantenuto per compatibilità futura)
// Ritorna: error se la scrittura fallisce, altrimenti nil
func (lh *LoggerHandler) processOpLog(_ string) error {
	// Scrivo sul log handler la stringa presente nello string builder
	_, err := fmt.Fprintln(lh.logWriter.GetMultiWriter(), lh.strBuilder.String())
	if err != nil {
		return err
	}
	return nil
}

// processOpFailure gestisce la chiusura dello span con failure (OpReleaseFailure).
// Cosa fa: aggiorna metriche, rimuove lo span e scrive il log di errore.
// Parametri: spanId string
// Ritorna: error se la scrittura fallisce, altrimenti nil
func (lh *LoggerHandler) processOpFailure(spanId string) error {
	// Controllo se è presente un Meter; le metriche sono opzionali quindi aggiorniamo
	// solo se `lh.meter` è non-nil`.
	if lh.meter != nil {
		// Aggiorno il contatore dei fallimenti
		lh.failureCounter.Add(1)
		// Aggiorno il contatore totale
		lh.totalCounter.Add(1)
	}

	// Rimuovo gli span completati con successo dalla mappa
	lh.RemoveSpan(spanId)

	_, err := fmt.Fprintln(lh.errWriter.GetMultiWriter(), lh.strBuilder.String())
	if err != nil {
		return err
	}

	return nil
}

// processOpSuccess gestisce la chiusura dello span con successo (OpReleaseSuccess).
// Cosa fa: aggiorna metriche, rimuove lo span e scrive il log se presente.
// Parametri: spanId string
// Ritorna: error se la scrittura fallisce, altrimenti nil
func (lh *LoggerHandler) processOpSuccess(spanId string) error {
	// Controllo se è presente un Meter; le metriche sono opzionali quindi aggiorniamo
	// solo se `lh.meter` è non-nil`.
	if lh.meter != nil {
		// Incremento il contatore dei successi
		lh.successCounter.Add(1)
		// Incremento il contatore totale
		lh.totalCounter.Add(1)
	}

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

// processOpTimeout gestisce la chiusura dello span per timeout (OpTimeout).
// Cosa fa: aggiorna metriche, rimuove lo span e scrive il log di errore.
// Parametri: spanId string
// Ritorna: error se la scrittura fallisce, altrimenti nil
func (lh *LoggerHandler) processOpTimeout(spanId string) error {
	// Controllo se è presente un Meter; le metriche sono opzionali quindi aggiorniamo
	// solo se `lh.meter` è non-nil`.
	if lh.meter != nil {
		// Aggiorno il contatore dei fallimenti
		lh.failureCounter.Add(1)
		// Aggiorno il contatore totale
		lh.totalCounter.Add(1)
	}

	// Rimuovo gli span completati con successo dalla mappa
	lh.RemoveSpan(spanId)

	_, err := fmt.Fprintln(lh.errWriter.GetMultiWriter(), lh.strBuilder.String())
	if err != nil {
		return err
	}

	return nil
}

// checkSpanExists verifica che lo SpanID nel LogCommand esista nella mappa degli span.
// Cosa fa: se lo span non esiste modifica il LogCommand per trasformarlo in OpReleaseFailure
//
//	e aggiunge un record di errore.
//
// Parametri: cmd LogCommand
// Ritorna: (LogCommand, bool) -> il comando (eventualmente modificato) e true se lo span esiste,
//
//	false se non esiste (in quel caso il comando è trasformato in failure)
func (lh *LoggerHandler) checkSpanExists(cmd LogCommand) (LogCommand, bool) {
	lh.mu.Lock()
	_, exists := lh.spans[cmd.SpanID]
	lh.mu.Unlock()

	if !exists {
		// Lo SpanID non esiste
		// Controllo se è presente un Meter; le metriche sono opzionali quindi aggiorniamo
		// solo se `lh.meter` è non-nil`.
		if lh.meter != nil {
			if cmd.Op == OpTimeout {
				// Non conto i timeout come span non validi
				// la chiusura dello span è già avvenuta poichè non presente in mappa degli Span
				return cmd, false
			}
			// Incremento il contatore degli span non validi (robustezza nil)
			lh.invalidSpanCounter.Add(1)
		}
		// Creo un record di errore
		errRecord := slog.NewRecord(time.Now(), slog.LevelError, "SpanID non trovato: "+cmd.SpanID, 0)
		// Aggiungo il record al log
		cmd.Records = append(cmd.Records, errRecord)
		// Forzo l'operazione a essere un failure
		cmd.Op = OpReleaseFailure
	}
	return cmd, true
}

// Close ferma tutti i timer, chiude i canali e aspetta la terminazione delle goroutine.
// Parametri: nessuno
// Ritorna: nulla
func (lh *LoggerHandler) Close() {
	lh.closeOnce.Do(func() {
		// segnalo che stiamo chiudendo per le callback dei timer
		atomic.StoreInt32(&lh.closing, 1)

		// fermo tutti i timer e svuoto la mappa
		lh.mu.Lock()
		for id, t := range lh.timeouts {
			if t != nil {
				stopped := t.Stop()
				if stopped {
					// se abbiamo fermato il timer prima che scattasse, bilanciamo il waitgroup
					lh.timersWg.Done()
				}
			}
			delete(lh.timeouts, id)
		}
		lh.timeouts = make(map[string]*time.Timer)
		lh.mu.Unlock()

		// aspettiamo che eventuali callback in esecuzione terminino
		lh.timersWg.Wait()

		// ora è sicuro chiudere il canale dei timer (nessun callback invierà)
		close(lh.chTimers)

		// segnalo alle goroutine di fermarsi
		close(lh.channel)
		// aspetto che le goroutine finiscano
		lh.wg.Wait()
	})
}
