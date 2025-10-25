package app

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/indiefan/home_assistant_nanit/pkg/baby"
	"github.com/indiefan/home_assistant_nanit/pkg/client"
	"github.com/indiefan/home_assistant_nanit/pkg/history"
	"github.com/indiefan/home_assistant_nanit/pkg/message"
	"github.com/indiefan/home_assistant_nanit/pkg/mqtt"
	"github.com/indiefan/home_assistant_nanit/pkg/rtmpserver"
	"github.com/indiefan/home_assistant_nanit/pkg/session"
	"github.com/indiefan/home_assistant_nanit/pkg/streaming"
	"github.com/indiefan/home_assistant_nanit/pkg/utils"
	"github.com/indiefan/home_assistant_nanit/pkg/webauth"
)

// App - application container
type App struct {
	Opts             Opts
	SessionStore     *session.Store
	BabyStateManager *baby.StateManager
	RestClient       *client.NanitClient
	MQTTConnection   *mqtt.Connection
	HLSManager       *streaming.HLSManager
	HistoryTracker   *history.Tracker
	WebAuth          *webauth.WebAuth
	connections      map[string]*client.WebsocketConnection
	connectionsMutex sync.RWMutex
	mainContext      utils.GracefulContext // Store main application context
}

// NewApp - constructor
func NewApp(opts Opts) (*App, error) {
	sessionStore, err := session.InitSessionStore(opts.SessionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session store: %w", err)
	}

	instance := &App{
		Opts:             opts,
		BabyStateManager: baby.NewStateManager(),
		SessionStore:     sessionStore,
		RestClient: &client.NanitClient{
			Email:        opts.NanitCredentials.Email,
			Password:     opts.NanitCredentials.Password,
			RefreshToken: opts.NanitCredentials.RefreshToken,
			SessionStore: sessionStore,
		},
		HLSManager:  streaming.NewHLSManager(opts.DataDirectories.BaseDir + "/hls"),
		WebAuth:     webauth.NewWebAuth(opts.WebAuth.PasswordFile),
		connections: make(map[string]*client.WebsocketConnection),
	}

	if opts.MQTT != nil {
		instance.MQTTConnection = mqtt.NewConnection(*opts.MQTT)
	}

	// Initialize historical data tracker
	if historyTracker, err := history.NewTracker(opts.DataDirectories.HistoryDir, opts.History.Enabled); err != nil {
		log.Error().Err(err).Msg("Failed to initialize historical data tracker")
		// Continue without historical tracking
		instance.HistoryTracker = &history.Tracker{}
	} else {
		instance.HistoryTracker = historyTracker
	}

	return instance, nil
}

// Run - application main loop
func (app *App) Run(ctx utils.GracefulContext) {
	// Store main context for later use
	app.mainContext = ctx
	
	// Set up historical data tracking callback
	app.setupHistoryTracking()
	// Check if we have valid authentication
	hasValidAuth := false
	if app.SessionStore != nil && app.SessionStore.Session != nil && app.SessionStore.Session.RefreshToken != "" {
		// Try to authorize - if it fails, we'll run in web-only mode
		defer func() {
			if r := recover(); r != nil {
				log.Warn().Interface("error", r).Msg("Authorization failed, running in web-only mode")
				hasValidAuth = false
			}
		}()
		
		if err := app.RestClient.MaybeAuthorize(false); err != nil {
			log.Error().Err(err).Msg("Authentication failed")
			hasValidAuth = false
		} else {
			if _, err := app.RestClient.EnsureBabies(); err != nil {
				log.Error().Err(err).Msg("Failed to fetch babies")
				hasValidAuth = false
			} else {
				hasValidAuth = true
			}
		}
	} else {
		log.Info().Msg("No valid authentication found - running in web-only mode for initial setup")
	}

	// Always start HTTP server for web UI (including setup)
	var babies []baby.Baby
	if hasValidAuth && app.SessionStore.Session != nil {
		babies = app.SessionStore.Session.Babies
	}
	
	if app.Opts.HTTPEnabled {
		go ServeReact(babies, app.Opts.DataDirectories, app.BabyStateManager, app)
	}

	// Only start RTMP/MQTT/WebSocket if we have valid auth
	if hasValidAuth {
		// RTMP
		if app.Opts.RTMP != nil {
			go func() {
				if err := rtmpserver.StartRTMPServer(app.Opts.RTMP.ListenAddr, app.BabyStateManager); err != nil {
					log.Error().Err(err).Msg("RTMP server failed to start or crashed")
				}
			}()
		}

		// MQTT
		if app.MQTTConnection != nil {
			ctx.RunAsChild(func(childCtx utils.GracefulContext) {
				app.MQTTConnection.Run(app.BabyStateManager, childCtx)
			})
		}

		// Start reading the data from the stream
		for _, babyInfo := range app.SessionStore.Session.Babies {
			_babyInfo := babyInfo
			ctx.RunAsChild(func(childCtx utils.GracefulContext) {
				app.handleBaby(_babyInfo, childCtx)
			})
		}
		
		log.Info().Msg("All services started with authentication")
	} else {
		log.Info().Msg("Web server started - visit http://localhost:8080/setup to configure authentication")
	}

	<-ctx.Done()
}

