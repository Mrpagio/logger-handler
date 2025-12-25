package loggerhandler

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"log/slog"
	"os"
)

type WriterConfigs struct {
	consoleLogging bool   // Se true, si usa il logging su console
	fileLocation   string // Percorso del file di log. Se vuoto, non si usa il file logging
	fileMaxSize    int    // MB prima di fare la rotazione
	fileMaxBackups int    // Numero di file di backup da mantenere
	compress       bool   // Se true, i file di log di backup vengono compressi in gzip
	multi          io.Writer
	textHandler    *slog.TextHandler
}

// NewLogConfigs crea e inizializza una struttura WriterConfigs.
// Cosa fa: imposta i campi in base ai parametri, crea l'io.MultiWriter e init del TextHandler.
// Parametri:
//   - consoleLogging: abilita o meno il logging sulla console
//   - fileLocation: percorso del file di log (se vuoto non viene creato un file)
//   - fileMaxSize: dimensione massima in MB prima della rotazione
//   - fileMaxBackups: numero di file di backup da mantenere
//   - compress: abilita o meno la compressione dei file di backup
//
// Ritorna: puntatore a WriterConfigs pronto all'uso
func NewLogConfigs(consoleLogging bool, fileLocation string, fileMaxSize int, fileMaxBackups int, compress bool) *WriterConfigs {
	wc := &WriterConfigs{
		consoleLogging: consoleLogging,
		fileLocation:   fileLocation,
		fileMaxSize:    fileMaxSize,
		fileMaxBackups: fileMaxBackups,
		compress:       compress,
	}
	wc.toMulti()
	wc.initTextHandler()
	return wc
}

// toMulti costruisce l'io.MultiWriter in base alla configurazione.
// Cosa fa: aggiunge stdout e/o un writer su file (lumberjack) al MultiWriter.
// Parametri: nessuno
// Ritorna: nulla
func (wc *WriterConfigs) toMulti() {
	// Creo un slice di writer
	var writers []io.Writer

	if wc.consoleLogging {
		writers = append(writers, os.Stdout)
	}

	if wc.fileLocation != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   wc.fileLocation,
			MaxSize:    wc.fileMaxSize,
			MaxBackups: wc.fileMaxBackups,
			Compress:   wc.compress,
		}
		writers = append(writers, fileWriter)
	}
	wc.multi = io.MultiWriter(writers...)
}

// initTextHandler inizializza un TextHandler di slog usando il MultiWriter.
// Parametri: nessuno
// Ritorna: nulla
func (wc *WriterConfigs) initTextHandler() {
	wc.textHandler = slog.NewTextHandler(wc.multi, nil)
}

// GetTextHandler restituisce il TextHandler interno.
// Parametri: nessuno
// Ritorna: *slog.TextHandler (può essere nil)
func (wc *WriterConfigs) GetTextHandler() *slog.TextHandler {
	return wc.textHandler
}

// GetMultiWriter restituisce l'io.Writer combinato.
// Parametri: nessuno
// Ritorna: io.Writer
func (wc *WriterConfigs) GetMultiWriter() io.Writer {
	return wc.multi
}

// IsConsoleLoggingEnabled indica se il logging su console è abilitato.
// Parametri: nessuno
// Ritorna: bool
func (wc *WriterConfigs) IsConsoleLoggingEnabled() bool {
	return wc.consoleLogging
}

// GetFileLocation restituisce il percorso del file di log configurato.
// Parametri: nessuno
// Ritorna: string
func (wc *WriterConfigs) GetFileLocation() string {
	return wc.fileLocation
}

// GetFileMaxSize restituisce la dimensione massima configurata per i file di log.
// Parametri: nessuno
// Ritorna: int (MB)
func (wc *WriterConfigs) GetFileMaxSize() int {
	return wc.fileMaxSize
}

// GetFileMaxBackups restituisce il numero di backup configurati.
// Parametri: nessuno
// Ritorna: int
func (wc *WriterConfigs) GetFileMaxBackups() int {
	return wc.fileMaxBackups
}

// IsCompressEnabled indica se la compressione dei backup è abilitata.
// Parametri: nessuno
// Ritorna: bool
func (wc *WriterConfigs) IsCompressEnabled() bool {
	return wc.compress
}
