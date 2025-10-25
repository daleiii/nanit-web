package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/indiefan/home_assistant_nanit/pkg/baby"
)

// ServeReact serves the React frontend instead of Go templates
func ServeReact(babies []baby.Baby, dataDir DataDirectories, stateManager *baby.StateManager, app *App) {
	port := app.Opts.HTTPPort
	
	log.Info().Msg("=== Setting up HTTP server routes for React frontend ===")
	log.Info().Int("babies_count", len(babies)).Msg("Number of babies available")

	// Serve React static files
	fs := http.FileServer(http.Dir("web"))
	
	// Handle Next.js static assets (_next/static/*)
	http.Handle("/_next/static/", http.StripPrefix("/_next/static/", http.FileServer(http.Dir("web/_next/static"))))
	
	// Handle other static files (favicon, etc.)
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	
	// Handle Next.js app routes - serve appropriate HTML files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// API routes should not serve the React app
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		
		// Handle Next.js static files directly
		if strings.HasPrefix(r.URL.Path, "/_next/") {
			// Try to serve the file directly
			filePath := filepath.Join("web", r.URL.Path)
			if _, err := os.Stat(filePath); err == nil {
				http.ServeFile(w, r, filePath)
				return
			}
		}
		
		// Try to serve Next.js route-specific HTML files first
		var routePath string
		if r.URL.Path == "/" {
			routePath = "web/index.html"
		} else {
			// For other routes like /settings, look for /settings/index.html
			routePath = filepath.Join("web", strings.TrimPrefix(r.URL.Path, "/"), "index.html")
		}
		
		// Check if route-specific HTML exists
		if _, err := os.Stat(routePath); err == nil {
			w.Header().Set("Content-Type", "text/html")
			http.ServeFile(w, r, routePath)
			return
		}
		
		// Fallback to main index.html for client-side routing
		indexPath := filepath.Join("web", "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			log.Error().Err(err).Str("path", indexPath).Msg("Next.js index.html not found")
			http.Error(w, "Frontend not built. Run 'npm run build' in frontend directory.", http.StatusNotFound)
			return
		}
		
		w.Header().Set("Content-Type", "text/html")
		http.ServeFile(w, r, indexPath)
	})

	// API endpoints - keep existing API structure
	setupAPIRoutes(babies, dataDir, stateManager, app)

	log.Info().Int("port", port).Msg("Starting HTTP server with React frontend")
	http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
}

// requireAuth is middleware that checks for web authentication
func requireAuth(app *App, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if password protection is disabled
		if !app.Opts.WebAuth.Enabled || !app.WebAuth.IsPasswordSet() {
			handler(w, r)
			return
		}

		// Check for session cookie
		cookie, err := r.Cookie("nanit_session")
		if err != nil || !app.WebAuth.ValidateSession(cookie.Value) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "authentication_required",
				"message": "Please log in to access this resource",
			})
			return
		}

		handler(w, r)
	}
}

