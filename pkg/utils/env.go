package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// TODO: We should probably use some library if there is need for additional functionality

// EnvVarStr - retrieves value of string environment variable, while applying default
func EnvVarStr(varName string, defaultValue string) string {
	value := os.Getenv(varName)

	if value == "" {
		return defaultValue
	}

	return value
}

// EnvVarReqStr - retrieves value of string environment variable, fails if it is not present or empty
func EnvVarReqStr(varName string) string {
	value := EnvVarStr(varName, "")

	if value == "" {
		log.Fatal().Msgf("Missing required environment variable %v. Please set this variable and restart the application.", varName)
	}

	return value
}

// EnvVarBool - retrieves value of boolean environment variable, fails if variable contains non-boolean value
func EnvVarBool(varName string, defaultValue bool) bool {
	value := EnvVarStr(varName, "")
	if value == "true" {
		return true
	} else if value == "false" {
		return false
	} else if value == "" {
		return defaultValue
	}

	log.Fatal().Msgf("Invalid value '%v' for boolean environment variable %v. Allowed values: 'true', 'false'", value, varName)
	return false
}

// EnvVarInt - retrieves value of integer environment variable, while applying default
func EnvVarInt(varName string, defaultValue int) int {
	valueStr := os.Getenv(varName)

	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Fatal().Msgf("Invalid value '%v' for integer environment variable %v. Please provide a valid integer.", valueStr, varName)
	}

	return value
}

// EnvVarSeconds - retrieves value of environment variable reperesenting duration in seconds, fails if variable non-parseable values
func EnvVarSeconds(varName string, defaultValue time.Duration) time.Duration {
	valueStr, found := os.LookupEnv(varName)

	if !found {
		return defaultValue
	}

	valueInt, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		log.Fatal().Msgf("Invalid value '%v' for duration environment variable %v. Please provide a valid number of seconds.", valueStr, varName)
	}

	value := time.Duration(valueInt) * time.Second

	return value
}

// LoadDotEnvFile - Loads environment variables from .env file in the current working directory (if found)
func LoadDotEnvFile() {
	absFilepath, filePathErr := filepath.Abs(".env")
	if filePathErr != nil {
		log.Fatal().Str("path", absFilepath).Err(filePathErr).Msg("Unable to retrieve absolute file path for .env file")
	}

	// loads values from .env into the system
	if err := godotenv.Load(absFilepath); err != nil {
		log.Info().Str("path", absFilepath).Msg("No .env file found. Using only environment variables")
	} else {
		log.Info().Str("path", absFilepath).Msg("Additional environment variables loaded from .env file")
	}
}
