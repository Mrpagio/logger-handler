package main

import (
	"fmt"
	"log/slog"
	"time"

	loggerhandler "github.com/Mrpagio/logger-handler"
)

func main() {
	// Configuro writer: console abilitato, nessun file
	logCfg := loggerhandler.NewLogConfigs(true, "", 10, 3, false)
	errCfg := loggerhandler.NewLogConfigs(true, "", 10, 3, false)

	// Passo nil come meter: le metriche sono opzionali
	lh := loggerhandler.NewLoggerHandler(logCfg, errCfg, nil, 100)

	fmt.Printf("Meter is nil? %v\n", lh.GetMeter() == nil)
	fmt.Printf("TotalCounter is nil? %v\n", lh.GetTotalCounter() == nil)

	// Creo uno span senza timeout
	span := lh.AddSpan(0, []string{"example-nil-meter"}, 10, slog.LevelDebug)

	span.Info("info with nil meter")

	// Rilascio con successo
	span.ReleaseSuccess()

	// Attendere brevemente per permettere l'elaborazione dei comandi
	time.Sleep(50 * time.Millisecond)

	// Chiudo il logger handler
	lh.Close()
}