func (app *App) handleBaby(baby baby.Baby, ctx utils.GracefulContext) {
	if app.Opts.RTMP != nil || app.MQTTConnection != nil {
		// Websocket connection
		ws := client.NewWebsocketConnectionManager(baby.UID, baby.CameraUID, app.SessionStore.Session, app.RestClient, app.BabyStateManager)

		ws.WithReadyConnection(func(conn *client.WebsocketConnection, childCtx utils.GracefulContext) {
			// Register connection
			app.registerConnection(baby.UID, conn)
			defer func() {
				app.unregisterConnection(baby.UID)
				// Gracefully stop streaming when WebSocket disconnects
				if app.Opts.RTMP != nil && app.Opts.RTMP.AutoStart {
					app.autoStopStreaming(baby.UID, conn)
				}
			}()
			
			// Auto-start streaming if RTMP is enabled and auto-start is configured
			if app.Opts.RTMP != nil && app.Opts.RTMP.AutoStart {
				log.Info().Str("baby_uid", baby.UID).Msg("Auto-starting RTMP stream")
				go app.autoStartStreaming(baby.UID, conn)
				
				// Start persistent retry mechanism for failed connections
				go app.startStreamingRetryMonitor(baby.UID, childCtx)
			}
			
			app.runWebsocket(baby.UID, conn, childCtx)
		})

		if app.Opts.EventPolling.Enabled {
			go app.pollMessages(baby.UID, app.BabyStateManager)
		}

		ctx.RunAsChild(func(childCtx utils.GracefulContext) {
			ws.RunWithinContext(childCtx)
		})
	}

	<-ctx.Done()
}

func (app *App) pollMessages(babyUID string, babyStateManager *baby.StateManager) {
	newMessages, err := app.RestClient.FetchNewMessages(babyUID, app.Opts.EventPolling.MessageTimeout)
	if err != nil {
		log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to fetch new messages")
		// Continue with empty messages rather than crash
		newMessages = []message.Message{}
	}

	for _, msg := range newMessages {
		switch msg.Type {
		case message.SoundEventMessageType:
			go babyStateManager.NotifySoundSubscribers(babyUID, time.Time(msg.Time))
			break
		case message.MotionEventMessageType:
			go babyStateManager.NotifyMotionSubscribers(babyUID, time.Time(msg.Time))
			break
		}
	}

	// wait for the specified interval
	time.Sleep(app.Opts.EventPolling.PollingInterval)
	app.pollMessages(babyUID, babyStateManager)
}

