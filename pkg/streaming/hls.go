package streaming

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// StreamStatus represents the current state of the transcoder
type StreamStatus string

const (
	StatusStarting        StreamStatus = "starting"
	StatusConnecting      StreamStatus = "connecting"
	StatusStreaming       StreamStatus = "streaming"
	StatusError           StreamStatus = "error"
	StatusStopped         StreamStatus = "stopped"
)

// StreamError represents different types of streaming errors
type StreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Common error types
const (
	ErrorTypeRTMPConnection = "rtmp_connection"
	ErrorTypeRTMPTimeout    = "rtmp_timeout" 
	ErrorTypeFFmpegFailed   = "ffmpeg_failed"
	ErrorTypeNetworkError   = "network_error"
	ErrorTypeUnknown        = "unknown"
)

// HLSTranscoder manages FFmpeg processes for RTMP to HLS conversion
type HLSTranscoder struct {
	babyUID      string
	rtmpURL      string
	hlsDir       string
	cmd          *exec.Cmd
	mutex        sync.RWMutex
	isRunning    bool
	stopChan     chan struct{}
	status       StreamStatus
	lastError    *StreamError
	startTime    time.Time
	retryCount   int
	maxRetries   int
	retryDelay   time.Duration
}

// NewHLSTranscoder creates a new HLS transcoder for a baby
func NewHLSTranscoder(babyUID, rtmpURL, baseHLSDir string) *HLSTranscoder {
	hlsDir := filepath.Join(baseHLSDir, babyUID)
	
	return &HLSTranscoder{
		babyUID:    babyUID,
		rtmpURL:    rtmpURL,
		hlsDir:     hlsDir,
		stopChan:   make(chan struct{}),
		isRunning:  false,
		status:     StatusStopped,
		maxRetries: 5,
		retryDelay: 10 * time.Second,
	}
}

// Start begins the HLS transcoding process
func (h *HLSTranscoder) Start() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.isRunning {
		return fmt.Errorf("transcoder already running for baby %s", h.babyUID)
	}

	// Reset state
	h.status = StatusStarting
	h.lastError = nil
	h.startTime = time.Now()
	h.retryCount = 0

	// Ensure HLS directory exists
	if err := os.MkdirAll(h.hlsDir, 0755); err != nil {
		h.setError(ErrorTypeFFmpegFailed, "Failed to create HLS directory", err.Error())
		return fmt.Errorf("failed to create HLS directory: %v", err)
	}

	// Clean up any existing files
	h.cleanupFiles()

	// Build FFmpeg command
	playlistPath := filepath.Join(h.hlsDir, "playlist.m3u8")
	segmentPath := filepath.Join(h.hlsDir, "segment_%d.ts")

	args := []string{
		"-i", h.rtmpURL,                    // Input RTMP stream
		"-c:v", "libx264",                  // Video codec
		"-preset", "ultrafast",             // Fast encoding
		"-tune", "zerolatency",             // Low latency
		"-c:a", "aac",                      // Audio codec
		"-f", "hls",                        // HLS format
		"-hls_time", "2",                   // 2 second segments
		"-hls_list_size", "5",              // Keep 5 segments in playlist
		"-hls_flags", "delete_segments",    // Auto-delete old segments
		"-hls_segment_filename", segmentPath,
		"-y",                               // Overwrite output
		playlistPath,
	}

	h.cmd = exec.Command("ffmpeg", args...)
	h.cmd.Dir = h.hlsDir

	// Set up logging
	h.cmd.Stdout = nil // Suppress stdout
	h.cmd.Stderr = nil // Suppress stderr for now - could add logging if needed

	log.Info().
		Str("baby_uid", h.babyUID).
		Str("rtmp_url", h.rtmpURL).
		Str("hls_dir", h.hlsDir).
		Int("retry_count", h.retryCount).
		Msg("Starting HLS transcoding")

	if err := h.cmd.Start(); err != nil {
		h.setError(ErrorTypeFFmpegFailed, "Failed to start FFmpeg process", err.Error())
		return fmt.Errorf("failed to start FFmpeg: %v", err)
	}

	h.isRunning = true
	h.status = StatusConnecting

	// Monitor the process
	go h.monitor()

	return nil
}

