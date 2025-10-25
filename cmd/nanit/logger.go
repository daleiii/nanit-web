package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/indiefan/home_assistant_nanit/pkg/utils"
)

// Set log level after env. initialization
func setLogLevel() {
	// Try to read log level from env. variable
	logLevelStr := utils.EnvVarStr("NANIT_LOG_LEVEL", "info")
	logLevel, err := zerolog.ParseLevel(logLevelStr)
	if err != nil || logLevel == zerolog.NoLevel {
		log.Error().
			Str("value", logLevelStr).
			Msg("Unknown log level specified. Valid levels: trace, debug, info, warn, error, fatal, panic")
		os.Exit(1)
	}

	log.Info().Msgf("Setting log level to %v", logLevel)
	zerolog.SetGlobalLevel(logLevel)
}

// Set logger for application bootstrap
func initLogger() {
	// Initial log level, overridden later by setLogLevel
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC822}
	log.Logger = log.Output(consoleWriter)
}