func (app *App) runWebsocket(babyUID string, conn *client.WebsocketConnection, childCtx utils.GracefulContext) {
	// Reading sensor data
	conn.RegisterMessageHandler(func(m *client.Message, conn *client.WebsocketConnection) {
		// Sensor request initiated by us on start (or some other client, we don't care)
		if *m.Type == client.Message_RESPONSE && m.Response != nil {
			if *m.Response.RequestType == client.RequestType_GET_SENSOR_DATA && len(m.Response.SensorData) > 0 {
				processSensorData(babyUID, m.Response.SensorData, app.BabyStateManager)
			} else if *m.Response.RequestType == client.RequestType_GET_CONTROL && m.Response.Control != nil {
				processLight(babyUID, m.Response.Control, app.BabyStateManager)
			} else if *m.Response.RequestType == client.RequestType_GET_SETTINGS && m.Response.Settings != nil {
				processStandby(babyUID, m.Response.Settings, app.BabyStateManager)
			} else if *m.Response.RequestType == client.RequestType_GET_STATUS && m.Response.Status != nil {
				processStatus(babyUID, m.Response.Status, app.BabyStateManager)
			}
		} else

		// Communication initiated from a cam
		// Note: it sends the updates periodically on its own + whenever some significant change occurs
		if *m.Type == client.Message_REQUEST && m.Request != nil {
			if *m.Request.Type == client.RequestType_PUT_SENSOR_DATA && len(m.Request.SensorData_) > 0 {
				processSensorData(babyUID, m.Request.SensorData_, app.BabyStateManager)
			} else if *m.Request.Type == client.RequestType_PUT_CONTROL && m.Request.Control != nil {
				processLight(babyUID, m.Request.Control, app.BabyStateManager)
			} else if *m.Request.Type == client.RequestType_PUT_SETTINGS && m.Request.Settings != nil {
				processStandby(babyUID, m.Request.Settings, app.BabyStateManager)
			}
		}
	})

	if app.Opts.MQTT != nil && app.MQTTConnection != nil {
		app.MQTTConnection.RegisterLightHandler(func(enabled bool) {
			sendLightCommand(enabled, conn)
		})
		app.MQTTConnection.RegisterStandyHandler(func(enabled bool) {
			sendStandbyCommand(enabled, conn)
		})
	}

	// Get the initial state of the light
	conn.SendRequest(client.RequestType_GET_CONTROL, &client.Request{GetControl_: &client.GetControl{
		NightLight: utils.ConstRefBool(true),
	}})

	// Ask for sensor data (initial request)
	conn.SendRequest(client.RequestType_GET_SENSOR_DATA, &client.Request{
		GetSensorData: &client.GetSensorData{
			All: utils.ConstRefBool(true),
		},
	})

	// Ask for status
	conn.SendRequest(client.RequestType_GET_STATUS, &client.Request{
		GetStatus_: &client.GetStatus{
			All: utils.ConstRefBool(true),
		},
	})

	// Ask for settings to get device configuration
	conn.SendRequest(client.RequestType_GET_SETTINGS, &client.Request{})

	// Ask for logs
	// conn.SendRequest(client.RequestType_GET_LOGS, &client.Request{
	// 	GetLogs: &client.GetLogs{
	// 		Url: utils.ConstRefStr("http://192.168.3.234:8080/log"),
	// 	},
	// })

	var cleanup func()

	// Local streaming
	if app.Opts.RTMP != nil {
		initializeLocalStreaming := func() {
			requestLocalStreaming(babyUID, app.getLocalStreamURL(babyUID), client.Streaming_STARTED, conn, app.BabyStateManager)
		}

		// Watch for stream liveness change
		unsubscribe := app.BabyStateManager.Subscribe(func(updatedBabyUID string, stateUpdate baby.State) {
			// Do another streaming request if stream just turned unhealthy
			if updatedBabyUID == babyUID && stateUpdate.StreamState != nil && *stateUpdate.StreamState == baby.StreamState_Unhealthy {
				// Prevent duplicate request if we already received failure
				if app.BabyStateManager.GetBabyState(babyUID).GetStreamRequestState() != baby.StreamRequestState_RequestFailed {
					go initializeLocalStreaming()
				}
			}
		})

		cleanup = func() {
			// Stop listening for stream liveness change
			unsubscribe()

			// Stop local streaming
			state := app.BabyStateManager.GetBabyState(babyUID)
			if state.GetIsWebsocketAlive() && state.GetStreamState() == baby.StreamState_Alive {
				requestLocalStreaming(babyUID, app.getLocalStreamURL(babyUID), client.Streaming_STOPPED, conn, app.BabyStateManager)
			}
		}

		// Initialize local streaming upon connection if we know that the stream is not alive
		babyState := app.BabyStateManager.GetBabyState(babyUID)
		if babyState.GetStreamState() != baby.StreamState_Alive {
			if babyState.GetStreamRequestState() != baby.StreamRequestState_Requested || babyState.GetStreamState() == baby.StreamState_Unhealthy {
				go initializeLocalStreaming()
			}
		}
	}

	<-childCtx.Done()
	if cleanup != nil {
		cleanup()
	}
}

func (app *App) getRemoteStreamURL(babyUID string) string {
	return fmt.Sprintf("rtmps://media-secured.nanit.com/nanit/%v.%v", babyUID, app.SessionStore.Session.AuthToken)
}