// Stop terminates the HLS transcoding process
func (h *HLSTranscoder) Stop() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.isRunning {
		return
	}

	log.Info().Str("baby_uid", h.babyUID).Msg("Stopping HLS transcoding")

	// Signal stop
	close(h.stopChan)
	h.isRunning = false

	// Terminate FFmpeg process
	if h.cmd != nil && h.cmd.Process != nil {
		h.cmd.Process.Kill()
		h.cmd.Wait() // Wait for process to exit
	}

	// Clean up files
	h.cleanupFiles()
}

// IsRunning returns whether the transcoder is currently running
func (h *HLSTranscoder) IsRunning() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.isRunning
}

// GetPlaylistPath returns the path to the HLS playlist
func (h *HLSTranscoder) GetPlaylistPath() string {
	return filepath.Join(h.hlsDir, "playlist.m3u8")
}

// GetHLSDir returns the HLS directory path
func (h *HLSTranscoder) GetHLSDir() string {
	return h.hlsDir
}

// monitor watches the FFmpeg process and handles cleanup
func (h *HLSTranscoder) monitor() {
	defer func() {
		h.mutex.Lock()
		h.isRunning = false
		if h.status != StatusError {
			h.status = StatusStopped
		}
		h.mutex.Unlock()
	}()

	// Check if HLS files are being generated (indicates successful connection)
	checkTicker := time.NewTicker(5 * time.Second)
	defer checkTicker.Stop()
	
	connected := false
	go func() {
		for range checkTicker.C {
			if h.hasHLSFiles() && !connected {
				h.mutex.Lock()
				h.status = StatusStreaming
				connected = true
				h.mutex.Unlock()
				log.Info().Str("baby_uid", h.babyUID).Msg("HLS transcoding producing files successfully")
				break
			}
		}
	}()

	// Wait for process to finish or stop signal
	done := make(chan error, 1)
	go func() {
		done <- h.cmd.Wait()
	}()

	select {
	case <-h.stopChan:
		log.Info().Str("baby_uid", h.babyUID).Msg("HLS transcoding stopped by request")
	case err := <-done:
		if err != nil {
			h.classifyAndSetError(err)
			log.Error().
				Err(err).
				Str("baby_uid", h.babyUID).
				Str("error_type", h.lastError.Type).
				Int("retry_count", h.retryCount).
				Msg("HLS transcoding process exited with error")
			
			// Attempt retry for connection issues
			if h.shouldRetry() {
				h.scheduleRetry()
				return
			}
		} else {
			log.Info().Str("baby_uid", h.babyUID).Msg("HLS transcoding process exited normally")
		}
	}
}

// cleanupFiles removes HLS files from the directory
func (h *HLSTranscoder) cleanupFiles() {
	pattern := filepath.Join(h.hlsDir, "*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Warn().Err(err).Str("baby_uid", h.babyUID).Msg("Failed to glob HLS files for cleanup")
		return
	}

	for _, file := range matches {
		if err := os.Remove(file); err != nil {
			log.Warn().Err(err).Str("file", file).Msg("Failed to remove HLS file")
		}
	}
}

// HLSManager manages multiple HLS transcoders
type HLSManager struct {
	transcoders   map[string]*HLSTranscoder
	baseHLSDir    string
	mutex         sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewHLSManager creates a new HLS manager
func NewHLSManager(baseHLSDir string) *HLSManager {
	manager := &HLSManager{
		transcoders: make(map[string]*HLSTranscoder),
		baseHLSDir:  baseHLSDir,
		stopCleanup: make(chan struct{}),
	}
	
	// Start periodic cleanup of orphaned files
	manager.startPeriodicCleanup()
	
	return manager
}

// StartTranscoding starts HLS transcoding for a baby
func (m *HLSManager) StartTranscoding(babyUID, rtmpURL string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Stop existing transcoder if running
	if existing, exists := m.transcoders[babyUID]; exists {
		existing.Stop()
	}

	// Create new transcoder
	transcoder := NewHLSTranscoder(babyUID, rtmpURL, m.baseHLSDir)
	if err := transcoder.Start(); err != nil {
		return err
	}

	m.transcoders[babyUID] = transcoder
	return nil
}

// StopTranscoding stops HLS transcoding for a baby
func (m *HLSManager) StopTranscoding(babyUID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if transcoder, exists := m.transcoders[babyUID]; exists {
		transcoder.Stop()
		delete(m.transcoders, babyUID)
	}
}

// GetTranscoder returns the transcoder for a baby
func (m *HLSManager) GetTranscoder(babyUID string) (*HLSTranscoder, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	transcoder, exists := m.transcoders[babyUID]
	return transcoder, exists
}

// StopAll stops all transcoders
func (m *HLSManager) StopAll() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Stop cleanup ticker
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
		close(m.stopCleanup)
	}

	for babyUID, transcoder := range m.transcoders {
		transcoder.Stop()
		delete(m.transcoders, babyUID)
	}
}

