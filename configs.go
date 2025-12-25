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

func (wc *WriterConfigs) initTextHandler() {
	wc.textHandler = slog.NewTextHandler(wc.multi, nil)
}

func (wc *WriterConfigs) GetTextHandler() *slog.TextHandler {
	return wc.textHandler
}

func (wc *WriterConfigs) GetMultiWriter() io.Writer {
	return wc.multi
}

func (wc *WriterConfigs) IsConsoleLoggingEnabled() bool {
	return wc.consoleLogging
}

func (wc *WriterConfigs) GetFileLocation() string {
	return wc.fileLocation
}

func (wc *WriterConfigs) GetFileMaxSize() int {
	return wc.fileMaxSize
}

func (wc *WriterConfigs) GetFileMaxBackups() int {
	return wc.fileMaxBackups
}

func (wc *WriterConfigs) IsCompressEnabled() bool {
	return wc.compress
}