func (app *App) getLocalStreamURL(babyUID string) string {
	if app.Opts.RTMP != nil {
		tpl := "rtmp://{publicAddr}/local/{babyUid}"
		return strings.NewReplacer("{publicAddr}", app.Opts.RTMP.PublicAddr, "{babyUid}", babyUID).Replace(tpl)
	}

	return ""
}

// Connection management methods for WebSocket connections
func (app *App) registerConnection(babyUID string, conn *client.WebsocketConnection) {
	app.connectionsMutex.Lock()
	defer app.connectionsMutex.Unlock()
	app.connections[babyUID] = conn
}

func (app *App) unregisterConnection(babyUID string) {
	app.connectionsMutex.Lock()
	defer app.connectionsMutex.Unlock()
	delete(app.connections, babyUID)
}

func (app *App) getConnection(babyUID string) *client.WebsocketConnection {
	app.connectionsMutex.RLock()
	defer app.connectionsMutex.RUnlock()
	return app.connections[babyUID]
}

// RefreshAuthentication - reload session after successful web authentication
func (app *App) RefreshAuthentication() error {
	// Reinitialize session store to pick up new session file
	sessionStore, err := session.InitSessionStore(app.Opts.SessionFile)
	if err != nil {
		return fmt.Errorf("failed to reinitialize session store: %w", err)
	}
	app.SessionStore = sessionStore
	
	// Update RestClient with new session
	if app.SessionStore.Session != nil {
		app.RestClient.SessionStore = app.SessionStore
		if app.SessionStore.Session.RefreshToken != "" {
			app.RestClient.RefreshToken = app.SessionStore.Session.RefreshToken
		}
	}
	
	log.Info().Msg("Authentication refreshed successfully")
	return nil
}

// StartMonitoringServices - start all monitoring services after authentication
func (app *App) StartMonitoringServices() {
	// Use the main application context stored during Run()
	ctx := app.mainContext
	if ctx == nil {
		log.Error().Msg("Cannot start monitoring services: main context not available")
		return
	}
	log.Info().Msg("Starting monitoring services after authentication...")
	
	// Force refresh authorization and fetch babies (token may have expired since web auth)
	if err := app.RestClient.MaybeAuthorize(true); err != nil {  // Force refresh
		log.Error().Err(err).Msg("Failed to refresh authorization")
		return
	}
	if _, err := app.RestClient.EnsureBabies(); err != nil {
		log.Error().Err(err).Msg("Failed to ensure babies after authorization")
		return
	}
	
	if app.SessionStore.Session == nil || len(app.SessionStore.Session.Babies) == 0 {
		log.Warn().Msg("No babies found after authentication")
		return
	}
	
	log.Info().Int("babies_count", len(app.SessionStore.Session.Babies)).Msg("Found babies, starting services")
	
	// Start RTMP server if configured
	if app.Opts.RTMP != nil {
		go func() {
			if err := rtmpserver.StartRTMPServer(app.Opts.RTMP.ListenAddr, app.BabyStateManager); err != nil {
				log.Error().Err(err).Msg("RTMP server failed to start or crashed")
			}
		}()
		log.Info().Msg("RTMP server startup initiated")
	}
	
	// Start MQTT if configured  
	if app.MQTTConnection != nil {
		ctx.RunAsChild(func(childCtx utils.GracefulContext) {
			app.MQTTConnection.Run(app.BabyStateManager, childCtx)
		})
		log.Info().Msg("MQTT connection started")
	}
	
	// Start baby monitoring for each baby (use same pattern as original Run method)
	for _, babyInfo := range app.SessionStore.Session.Babies {
		_babyInfo := babyInfo
		ctx.RunAsChild(func(childCtx utils.GracefulContext) {
			app.handleBaby(_babyInfo, childCtx)
		})
		log.Info().Str("baby_uid", _babyInfo.UID).Str("name", _babyInfo.Name).Msg("Started monitoring baby")
	}
	
	log.Info().Msg("All monitoring services started successfully")
	
	// Set up cleanup handler for graceful shutdown
	ctx.RunAsChild(func(childCtx utils.GracefulContext) {
		<-childCtx.Done()
		
		log.Info().Msg("Shutting down application...")
		
		if app.HistoryTracker != nil {
			if err := app.HistoryTracker.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close history tracker")
			}
		}
		if app.HLSManager != nil {
			app.HLSManager.StopAll()
		}
		log.Info().Msg("Application cleanup completed")
	})
}