// startPeriodicCleanup starts a background routine to clean up orphaned HLS files
func (m *HLSManager) startPeriodicCleanup() {
	m.cleanupTicker = time.NewTicker(30 * time.Minute) // Clean up every 30 minutes
	
	go func() {
		for {
			select {
			case <-m.cleanupTicker.C:
				m.cleanupOrphanedFiles()
			case <-m.stopCleanup:
				return
			}
		}
	}()
}

// cleanupOrphanedFiles removes HLS files for babies that are no longer being transcoded
func (m *HLSManager) cleanupOrphanedFiles() {
	log.Debug().Msg("Starting periodic HLS cleanup")
	
	// Get list of all baby directories
	pattern := filepath.Join(m.baseHLSDir, "*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to glob baby directories for cleanup")
		return
	}
	
	m.mutex.RLock()
	activeTranscoders := make(map[string]bool)
	for babyUID := range m.transcoders {
		activeTranscoders[babyUID] = true
	}
	m.mutex.RUnlock()
	
	cleanedCount := 0
	for _, dir := range matches {
		babyUID := filepath.Base(dir)
		
		// Skip if transcoder is still active
		if activeTranscoders[babyUID] {
			continue
		}
		
		// Check if directory has old files (older than 1 hour)
		if m.hasOldFiles(dir, time.Hour) {
			if err := os.RemoveAll(dir); err != nil {
				log.Warn().Err(err).Str("dir", dir).Msg("Failed to remove orphaned HLS directory")
			} else {
				cleanedCount++
				log.Debug().Str("baby_uid", babyUID).Msg("Cleaned up orphaned HLS directory")
			}
		}
	}
	
	if cleanedCount > 0 {
		log.Info().Int("cleaned_count", cleanedCount).Msg("Completed HLS cleanup")
	}
}

// hasOldFiles checks if a directory contains files older than the specified duration
func (m *HLSManager) hasOldFiles(dir string, maxAge time.Duration) bool {
	pattern := filepath.Join(dir, "*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}
	
	cutoff := time.Now().Add(-maxAge)
	
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		if info.ModTime().Before(cutoff) {
			return true
		}
	}
	
	return false
}

// setError sets the error state with detailed information
func (h *HLSTranscoder) setError(errorType, message, code string) {
	h.status = StatusError
	h.lastError = &StreamError{
		Type:    errorType,
		Message: message,
		Code:    code,
	}
}

// classifyAndSetError analyzes the FFmpeg error and sets appropriate error type
func (h *HLSTranscoder) classifyAndSetError(err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	errStr := err.Error()
	
	// Check for common RTMP connection issues
	if strings.Contains(errStr, "Connection refused") || 
	   strings.Contains(errStr, "Connection reset") ||
	   strings.Contains(errStr, "No route to host") {
		h.setError(ErrorTypeRTMPConnection, "Cannot connect to RTMP server", errStr)
	} else if strings.Contains(errStr, "Connection timed out") ||
			  strings.Contains(errStr, "timeout") {
		h.setError(ErrorTypeRTMPTimeout, "RTMP connection timed out", errStr)
	} else if strings.Contains(errStr, "Server error") ||
			  strings.Contains(errStr, "403") ||
			  strings.Contains(errStr, "404") {
		h.setError(ErrorTypeRTMPConnection, "RTMP server rejected connection", errStr)
	} else if strings.Contains(errStr, "exit status") {
		// Check if we've been running long enough to classify as timeout
		if time.Since(h.startTime) < 10*time.Second {
			h.setError(ErrorTypeRTMPConnection, "RTMP stream not available", errStr)
		} else {
			h.setError(ErrorTypeNetworkError, "Stream disconnected unexpectedly", errStr)
		}
	} else {
		h.setError(ErrorTypeUnknown, "FFmpeg process failed", errStr)
	}
}

