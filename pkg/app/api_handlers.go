package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/indiefan/home_assistant_nanit/pkg/baby"
	"github.com/indiefan/home_assistant_nanit/pkg/session"
	"github.com/indiefan/home_assistant_nanit/pkg/streaming"
)

// API handler for current status
func handleStatusAPI(w http.ResponseWriter, r *http.Request, babies []baby.Baby, stateManager *baby.StateManager) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"babies":    make([]interface{}, 0),
	}

	for _, b := range babies {
		babyState := stateManager.GetBabyState(b.UID)
		babyStatus := map[string]interface{}{
			"uid":              b.UID,
			"name":             b.Name,
			"camera_uid":       b.CameraUID,
			"temperature":      babyState.GetTemperature(),
			"humidity":         babyState.GetHumidity(),
			"is_night":         babyState.IsNight,
			"night_light":      babyState.GetNightLight(),
			"standby":          babyState.GetStandby(),
			"websocket_alive":  babyState.GetIsWebsocketAlive(),
			"stream_state":     babyState.GetStreamState(),
		}
		status["babies"] = append(status["babies"].([]interface{}), babyStatus)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// API handler for babies list
func handleBabiesAPI(w http.ResponseWriter, r *http.Request, babies []baby.Baby, stateManager *baby.StateManager) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result := map[string]interface{}{
		"babies": babies,
		"count":  len(babies),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// API handler for control commands
func handleControlAPI(w http.ResponseWriter, r *http.Request, controlType string, babies []baby.Baby, stateManager *baby.StateManager, app *App) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		BabyUID string `json:"baby_uid"`
		Action  string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if requestData.BabyUID == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}

	// Verify baby exists
	var targetBaby *baby.Baby
	for _, b := range babies {
		if b.UID == requestData.BabyUID {
			targetBaby = &b
			break
		}
	}

	if targetBaby == nil {
		http.Error(w, "Baby not found", http.StatusNotFound)
		return
	}

	// Get WebSocket connection
	conn := app.getConnection(requestData.BabyUID)
	if conn == nil {
		http.Error(w, "WebSocket not connected", http.StatusServiceUnavailable)
		return
	}

	// Get current state
	currentState := stateManager.GetBabyState(requestData.BabyUID)

	// Execute control command
	switch controlType {
	case "night-light":
		if requestData.Action == "toggle" {
			newState := !currentState.GetNightLight()
			sendLightCommand(newState, conn)
			
			log.Info().
				Str("baby_uid", requestData.BabyUID).
				Bool("new_state", newState).
				Msg("Night light toggle command sent")
		} else {
			http.Error(w, "Invalid action for night-light", http.StatusBadRequest)
			return
		}

	case "standby":
		if requestData.Action == "toggle" {
			newState := !currentState.GetStandby()
			sendStandbyCommand(newState, conn)
			
			log.Info().
				Str("baby_uid", requestData.BabyUID).
				Bool("new_state", newState).
				Msg("Standby toggle command sent")
		} else {
			http.Error(w, "Invalid action for standby", http.StatusBadRequest)
			return
		}

	default:
		http.Error(w, "Unknown control type", http.StatusBadRequest)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success":   true,
		"baby_uid":  requestData.BabyUID,
		"control":   controlType,
		"action":    requestData.Action,
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeviceAlert represents an error or warning for the device
type DeviceAlert struct {
	Type     string `json:"type"`     // "error" or "warning"
	Message  string `json:"message"`
	Category string `json:"category"`
}

// DeviceInfoResponse represents the full device information response
type DeviceInfoResponse struct {
	BabyUID          string                 `json:"baby_uid"`
	BabyName         string                 `json:"baby_name"`
	CameraUID        string                 `json:"camera_uid"`
	Timestamp        int64                  `json:"timestamp"`
	DeviceInfo       *baby.DeviceInfo       `json:"device_info"`
	ConnectionStatus map[string]interface{} `json:"connection_status"`
	Alerts           []DeviceAlert          `json:"alerts"`
}

// Device info endpoint handler
func handleDeviceInfoAPI(w http.ResponseWriter, r *http.Request, babies []baby.Baby, stateManager *baby.StateManager) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract baby UID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/device-info/")
	if path == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}

	babyUID := path

	// Find the baby
	var targetBaby *baby.Baby
	for _, b := range babies {
		if b.UID == babyUID {
			targetBaby = &b
			break
		}
	}

	if targetBaby == nil {
		http.Error(w, "Baby not found", http.StatusNotFound)
		return
	}

	// Get current state with device info
	babyState := stateManager.GetBabyState(babyUID)
	deviceInfo := babyState.GetDeviceInfo()

	// Build connection status
	connectionStatus := map[string]interface{}{
		"websocket_alive": babyState.GetIsWebsocketAlive(),
		"stream_state":    getStreamStateString(babyState.StreamState),
	}

	// Build alerts based on current state
	var alerts []DeviceAlert

	// Check for websocket connection issues
	if !babyState.GetIsWebsocketAlive() {
		alerts = append(alerts, DeviceAlert{
			Type:     "error",
			Message:  "Camera is disconnected from Nanit servers",
			Category: "connectivity",
		})
	}

	// Check for streaming errors
	if deviceInfo.StreamingError != nil && *deviceInfo.StreamingError != "" {
		alerts = append(alerts, DeviceAlert{
			Type:     "error",
			Message:  *deviceInfo.StreamingError,
			Category: "streaming",
		})
	}

	// Check stream state for issues
	if babyState.StreamState != nil {
		switch *babyState.StreamState {
		case baby.StreamState_Unhealthy:
			alerts = append(alerts, DeviceAlert{
				Type:     "warning",
				Message:  "Video streaming is experiencing issues",
				Category: "streaming",
			})
		case baby.StreamState_Unknown:
			alerts = append(alerts, DeviceAlert{
				Type:     "warning",
				Message:  "Video stream status unknown",
				Category: "streaming",
			})
		}
	}

	// Check for connection limit issues (streaming blocked by too many mobile apps)
	if babyState.GetStreamRequestState() == baby.StreamRequestState_RequestFailed {
		alerts = append(alerts, DeviceAlert{
			Type:     "warning",
			Message:  "Streaming blocked: Too many Nanit mobile apps connected. Close the official Nanit app on your phone/tablet to enable streaming here.",
			Category: "connection_limit",
		})
	}

	// Check for device warnings
	if deviceInfo.SleepMode != nil && *deviceInfo.SleepMode {
		alerts = append(alerts, DeviceAlert{
			Type:     "warning",
			Message:  "Camera is in sleep mode",
			Category: "device_state",
		})
	}

	if deviceInfo.UpgradeDownloaded != nil && *deviceInfo.UpgradeDownloaded {
		alerts = append(alerts, DeviceAlert{
			Type:     "warning",
			Message:  "Firmware update available for installation",
			Category: "firmware",
		})
	}

	// Build full response
	response := DeviceInfoResponse{
		BabyUID:          targetBaby.UID,
		BabyName:         targetBaby.Name,
		CameraUID:        targetBaby.CameraUID,
		Timestamp:        time.Now().Unix(),
		DeviceInfo:       deviceInfo,
		ConnectionStatus: connectionStatus,
		Alerts:           alerts,
	}

	// Return full device info response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to convert stream state to string
func getStreamStateString(streamState *baby.StreamState) string {
	if streamState == nil {
		return "unknown"
	}
	
	switch *streamState {
	case baby.StreamState_Unknown:
		return "unknown"
	case baby.StreamState_Unhealthy:
		return "unhealthy"
	case baby.StreamState_Alive:
		return "connected"
	default:
		return "unknown"
	}
}

// Authentication API handlers
func handleAuthLoginAPI(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("=== Starting login attempt ===")
	
	if r.Method != "POST" {
		log.Warn().Str("method", r.Method).Msg("Invalid HTTP method for login")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		log.Error().Err(err).Msg("Failed to decode login request JSON")
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if requestData.Email == "" || requestData.Password == "" {
		log.Warn().Msg("Missing email or password in login request")
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	log.Info().Str("email", requestData.Email).Msg("Processing login request")

	// Call Nanit login API to get MFA token (matching original rest.go)
	loginData := map[string]string{
		"email":    requestData.Email,
		"password": requestData.Password,
	}

	loginJSON, _ := json.Marshal(loginData)
	log.Info().Str("payload", string(loginJSON)).Msg("Sending login request to Nanit API")
	
	req, err := http.NewRequest("POST", "https://api.nanit.com/login", strings.NewReader(string(loginJSON)))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create login request")
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	
	// Add required headers (matching original rest.go)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("nanit-api-version", "1")
	log.Info().Msg("Added headers: Content-Type=application/json, nanit-api-version=1")
	
	client := &http.Client{Timeout: 30 * time.Second}
	log.Info().Msg("Making HTTP request to Nanit API...")
	response, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to Nanit API")
		http.Error(w, "Failed to connect to Nanit", http.StatusServiceUnavailable)
		return
	}
	defer response.Body.Close()

	log.Info().Int("status_code", response.StatusCode).Msg("Received response from Nanit API")

	var nanitResponse map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&nanitResponse); err != nil {
		log.Error().Err(err).Msg("Failed to decode Nanit API response")
		http.Error(w, "Invalid response from Nanit", http.StatusInternalServerError)
		return
	}

	log.Info().Interface("response", nanitResponse).Msg("Nanit API response body")

	// Status 201 = success without 2FA, Status 482 = 2FA required
	if response.StatusCode != 201 && response.StatusCode != 482 {
		errorMsg := "Login failed"
		if errDetail, ok := nanitResponse["error"].(string); ok {
			errorMsg = errDetail
		}
		log.Error().Int("status_code", response.StatusCode).Str("error", errorMsg).Msg("Login failed with error from Nanit API")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": errorMsg})
		return
	}

	log.Info().Msg("Login successful, received MFA token")

	// Return MFA token to client
	result := map[string]interface{}{
		"success":   true,
		"mfa_token": nanitResponse["mfa_token"],
		"message":   "MFA token received. Please check your email for verification code.",
	}

	log.Info().Msg("=== Login completed successfully, returning MFA token ===")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleAuthVerify2FAAPI(w http.ResponseWriter, r *http.Request, app *App) {
	log.Info().Msg("=== Starting 2FA verification ===")
	
	if r.Method != "POST" {
		log.Warn().Str("method", r.Method).Msg("Invalid HTTP method for 2FA verification")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		Email    string      `json:"email"`
		Password string      `json:"password"`
		MFAToken interface{} `json:"mfa_token"`
		MFACode  string      `json:"mfa_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if requestData.Email == "" || requestData.Password == "" || requestData.MFACode == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Call Nanit verification API
	// MFA code must be sent as string to preserve leading zeros (like "0123")
	verifyData := map[string]interface{}{
		"email":     requestData.Email,
		"password":  requestData.Password,
		"mfa_token": requestData.MFAToken,
		"mfa_code":  requestData.MFACode, // Already a string from JSON
		"channel":   "email",
	}
	
	log.Info().Str("mfa_code", requestData.MFACode).Msg("Sending 2FA verification request")

	verifyJSON, _ := json.Marshal(verifyData)
	log.Info().Str("payload", string(verifyJSON)).Msg("Sending verification request to Nanit API")
	
	req, err := http.NewRequest("POST", "https://api.nanit.com/login", strings.NewReader(string(verifyJSON)))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create verification request")
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	
	// Add required headers (matching original)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("nanit-api-version", "1")
	log.Info().Msg("Added headers for verification: Content-Type=application/json, nanit-api-version=1")
	
	client := &http.Client{Timeout: 30 * time.Second}
	log.Info().Msg("Making HTTP verification request to Nanit API...")
	response, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to Nanit API for verification")
		http.Error(w, "Failed to connect to Nanit", http.StatusServiceUnavailable)
		return
	}
	defer response.Body.Close()

	log.Info().Int("status_code", response.StatusCode).Msg("Received verification response from Nanit API")

	var nanitResponse map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&nanitResponse); err != nil {
		log.Error().Err(err).Msg("Failed to decode Nanit verification response")
		http.Error(w, "Invalid response from Nanit", http.StatusInternalServerError)
		return
	}

	log.Info().Interface("response", nanitResponse).Msg("Nanit verification API response body")

	if response.StatusCode != 201 {
		errorMsg := "Verification failed"
		if errDetail, ok := nanitResponse["error"].(string); ok {
			errorMsg = errDetail
		}
		log.Error().Int("status_code", response.StatusCode).Str("error", errorMsg).Msg("2FA verification failed with error from Nanit API")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": errorMsg})
		return
	}

	log.Info().Msg("2FA verification successful!")

	// Extract refresh token
	refreshToken, ok := nanitResponse["refresh_token"].(string)
	if !ok {
		http.Error(w, "No refresh token received", http.StatusInternalServerError)
		return
	}

	// Save session data (similar to init-nanit.sh)
	sessionData := map[string]interface{}{
		"revision":     3, // Keep in sync with session.go
		"authToken":    requestData.MFAToken,
		"refreshToken": refreshToken,
	}

	sessionJSON, _ := json.Marshal(sessionData)
	sessionFile := app.Opts.SessionFile
	
	if err := os.WriteFile(sessionFile, sessionJSON, 0600); err != nil {
		log.Error().Err(err).Str("file", sessionFile).Msg("Failed to save session file")
		http.Error(w, "Failed to save authentication", http.StatusInternalServerError)
		return
	}

	log.Info().Str("file", sessionFile).Msg("Authentication saved successfully")

	// Refresh app authentication and start services
	if err := app.RefreshAuthentication(); err != nil {
		log.Error().Err(err).Msg("Failed to refresh authentication")
	} else {
		// Start monitoring services now that we have authentication
		go app.StartMonitoringServices()
	}

	// Return success
	result := map[string]interface{}{
		"success": true,
		"message": "Authentication completed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleAuthStatusAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if session file exists and is valid
	sessionFile := app.Opts.SessionFile
	isAuthenticated := false
	var message string
	var email string
	var authTime *time.Time
	var babiesCount int
	servicesRunning := false

	if _, err := os.Stat(sessionFile); err == nil {
		// Session file exists - check if it's valid
		if app.SessionStore != nil && app.SessionStore.Session != nil && app.SessionStore.Session.RefreshToken != "" {
			isAuthenticated = true
			message = "Authenticated"
			
			// Extract email from credentials if available
			if app.RestClient != nil && app.RestClient.Email != "" {
				email = app.RestClient.Email
			}
			
			// Get auth time
			if !app.SessionStore.Session.AuthTime.IsZero() {
				authTime = &app.SessionStore.Session.AuthTime
			}
			
			// Count babies
			babiesCount = len(app.SessionStore.Session.Babies)
			
			// Check if services are running (at least one baby has active WebSocket)
			if babiesCount > 0 {
				for _, baby := range app.SessionStore.Session.Babies {
					state := app.BabyStateManager.GetBabyState(baby.UID)
					if state.GetIsWebsocketAlive() {
						servicesRunning = true
						break
					}
				}
			}
		} else {
			message = "Session file exists but invalid"
		}
	} else {
		message = "No authentication found"
	}

	result := map[string]interface{}{
		"authenticated":     isAuthenticated,
		"message":           message,
		"email":             email,
		"babies_count":      babiesCount,
		"services_running":  servicesRunning,
	}
	
	if authTime != nil {
		result["auth_time"] = authTime.Unix()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleAuthResetAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session file path
	sessionFile := app.Opts.SessionFile
	
	// Stop all monitoring services first
	log.Info().Msg("Stopping monitoring services for authentication reset")
	
	// Stop HLS transcoding for all babies
	if app.HLSManager != nil {
		app.HLSManager.StopAll()
		log.Info().Msg("Stopped all HLS transcoding")
	}
	
	// Clear session store in memory
	if app.SessionStore != nil {
		app.SessionStore.Session = &session.Session{Revision: session.Revision}
		log.Info().Msg("Cleared session store from memory")
	}
	
	// Remove session file
	if sessionFile != "" {
		if err := os.Remove(sessionFile); err != nil {
			if !os.IsNotExist(err) {
				log.Error().Err(err).Str("file", sessionFile).Msg("Failed to remove session file")
				http.Error(w, "Failed to reset authentication", http.StatusInternalServerError)
				return
			}
		} else {
			log.Info().Str("file", sessionFile).Msg("Removed session file")
		}
	}
	
	// Clear REST client credentials
	if app.RestClient != nil {
		app.RestClient.RefreshToken = ""
		app.RestClient.SessionStore = session.NewSessionStore()
		log.Info().Msg("Cleared REST client credentials")
	}
	
	log.Info().Msg("Authentication reset completed successfully")
	
	result := map[string]interface{}{
		"success": true,
		"message": "Authentication reset successfully. Please re-authenticate.",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Streaming and historical API handlers - simplified implementations
func handleHLSStreamAPI(w http.ResponseWriter, r *http.Request, app *App) {
	// Extract baby UID from URL path: /api/stream/hls/{baby_uid}/playlist.m3u8
	path := strings.TrimPrefix(r.URL.Path, "/api/stream/hls/")
	parts := strings.Split(path, "/")
	
	if len(parts) < 2 {
		http.Error(w, "Invalid stream path", http.StatusBadRequest)
		return
	}
	
	babyUID := parts[0]
	fileName := parts[1]
	
	// Get transcoder for this baby
	transcoder, exists := app.HLSManager.GetTranscoder(babyUID)
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "no_transcoder",
			"message": "No stream transcoder found for this baby",
		})
		return
	}
	
	if !transcoder.IsRunning() {
		// Get error details if available
		status, streamError := transcoder.GetStatus()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		
		response := map[string]interface{}{
			"error": "transcoder_not_running",
			"status": string(status),
			"message": "Stream transcoder is not running",
		}
		
		if streamError != nil {
			response["stream_error"] = streamError
		}
		
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Serve the HLS file
	filePath := filepath.Join(transcoder.GetHLSDir(), fileName)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Check transcoder status to provide better error info
		status, streamError := transcoder.GetStatus()
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		
		response := map[string]interface{}{
			"error": "file_not_found",
			"status": string(status),
			"message": "HLS file not available yet",
			"file": fileName,
		}
		
		if streamError != nil {
			response["stream_error"] = streamError
		}
		
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Set appropriate headers
	if strings.HasSuffix(fileName, ".m3u8") {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-cache")
	} else if strings.HasSuffix(fileName, ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
		w.Header().Set("Cache-Control", "max-age=3600")
	}
	
	// Enable CORS for HLS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	
	// Serve the file
	http.ServeFile(w, r, filePath)
}

func handleStreamStartAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var requestData struct {
		BabyUID string `json:"baby_uid"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if requestData.BabyUID == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}
	
	// Build RTMP URL for this baby
	rtmpURL := app.getLocalStreamURL(requestData.BabyUID)
	if rtmpURL == "" {
		http.Error(w, "RTMP not configured", http.StatusServiceUnavailable)
		return
	}
	
	// Start HLS transcoding
	if err := app.HLSManager.StartTranscoding(requestData.BabyUID, rtmpURL); err != nil {
		log.Error().Err(err).Str("baby_uid", requestData.BabyUID).Msg("Failed to start HLS transcoding")
		http.Error(w, "Failed to start stream", http.StatusInternalServerError)
		return
	}
	
	log.Info().Str("baby_uid", requestData.BabyUID).Msg("HLS transcoding started")
	
	result := map[string]interface{}{
		"success":      true,
		"baby_uid":     requestData.BabyUID,
		"hls_url":      fmt.Sprintf("/api/stream/hls/%s/playlist.m3u8", requestData.BabyUID),
		"message":      "Stream started successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleStreamStopAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var requestData struct {
		BabyUID string `json:"baby_uid"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if requestData.BabyUID == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}
	
	// Stop HLS transcoding
	app.HLSManager.StopTranscoding(requestData.BabyUID)
	
	log.Info().Str("baby_uid", requestData.BabyUID).Msg("HLS transcoding stopped")
	
	result := map[string]interface{}{
		"success":  true,
		"baby_uid": requestData.BabyUID,
		"message":  "Stream stopped successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleStreamStatusAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract baby UID from URL path: /api/stream/status/{baby_uid}
	path := strings.TrimPrefix(r.URL.Path, "/api/stream/status/")
	if path == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}
	
	babyUID := path
	
	// Check for connection limit issues first
	babyState := app.BabyStateManager.GetBabyState(babyUID)
	if babyState.GetStreamRequestState() == baby.StreamRequestState_RequestFailed {
		result := map[string]interface{}{
			"baby_uid": babyUID,
			"status":   "blocked",
			"message":  "Streaming blocked by connection limit",
			"stream_error": map[string]interface{}{
				"type":    "connection_limit",
				"message": "Too many Nanit mobile apps connected. Close the official Nanit app to enable streaming.",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}
	
	// Get transcoder for this baby
	transcoder, exists := app.HLSManager.GetTranscoder(babyUID)
	if !exists {
		result := map[string]interface{}{
			"baby_uid": babyUID,
			"status":   "not_found",
			"message":  "No transcoder found for this baby",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}
	
	// Get detailed status information
	info := transcoder.GetDetailedInfo()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// Historical data API handlers - simplified implementations that check if feature is enabled
func handleHistorySensorAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !app.HistoryTracker.IsEnabled() {
		http.Error(w, "Historical tracking disabled", http.StatusServiceUnavailable)
		return
	}
	
	// Extract baby UID from URL path: /api/history/sensor/{baby_uid}
	path := strings.TrimPrefix(r.URL.Path, "/api/history/sensor/")
	if path == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}
	
	babyUID := path
	
	// Parse query parameters
	query := r.URL.Query()
	
	// Default to last 24 hours if not specified
	endTime := time.Now().Unix()
	startTime := endTime - (24 * 60 * 60) // 24 hours ago
	
	if startStr := query.Get("start"); startStr != "" {
		if parsedStart, err := parseTimeParam(startStr); err == nil {
			startTime = parsedStart
		}
	}
	
	if endStr := query.Get("end"); endStr != "" {
		if parsedEnd, err := parseTimeParam(endStr); err == nil {
			endTime = parsedEnd
		}
	}
	
	// Use smart sampling based on timeframe duration
	readings, err := app.HistoryTracker.GetSensorReadingsWithSampling(babyUID, startTime, endTime)
	if err != nil {
		log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to get sensor readings")
		http.Error(w, "Failed to retrieve sensor data", http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"baby_uid":   babyUID,
		"start_time": startTime,
		"end_time":   endTime,
		"readings":   readings,
		"count":      len(readings),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHistoryEventsAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !app.HistoryTracker.IsEnabled() {
		http.Error(w, "Historical tracking disabled", http.StatusServiceUnavailable)
		return
	}
	
	// Extract baby UID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/history/events/")
	if path == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}
	
	babyUID := path
	
	// Parse query parameters with defaults
	query := r.URL.Query()
	endTime := time.Now().Unix()
	startTime := endTime - (24 * 60 * 60)
	eventType := query.Get("type")
	limit := 500
	
	if startStr := query.Get("start"); startStr != "" {
		if parsedStart, err := parseTimeParam(startStr); err == nil {
			startTime = parsedStart
		}
	}
	
	if endStr := query.Get("end"); endStr != "" {
		if parsedEnd, err := parseTimeParam(endStr); err == nil {
			endTime = parsedEnd
		}
	}
	
	if limitStr := query.Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 5000 {
			limit = parsedLimit
		}
	}
	
	events, err := app.HistoryTracker.GetEvents(babyUID, startTime, endTime, eventType, limit)
	if err != nil {
		log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to get events")
		http.Error(w, "Failed to retrieve event data", http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"baby_uid":   babyUID,
		"start_time": startTime,
		"end_time":   endTime,
		"event_type": eventType,
		"events":     events,
		"count":      len(events),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHistorySummaryAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !app.HistoryTracker.IsEnabled() {
		http.Error(w, "Historical tracking disabled", http.StatusServiceUnavailable)
		return
	}
	
	// Extract baby UID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/history/summary/")
	if path == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}
	
	babyUID := path
	
	// Parse query parameters with defaults
	query := r.URL.Query()
	endTime := time.Now().Unix()
	startTime := endTime - (24 * 60 * 60)
	
	if startStr := query.Get("start"); startStr != "" {
		if parsedStart, err := parseTimeParam(startStr); err == nil {
			startTime = parsedStart
		}
	}
	
	if endStr := query.Get("end"); endStr != "" {
		if parsedEnd, err := parseTimeParam(endStr); err == nil {
			endTime = parsedEnd
		}
	}
	
	summary, err := app.HistoryTracker.GetSummary(babyUID, startTime, endTime)
	if err != nil {
		log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to get summary")
		http.Error(w, "Failed to retrieve summary data", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func handleHistoryDayNightAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !app.HistoryTracker.IsEnabled() {
		http.Error(w, "Historical tracking disabled", http.StatusServiceUnavailable)
		return
	}
	
	// Extract baby UID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/history/day-night/")
	if path == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}
	
	babyUID := path
	
	// Parse query parameters with defaults
	query := r.URL.Query()
	endTime := time.Now().Unix()
	startTime := endTime - (24 * 60 * 60)
	
	if startStr := query.Get("start"); startStr != "" {
		if parsedStart, err := parseTimeParam(startStr); err == nil {
			startTime = parsedStart
		}
	}
	
	if endStr := query.Get("end"); endStr != "" {
		if parsedEnd, err := parseTimeParam(endStr); err == nil {
			endTime = parsedEnd
		}
	}
	
	dayNightData, err := app.HistoryTracker.GetDayNightAnalytics(babyUID, startTime, endTime)
	if err != nil {
		log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to get day/night data")
		http.Error(w, "Failed to retrieve day/night data", http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"baby_uid":   babyUID,
		"start_time": startTime,
		"end_time":   endTime,
		"day_night":  dayNightData,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHistoryResetAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !app.HistoryTracker.IsEnabled() {
		http.Error(w, "Historical tracking disabled", http.StatusServiceUnavailable)
		return
	}
	
	// Extract baby UID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/history/reset/")
	if path == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}
	
	babyUID := path
	
	_, err := app.HistoryTracker.ResetData(babyUID)
	if err != nil {
		log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to reset history data")
		http.Error(w, "Failed to reset history data", http.StatusInternalServerError)
		return
	}
	
	log.Info().Str("baby_uid", babyUID).Msg("History data reset successfully")
	
	response := map[string]interface{}{
		"success":  true,
		"baby_uid": babyUID,
		"message":  "History data reset successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to parse time parameters
func parseTimeParam(timeStr string) (int64, error) {
	// Try parsing as Unix timestamp first
	if timestamp, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
		return timestamp, nil
	}
	
	// Try parsing as RFC3339 format
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t.Unix(), nil
	}
	
	return 0, fmt.Errorf("invalid time format")
}

// streamStateToString converts a StreamState to its string representation
func streamStateToString(state baby.StreamState) string {
	switch state {
	case baby.StreamState_Alive:
		return "alive"
	case baby.StreamState_Unhealthy:
		return "unhealthy"
	case baby.StreamState_Unknown:
		return "unknown"
	default:
		return "unknown"
	}
}

func handleHealthAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract baby UID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/health/")
	babyUID := strings.TrimSuffix(path, "/")
	
	if babyUID == "" {
		http.Error(w, "baby_uid is required", http.StatusBadRequest)
		return
	}
	
	// Get baby state for WebSocket and RTMP status
	babyState := app.BabyStateManager.GetBabyState(babyUID)
	
	// Get HLS transcoding status
	var hlsStatus streaming.StreamStatus
	var hlsError *streaming.StreamError
	var hlsRunning bool
	
	if transcoder, exists := app.HLSManager.GetTranscoder(babyUID); exists {
		hlsRunning = transcoder.IsRunning()
		hlsStatus, hlsError = transcoder.GetStatus()
	} else {
		hlsStatus = streaming.StatusStopped
		hlsRunning = false
	}
	
	// Determine WebSocket status
	websocketStatus := "disconnected"
	if babyState.GetIsWebsocketAlive() {
		websocketStatus = "connected"
	}
	
	// Determine RTMP stream status using real video packet detection
	rtmpStatus := "inactive"
	if babyState.IsActivelyStreaming() {
		rtmpStatus = "active"
	} else if babyState.GetStreamState() == baby.StreamState_Alive {
		rtmpStatus = "connected_no_video"
	} else if babyState.GetStreamState() == baby.StreamState_Unhealthy {
		rtmpStatus = "unhealthy"
	}
	
	// Determine HLS status string
	hlsStatusStr := "stopped"
	if hlsRunning {
		switch hlsStatus {
		case streaming.StatusStreaming:
			hlsStatusStr = "streaming"
		case streaming.StatusConnecting:
			hlsStatusStr = "connecting"
		case streaming.StatusStarting:
			hlsStatusStr = "starting"
		case streaming.StatusError:
			hlsStatusStr = "error"
		default:
			hlsStatusStr = "unknown"
		}
	}
	
	// Calculate overall health
	overallHealth := "unhealthy"
	if websocketStatus == "connected" && rtmpStatus == "active" && hlsStatusStr == "streaming" {
		overallHealth = "healthy"
	} else if websocketStatus == "connected" && (rtmpStatus == "active" || hlsStatusStr == "streaming") {
		overallHealth = "degraded"
	} else if websocketStatus == "connected" && rtmpStatus == "connected_no_video" {
		overallHealth = "connected_no_video"
	} else if websocketStatus == "connected" || rtmpStatus == "active" || hlsRunning {
		overallHealth = "starting"
	}
	
	// Build detailed status
	details := map[string]interface{}{
		"websocket": map[string]interface{}{
			"status": websocketStatus,
			"alive":  babyState.GetIsWebsocketAlive(),
		},
		"rtmp": map[string]interface{}{
			"status":                 rtmpStatus,
			"stream_state":           streamStateToString(babyState.GetStreamState()),
			"actively_streaming":     babyState.IsActivelyStreaming(),
			"last_video_packet_time": babyState.GetLastVideoPacketTime(),
		},
		"hls": map[string]interface{}{
			"status":     hlsStatusStr,
			"is_running": hlsRunning,
		},
	}
	
	// Add HLS error if present
	if hlsError != nil {
		details["hls"].(map[string]interface{})["error"] = map[string]interface{}{
			"type":    hlsError.Type,
			"message": hlsError.Message,
			"code":    hlsError.Code,
		}
	}
	
	response := map[string]interface{}{
		"baby_uid":       babyUID,
		"overall_health": overallHealth,
		"details":        details,
		"timestamp":      time.Now().Unix(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Basic liveness check endpoint 
func handleLivenessAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(startTime).Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Readiness check endpoint for detailed service health
func handleReadinessAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	readiness := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().Unix(),
		"services":  make(map[string]interface{}),
	}

	// Check authentication status
	authReady := false
	if app.SessionStore != nil && app.SessionStore.Session != nil && app.SessionStore.Session.RefreshToken != "" {
		authReady = true
	}
	readiness["services"].(map[string]interface{})["authentication"] = map[string]interface{}{
		"ready":   authReady,
		"message": func() string {
			if authReady {
				return "Authentication configured"
			}
			return "No authentication configured"
		}(),
	}

	// Check if any babies are configured
	babiesReady := false
	babyCount := 0
	if app.SessionStore != nil && app.SessionStore.Session != nil {
		babyCount = len(app.SessionStore.Session.Babies)
		babiesReady = babyCount > 0
	}
	readiness["services"].(map[string]interface{})["babies"] = map[string]interface{}{
		"ready":      babiesReady,
		"baby_count": babyCount,
		"message": func() string {
			if babiesReady {
				return fmt.Sprintf("%d babies configured", babyCount)
			}
			return "No babies configured"
		}(),
	}

	// Check RTMP server status (assume healthy if configured)
	rtmpReady := app.Opts.RTMP != nil
	readiness["services"].(map[string]interface{})["rtmp"] = map[string]interface{}{
		"ready": rtmpReady,
		"message": func() string {
			if rtmpReady {
				return "RTMP server configured"
			}
			return "RTMP server not configured"
		}(),
	}

	// Check MQTT status
	mqttReady := app.MQTTConnection != nil
	readiness["services"].(map[string]interface{})["mqtt"] = map[string]interface{}{
		"ready": mqttReady,
		"message": func() string {
			if mqttReady {
				return "MQTT configured"
			}
			return "MQTT not configured"
		}(),
	}

	// Determine overall readiness
	overallReady := authReady && babiesReady
	if !overallReady {
		readiness["status"] = "not_ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readiness)
}

var startTime = time.Now()