// autoStartStreaming automatically starts RTMP streaming and HLS transcoding when a baby comes online
func (app *App) autoStartStreaming(babyUID string, conn *client.WebsocketConnection) {
	// Give the WebSocket connection a moment to fully establish
	time.Sleep(2 * time.Second)
	
	// Get the RTMP URL for this baby
	streamURL := app.getLocalStreamURL(babyUID)
	if streamURL == "" {
		log.Error().Str("baby_uid", babyUID).Msg("Cannot auto-start streaming: no RTMP URL available")
		return
	}
	
	log.Info().
		Str("baby_uid", babyUID).
		Str("rtmp_url", streamURL).
		Msg("Auto-starting RTMP streaming and HLS transcoding")
	
	// Start RTMP streaming first
	requestLocalStreaming(babyUID, streamURL, client.Streaming_STARTED, conn, app.BabyStateManager)
	
	// Start HLS transcoding for instant playback
	if app.HLSManager != nil {
		// Give RTMP stream a moment to establish before starting HLS transcoding
		go func() {
			time.Sleep(3 * time.Second)
			
			if err := app.HLSManager.StartTranscoding(babyUID, streamURL); err != nil {
				log.Error().
					Err(err).
					Str("baby_uid", babyUID).
					Msg("Failed to auto-start HLS transcoding")
			} else {
				log.Info().
					Str("baby_uid", babyUID).
					Msg("Auto-started HLS transcoding for instant playback")
			}
		}()
	}
}

// autoStopStreaming gracefully stops RTMP streaming and HLS transcoding when WebSocket disconnects
func (app *App) autoStopStreaming(babyUID string, conn *client.WebsocketConnection) {
	// Get the RTMP URL for this baby
	streamURL := app.getLocalStreamURL(babyUID)
	if streamURL == "" {
		log.Debug().Str("baby_uid", babyUID).Msg("No RTMP URL available for auto-stop")
		return
	}
	
	log.Info().
		Str("baby_uid", babyUID).
		Str("rtmp_url", streamURL).
		Msg("Auto-stopping RTMP streaming and HLS transcoding due to WebSocket disconnect")
	
	// Send stop streaming command to camera (best effort - may not reach if connection is already dead)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Debug().Interface("error", r).Msg("Expected error stopping stream on dead connection")
			}
		}()
		requestLocalStreaming(babyUID, streamURL, client.Streaming_STOPPED, conn, app.BabyStateManager)
	}()
	
	// Stop HLS transcoding to prevent orphaned processes
	if app.HLSManager != nil {
		app.HLSManager.StopTranscoding(babyUID)
		log.Info().Str("baby_uid", babyUID).Msg("Auto-stopped HLS transcoding")
	}
	
	// Update state to reflect stream is no longer active
	app.BabyStateManager.Update(babyUID, *baby.NewState().SetStreamState(baby.StreamState_Unhealthy))
}

// setupHistoryTracking configures historical data tracking for state changes
func (app *App) setupHistoryTracking() {
	if !app.HistoryTracker.IsEnabled() {
		log.Debug().Msg("Historical tracking disabled")
		return
	}

	// Set up callback to track state changes
	app.BabyStateManager.SetHistoryCallback(func(babyUID string, state baby.State) {
		// Track sensor data (temperature, humidity, night mode)
		if state.TemperatureMilli != nil || state.HumidityMilli != nil || state.IsNight != nil {
			if err := app.HistoryTracker.TrackSensorData(babyUID, state); err != nil {
				log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to track sensor data")
			}
		}

		// Track motion events
		if state.MotionTimestamp != nil {
			if err := app.HistoryTracker.TrackEvent(babyUID, "motion", int64(*state.MotionTimestamp)); err != nil {
				log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to track motion event")
			}
		}

		// Track sound events  
		if state.SoundTimestamp != nil {
			if err := app.HistoryTracker.TrackEvent(babyUID, "sound", int64(*state.SoundTimestamp)); err != nil {
				log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to track sound event")
			}
		}

		// Track night light state changes
		if state.NightLight != nil {
			if err := app.HistoryTracker.TrackStateChange(babyUID, "night_light", *state.NightLight); err != nil {
				log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to track night light state change")
			}
		}

		// Track standby state changes
		if state.Standby != nil {
			if err := app.HistoryTracker.TrackStateChange(babyUID, "standby", *state.Standby); err != nil {
				log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to track standby state change")
			}
		}
	})

	log.Info().Msg("Historical data tracking enabled")

	// Set up periodic cleanup if enabled
	if app.Opts.History.CleanupEnabled {
		app.setupHistoryCleanup()
	}
}