// hasHLSFiles checks if HLS files are being generated
func (h *HLSTranscoder) hasHLSFiles() bool {
	playlistPath := filepath.Join(h.hlsDir, "playlist.m3u8")
	if _, err := os.Stat(playlistPath); err == nil {
		// Check if playlist was recently modified (within last 10 seconds)
		if info, err := os.Stat(playlistPath); err == nil {
			return time.Since(info.ModTime()) < 10*time.Second
		}
	}
	return false
}

// GetStatus returns the current status and error information
func (h *HLSTranscoder) GetStatus() (StreamStatus, *StreamError) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.status, h.lastError
}

// GetDetailedInfo returns comprehensive transcoder information
func (h *HLSTranscoder) GetDetailedInfo() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	info := map[string]interface{}{
		"baby_uid":     h.babyUID,
		"status":       string(h.status),
		"is_running":   h.isRunning,
		"start_time":   h.startTime,
		"retry_count":  h.retryCount,
		"max_retries":  h.maxRetries,
	}
	
	if h.lastError != nil {
		info["error"] = h.lastError
	}
	
	if h.isRunning {
		info["uptime"] = time.Since(h.startTime).Seconds()
		info["has_files"] = h.hasHLSFiles()
	}
	
	return info
}

// shouldRetry determines if we should retry based on error type and retry count
func (h *HLSTranscoder) shouldRetry() bool {
	if h.retryCount >= h.maxRetries {
		return false
	}
	
	// Only retry for connection-related errors
	if h.lastError != nil {
		switch h.lastError.Type {
		case ErrorTypeRTMPConnection, ErrorTypeRTMPTimeout, ErrorTypeNetworkError:
			return true
		}
	}
	
	return false
}

// scheduleRetry schedules a retry attempt after a delay
func (h *HLSTranscoder) scheduleRetry() {
	h.retryCount++
	
	log.Info().
		Str("baby_uid", h.babyUID).
		Int("retry_count", h.retryCount).
		Int("max_retries", h.maxRetries).
		Dur("retry_delay", h.retryDelay).
		Msg("Scheduling HLS transcoding retry")
	
	go func() {
		select {
		case <-time.After(h.retryDelay):
			h.mutex.Lock()
			if !h.isRunning {
				h.mutex.Unlock()
				return
			}
			h.mutex.Unlock()
			
			log.Info().
				Str("baby_uid", h.babyUID).
				Int("retry_count", h.retryCount).
				Msg("Retrying HLS transcoding")
			
			// Restart FFmpeg process
			h.mutex.Lock()
			h.status = StatusConnecting
			h.lastError = nil
			h.mutex.Unlock()
			
			// Build and start FFmpeg command again
			if err := h.restartFFmpeg(); err != nil {
				log.Error().
					Err(err).
					Str("baby_uid", h.babyUID).
					Msg("Failed to restart FFmpeg during retry")
			}
		case <-h.stopChan:
			return
		}
	}()
}

// restartFFmpeg restarts the FFmpeg process for retries
func (h *HLSTranscoder) restartFFmpeg() error {
	// Clean up any existing files
	h.cleanupFiles()

	// Build FFmpeg command
	playlistPath := filepath.Join(h.hlsDir, "playlist.m3u8")
	segmentPath := filepath.Join(h.hlsDir, "segment_%d.ts")

	args := []string{
		"-i", h.rtmpURL,                    // Input RTMP stream
		"-c:v", "libx264",                  // Video codec
		"-preset", "ultrafast",             // Fast encoding
		"-tune", "zerolatency",             // Low latency
		"-c:a", "aac",                      // Audio codec
		"-f", "hls",                        // HLS format
		"-hls_time", "2",                   // 2 second segments
		"-hls_list_size", "5",              // Keep 5 segments in playlist
		"-hls_flags", "delete_segments",    // Auto-delete old segments
		"-hls_segment_filename", segmentPath,
		"-y",                               // Overwrite output
		playlistPath,
	}

	h.cmd = exec.Command("ffmpeg", args...)
	h.cmd.Dir = h.hlsDir

	// Set up logging
	h.cmd.Stdout = nil // Suppress stdout
	h.cmd.Stderr = nil // Suppress stderr for now - could add logging if needed

	if err := h.cmd.Start(); err != nil {
		h.mutex.Lock()
		h.setError(ErrorTypeFFmpegFailed, "Failed to restart FFmpeg process", err.Error())
		h.mutex.Unlock()
		return fmt.Errorf("failed to restart FFmpeg: %v", err)
	}

	// Monitor the process
	go h.monitor()

	return nil
}