func setupAPIRoutes(babies []baby.Baby, dataDir DataDirectories, stateManager *baby.StateManager, app *App) {
	// Status and baby data - protected by auth if enabled
	http.HandleFunc("/api/status", requireAuth(app, func(w http.ResponseWriter, r *http.Request) {
		handleStatusAPI(w, r, babies, stateManager)
	}))

	http.HandleFunc("/api/babies", requireAuth(app, func(w http.ResponseWriter, r *http.Request) {
		handleBabiesAPI(w, r, babies, stateManager)
	}))

	// Control endpoints
	http.HandleFunc("/api/control/night-light", func(w http.ResponseWriter, r *http.Request) {
		handleControlAPI(w, r, "night-light", babies, stateManager, app)
	})

	http.HandleFunc("/api/control/standby", func(w http.ResponseWriter, r *http.Request) {
		handleControlAPI(w, r, "standby", babies, stateManager, app)
	})

	// Device info endpoint
	http.HandleFunc("/api/device-info/", func(w http.ResponseWriter, r *http.Request) {
		handleDeviceInfoAPI(w, r, babies, stateManager)
	})

	// Authentication endpoints (Nanit API)
	log.Info().Msg("Registering Nanit authentication endpoints")
	http.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		handleAuthLoginAPI(w, r)
	})

	http.HandleFunc("/api/auth/verify-2fa", func(w http.ResponseWriter, r *http.Request) {
		handleAuthVerify2FAAPI(w, r, app)
	})

	http.HandleFunc("/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		handleAuthStatusAPI(w, r, app)
	})

	http.HandleFunc("/api/auth/reset", func(w http.ResponseWriter, r *http.Request) {
		handleAuthResetAPI(w, r, app)
	})

	// Web password authentication endpoints
	log.Info().Msg("Registering web password authentication endpoints")
	http.HandleFunc("/api/webauth/status", func(w http.ResponseWriter, r *http.Request) {
		handleWebAuthStatusAPI(w, r, app)
	})

	http.HandleFunc("/api/webauth/login", func(w http.ResponseWriter, r *http.Request) {
		handleWebAuthLoginAPI(w, r, app)
	})

	http.HandleFunc("/api/webauth/logout", func(w http.ResponseWriter, r *http.Request) {
		handleWebAuthLogoutAPI(w, r, app)
	})

	http.HandleFunc("/api/webauth/set-password", requireAuth(app, func(w http.ResponseWriter, r *http.Request) {
		handleSetPasswordAPI(w, r, app)
	}))

	http.HandleFunc("/api/webauth/change-password", requireAuth(app, func(w http.ResponseWriter, r *http.Request) {
		handleChangePasswordAPI(w, r, app)
	}))

	http.HandleFunc("/api/webauth/remove-password", requireAuth(app, func(w http.ResponseWriter, r *http.Request) {
		handleRemovePasswordAPI(w, r, app)
	}))

	// HLS streaming endpoints
	http.HandleFunc("/api/stream/hls/", func(w http.ResponseWriter, r *http.Request) {
		handleHLSStreamAPI(w, r, app)
	})

	http.HandleFunc("/api/stream/start/", func(w http.ResponseWriter, r *http.Request) {
		handleStreamStartAPI(w, r, app)
	})

	http.HandleFunc("/api/stream/stop/", func(w http.ResponseWriter, r *http.Request) {
		handleStreamStopAPI(w, r, app)
	})

	http.HandleFunc("/api/stream/status/", func(w http.ResponseWriter, r *http.Request) {
		handleStreamStatusAPI(w, r, app)
	})

	// Historical data endpoints
	http.HandleFunc("/api/history/sensor/", func(w http.ResponseWriter, r *http.Request) {
		handleHistorySensorAPI(w, r, app)
	})

	http.HandleFunc("/api/history/events/", func(w http.ResponseWriter, r *http.Request) {
		handleHistoryEventsAPI(w, r, app)
	})

	http.HandleFunc("/api/history/summary/", func(w http.ResponseWriter, r *http.Request) {
		handleHistorySummaryAPI(w, r, app)
	})

	http.HandleFunc("/api/history/day-night/", func(w http.ResponseWriter, r *http.Request) {
		handleHistoryDayNightAPI(w, r, app)
	})

	http.HandleFunc("/api/history/reset/", func(w http.ResponseWriter, r *http.Request) {
		handleHistoryResetAPI(w, r, app)
	})

	// Health endpoints
	http.HandleFunc("/api/health/", func(w http.ResponseWriter, r *http.Request) {
		handleHealthAPI(w, r, app)
	})
	
	// Basic liveness check (no authentication required)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		handleLivenessAPI(w, r)
	})
	
	// Readiness check with detailed service status (no authentication required)
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		handleReadinessAPI(w, r, app)
	})

	// Video files
	http.Handle("/video/", http.StripPrefix("/video/", http.FileServer(http.Dir(dataDir.VideoDir))))

	// Dummy log handler - useful for receiving logs from cam
	http.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Join(dataDir.LogDir, fmt.Sprintf("camlogs-%v.tar.gz", time.Now().Format(time.RFC3339)))

		log.Info().Str("file", filename).Msg("Saving log to file")
		defer r.Body.Close()

		out, err := os.Create(filename)
		if err != nil {
			log.Error().Str("file", filename).Err(err).Msg("Unable to create file")
		}

		defer out.Close()

		_, err = io.Copy(out, r.Body)

		if err != nil {
			log.Error().Str("file", filename).Err(err).Msg("Unable to save received log file")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

// Web authentication API handlers

func handleWebAuthStatusAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"password_protection_enabled": app.Opts.WebAuth.Enabled,
		"password_set":                app.WebAuth.IsPasswordSet(),
		"authenticated":               false,
	}

	// Check if user is authenticated
	if app.Opts.WebAuth.Enabled && app.WebAuth.IsPasswordSet() {
		cookie, err := r.Cookie("nanit_session")
		if err == nil && app.WebAuth.ValidateSession(cookie.Value) {
			response["authenticated"] = true
		}
	} else {
		// No password protection, consider authenticated
		response["authenticated"] = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleWebAuthLoginAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if !app.Opts.WebAuth.Enabled {
		http.Error(w, "Password protection is disabled", http.StatusBadRequest)
		return
	}

	if !app.WebAuth.IsPasswordSet() {
		http.Error(w, "No password is set", http.StatusBadRequest)
		return
	}

	if !app.WebAuth.VerifyPassword(requestData.Password) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_password",
			"message": "Invalid password",
		})
		return
	}

	// Create session
	sessionID, err := app.WebAuth.CreateSession()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create session")
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "nanit_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true if using HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Login successful",
	})
}

func handleWebAuthLogoutAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session cookie and invalidate it
	cookie, err := r.Cookie("nanit_session")
	if err == nil {
		app.WebAuth.InvalidateSession(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "nanit_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Delete cookie
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logout successful",
	})
}

func handleSetPasswordAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if !app.Opts.WebAuth.Enabled {
		http.Error(w, "Password protection is disabled", http.StatusBadRequest)
		return
	}

	if app.WebAuth.IsPasswordSet() {
		http.Error(w, "Password is already set. Use change-password instead.", http.StatusBadRequest)
		return
	}

	err := app.WebAuth.SetPassword(requestData.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Info().Msg("Password protection enabled")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Password set successfully",
	})
}

func handleChangePasswordAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if !app.Opts.WebAuth.Enabled {
		http.Error(w, "Password protection is disabled", http.StatusBadRequest)
		return
	}

	if !app.WebAuth.IsPasswordSet() {
		http.Error(w, "No password is currently set", http.StatusBadRequest)
		return
	}

	// Verify current password
	if !app.WebAuth.VerifyPassword(requestData.CurrentPassword) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_current_password",
			"message": "Current password is incorrect",
		})
		return
	}

	// Set new password
	err := app.WebAuth.SetPassword(requestData.NewPassword)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Info().Msg("Password changed successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Password changed successfully",
	})
}

func handleRemovePasswordAPI(w http.ResponseWriter, r *http.Request, app *App) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if !app.Opts.WebAuth.Enabled {
		http.Error(w, "Password protection is disabled", http.StatusBadRequest)
		return
	}

	if !app.WebAuth.IsPasswordSet() {
		http.Error(w, "No password is currently set", http.StatusBadRequest)
		return
	}

	// Verify current password
	if !app.WebAuth.VerifyPassword(requestData.Password) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_password",
			"message": "Password is incorrect",
		})
		return
	}

	// Remove password
	err := app.WebAuth.RemovePassword()
	if err != nil {
		log.Error().Err(err).Msg("Failed to remove password")
		http.Error(w, "Failed to remove password", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Password protection disabled successfully",
	})
}