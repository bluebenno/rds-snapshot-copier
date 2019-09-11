package main

import (
	"log"
	"time"

	"github.com/bluebenno/rds-snapshot-copier/cmd/rds-snapshot-copier/flags"
	"github.com/bluebenno/rds-snapshot-copier/internal/wiring"
	"github.com/bluebenno/rds-snapshot-copier/internal/worker"
	"go.uber.org/zap"
)

const (
	appName   = "rds-snapshot-copier"
	gitCommit = "dirty"
	version   = "devbuild"
	// AntiRateLimit will be slept in key places, to preclude AWS API rate-limiting.
	AntiRateLimit = (100 * time.Millisecond)
)

func main() {
	var cfg wiring.Config

	app := Flags.Flags(appName, gitCommit, version, &cfg)
	if app == nil {
		log.Fatalf("Failed to parse flags")
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Unable to create logger: %s", err.Error())
	}
	logger.Info("Starting")

	err2 := worker.Run(logger, &cfg)
	if err2 != nil {
		log.Fatalf("Failed to parse flags: %+v", err2)
	}
}
