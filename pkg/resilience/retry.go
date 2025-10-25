package resilience

import (
	"math"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	Jitter          bool
	RetryableErrors []string
}

// DefaultRetryConfig returns sensible defaults for retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// RetryWithExponentialBackoff retries a function with exponential backoff
func RetryWithExponentialBackoff(name string, config RetryConfig, fn func() error) error {
	var lastErr error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := calculateDelay(attempt-1, config)
			log.Debug().
				Str("operation", name).
				Int("attempt", attempt).
				Dur("delay", delay).
				Msg("Retrying operation after delay")
			time.Sleep(delay)
		}
		
		err := fn()
		if err == nil {
			if attempt > 0 {
				log.Info().
					Str("operation", name).
					Int("attempts", attempt+1).
					Msg("Operation succeeded after retry")
			}
			return nil
		}
		
		lastErr = err
		log.Debug().
			Str("operation", name).
			Int("attempt", attempt+1).
			Err(err).
			Msg("Operation failed")
		
		// Check if the error is retryable
		if !isRetryableError(err, config.RetryableErrors) {
			log.Debug().
				Str("operation", name).
				Err(err).
				Msg("Error is not retryable, stopping retry attempts")
			break
		}
	}
	
	log.Error().
		Str("operation", name).
		Int("max_attempts", config.MaxRetries+1).
		Err(lastErr).
		Msg("Operation failed after all retry attempts")
	
	return lastErr
}

// calculateDelay calculates the delay for the given attempt with exponential backoff
func calculateDelay(attempt int, config RetryConfig) time.Duration {
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt))
	
	// Cap the delay at MaxDelay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	
	// Add jitter to prevent thundering herd
	if config.Jitter {
		jitter := rand.Float64() * 0.1 * delay // 10% jitter
		delay += jitter
	}
	
	return time.Duration(delay)
}

// isRetryableError checks if an error is retryable based on configuration
func isRetryableError(err error, retryableErrors []string) bool {
	if len(retryableErrors) == 0 {
		// If no specific retryable errors are configured, retry all errors
		return true
	}
	
	errStr := err.Error()
	for _, retryableErr := range retryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}
	
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || s[len(s)-len(substr):] == substr || s[:len(substr)] == substr)
}