// setupHistoryCleanup starts a background routine for cleaning up old historical data
func (app *App) setupHistoryCleanup() {
	if !app.HistoryTracker.IsEnabled() {
		return
	}

	app.mainContext.RunAsChild(func(childCtx utils.GracefulContext) {
		ticker := time.NewTicker(24 * time.Hour) // Run cleanup daily
		defer ticker.Stop()

		log.Info().Int("retention_days", app.Opts.History.RetentionDays).
			Msg("Starting historical data cleanup routine")

		for {
			select {
			case <-ticker.C:
				if err := app.HistoryTracker.Cleanup(app.Opts.History.RetentionDays); err != nil {
					log.Error().Err(err).Msg("Failed to cleanup historical data")
				}

			case <-childCtx.Done():
				log.Info().Msg("Historical data cleanup routine stopped")
				return
			}
		}
	})
}

// startStreamingRetryMonitor continuously monitors and retries failed streaming connections
func (app *App) startStreamingRetryMonitor(babyUID string, ctx utils.GracefulContext) {
	retryInterval := 60 * time.Second // Retry every 60 seconds
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	log.Info().
		Str("baby_uid", babyUID).
		Dur("retry_interval", retryInterval).
		Msg("Starting streaming retry monitor")

	for {
		select {
		case <-ticker.C:
			// Check if we should retry streaming
			if app.shouldRetryStreaming(babyUID) {
				conn := app.getConnection(babyUID)
				if conn != nil {
					log.Info().
						Str("baby_uid", babyUID).
						Msg("Retrying streaming connection due to previous failure")
					
					go app.retryStreaming(babyUID, conn)
				}
			}

		case <-ctx.Done():
			log.Info().
				Str("baby_uid", babyUID).
				Msg("Streaming retry monitor stopped")
			return
		}
	}
}

// shouldRetryStreaming determines if we should retry streaming for a baby
func (app *App) shouldRetryStreaming(babyUID string) bool {
	// Only retry if RTMP auto-start is enabled
	if app.Opts.RTMP == nil || !app.Opts.RTMP.AutoStart {
		return false
	}

	babyState := app.BabyStateManager.GetBabyState(babyUID)
	
	// Only retry if:
	// 1. WebSocket is alive (connection exists)
	// 2. Stream request failed (connection limit or other failure)
	// 3. Stream is not currently alive (no active stream)
	return babyState.GetIsWebsocketAlive() &&
		babyState.GetStreamRequestState() == baby.StreamRequestState_RequestFailed &&
		babyState.GetStreamState() != baby.StreamState_Alive
}

// retryStreaming attempts to restart streaming after a failure
func (app *App) retryStreaming(babyUID string, conn *client.WebsocketConnection) {
	streamURL := app.getLocalStreamURL(babyUID)
	if streamURL == "" {
		log.Error().Str("baby_uid", babyUID).Msg("Cannot retry streaming: no RTMP URL available")
		return
	}

	log.Info().
		Str("baby_uid", babyUID).
		Str("rtmp_url", streamURL).
		Msg("Retrying RTMP streaming and HLS transcoding")

	// Reset the failed state before retrying
	app.BabyStateManager.Update(babyUID, *baby.NewState().SetStreamRequestState(baby.StreamRequestState_NotRequested))

	// Retry RTMP streaming
	requestLocalStreaming(babyUID, streamURL, client.Streaming_STARTED, conn, app.BabyStateManager)

	// Start HLS transcoding if not already running
	if app.HLSManager != nil {
		if transcoder, exists := app.HLSManager.GetTranscoder(babyUID); !exists || !transcoder.IsRunning() {
			// Give RTMP stream a moment to establish before starting HLS transcoding
			go func() {
				time.Sleep(3 * time.Second)
				
				if err := app.HLSManager.StartTranscoding(babyUID, streamURL); err != nil {
					log.Error().
						Err(err).
						Str("baby_uid", babyUID).
						Msg("Failed to start HLS transcoding during retry")
				} else {
					log.Info().
						Str("baby_uid", babyUID).
						Msg("Started HLS transcoding during streaming retry")
				}
			}()
		}
	}
}
