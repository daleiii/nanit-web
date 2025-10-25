package history

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
	"github.com/indiefan/home_assistant_nanit/pkg/baby"
)

//go:embed schema.sql
var schemaSQL embed.FS

// Tracker manages historical data storage and retrieval
type Tracker struct {
	db       *sql.DB
	dbPath   string
	enabled  bool
}

// SensorReading represents a point-in-time sensor measurement
type SensorReading struct {
	ID               int64     `json:"id"`
	BabyUID          string    `json:"baby_uid"`
	Timestamp        int64     `json:"timestamp"`
	TemperatureCelsius *float64 `json:"temperature_celsius,omitempty"`
	HumidityPercent   *float64 `json:"humidity_percent,omitempty"`
	IsNight          *bool     `json:"is_night,omitempty"`
	CreatedAt        int64     `json:"created_at"`
}

// Event represents a motion or sound event
type Event struct {
	ID        int64  `json:"id"`
	BabyUID   string `json:"baby_uid"`
	Timestamp int64  `json:"timestamp"`
	EventType string `json:"event_type"` // "motion" or "sound"
	CreatedAt int64  `json:"created_at"`
}

// StateChange represents a change in baby state (night light, standby)
type StateChange struct {
	ID         int64  `json:"id"`
	BabyUID    string `json:"baby_uid"`
	Timestamp  int64  `json:"timestamp"`
	StateType  string `json:"state_type"`  // "night_light" or "standby"
	StateValue bool   `json:"state_value"`
	CreatedAt  int64  `json:"created_at"`
}

// HistoricalSummary provides aggregated data for a time period
type HistoricalSummary struct {
	BabyUID            string  `json:"baby_uid"`
	StartTime          int64   `json:"start_time"`
	EndTime            int64   `json:"end_time"`
	AvgTemperature     *float64 `json:"avg_temperature,omitempty"`
	MinTemperature     *float64 `json:"min_temperature,omitempty"`
	MaxTemperature     *float64 `json:"max_temperature,omitempty"`
	AvgHumidity        *float64 `json:"avg_humidity,omitempty"`
	MinHumidity        *float64 `json:"min_humidity,omitempty"`
	MaxHumidity        *float64 `json:"max_humidity,omitempty"`
	MotionEventCount   int64   `json:"motion_event_count"`
	SoundEventCount    int64   `json:"sound_event_count"`
	NightLightChanges  int64   `json:"night_light_changes"`
	StandbyChanges     int64   `json:"standby_changes"`
	DayModeMinutes     int64   `json:"day_mode_minutes"`
	NightModeMinutes   int64   `json:"night_mode_minutes"`
	DayModePercentage  float64 `json:"day_mode_percentage"`
	NightModePercentage float64 `json:"night_mode_percentage"`
}

// DayNightAnalytics provides detailed day/night mode analysis
type DayNightAnalytics struct {
	BabyUID              string                    `json:"baby_uid"`
	StartTime            int64                     `json:"start_time"`
	EndTime              int64                     `json:"end_time"`
	TotalMinutes         int64                     `json:"total_minutes"`
	DayModeMinutes       int64                     `json:"day_mode_minutes"`
	NightModeMinutes     int64                     `json:"night_mode_minutes"`
	UnknownModeMinutes   int64                     `json:"unknown_mode_minutes"`
	DayModePercentage    float64                   `json:"day_mode_percentage"`
	NightModePercentage  float64                   `json:"night_mode_percentage"`
	UnknownModePercentage float64                  `json:"unknown_mode_percentage"`
	ModeTransitions      int64                     `json:"mode_transitions"`
	DayNightChanges      []DayNightChange          `json:"day_night_changes"`
}

// DayNightChange represents a transition between day and night mode
type DayNightChange struct {
	Timestamp    int64 `json:"timestamp"`
	FromNight    bool  `json:"from_night"`
	ToNight      bool  `json:"to_night"`
	DurationMins int64 `json:"duration_mins"`
}

// NewTracker creates a new historical data tracker
func NewTracker(dataDir string, enabled bool) (*Tracker, error) {
	if !enabled {
		log.Info().Msg("Historical data tracking disabled")
		return &Tracker{enabled: false}, nil
	}

	dbPath := filepath.Join(dataDir, "history.db")
	
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	tracker := &Tracker{
		db:      db,
		dbPath:  dbPath,
		enabled: true,
	}

	// Initialize database schema
	if err := tracker.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}

	log.Info().Str("db_path", dbPath).Msg("Historical data tracking initialized")
	return tracker, nil
}

// initSchema creates the database tables
func (t *Tracker) initSchema() error {
	schemaBytes, err := schemaSQL.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %v", err)
	}

	if _, err := t.db.Exec(string(schemaBytes)); err != nil {
		return fmt.Errorf("failed to execute schema: %v", err)
	}

	return nil
}

// Close closes the database connection
func (t *Tracker) Close() error {
	if !t.enabled || t.db == nil {
		return nil
	}
	
	log.Info().Msg("Closing historical data tracker")
	return t.db.Close()
}

// TrackSensorData records sensor readings (temperature, humidity, night mode)
func (t *Tracker) TrackSensorData(babyUID string, state baby.State) error {
	if !t.enabled {
		return nil
	}

	// Only record if we have sensor data to record
	if state.TemperatureMilli == nil && state.HumidityMilli == nil && state.IsNight == nil {
		return nil
	}

	timestamp := time.Now().Unix()
	
	var temperature *float64
	var humidity *float64
	
	if state.TemperatureMilli != nil {
		temp := float64(*state.TemperatureMilli) / 1000.0
		temperature = &temp
	}
	
	if state.HumidityMilli != nil {
		hum := float64(*state.HumidityMilli) / 1000.0
		humidity = &hum
	}

	query := `
		INSERT INTO sensor_readings (baby_uid, timestamp, temperature_celsius, humidity_percent, is_night)
		VALUES (?, ?, ?, ?, ?)
	`
	
	_, err := t.db.Exec(query, babyUID, timestamp, temperature, humidity, state.IsNight)
	if err != nil {
		log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to record sensor data")
		return err
	}

	log.Debug().
		Str("baby_uid", babyUID).
		Interface("temperature", temperature).
		Interface("humidity", humidity).
		Interface("is_night", state.IsNight).
		Msg("Recorded sensor reading")
		
	return nil
}

// TrackEvent records motion or sound events
func (t *Tracker) TrackEvent(babyUID string, eventType string, eventTimestamp int64) error {
	if !t.enabled {
		return nil
	}

	query := `
		INSERT INTO events (baby_uid, timestamp, event_type)
		VALUES (?, ?, ?)
	`
	
	_, err := t.db.Exec(query, babyUID, eventTimestamp, eventType)
	if err != nil {
		log.Error().Err(err).
			Str("baby_uid", babyUID).
			Str("event_type", eventType).
			Msg("Failed to record event")
		return err
	}

	log.Debug().
		Str("baby_uid", babyUID).
		Str("event_type", eventType).
		Int64("timestamp", eventTimestamp).
		Msg("Recorded event")
		
	return nil
}

// TrackStateChange records changes in baby state (night light, standby)
func (t *Tracker) TrackStateChange(babyUID string, stateType string, value bool) error {
	if !t.enabled {
		return nil
	}

	timestamp := time.Now().Unix()
	
	query := `
		INSERT INTO state_changes (baby_uid, timestamp, state_type, state_value)
		VALUES (?, ?, ?, ?)
	`
	
	_, err := t.db.Exec(query, babyUID, timestamp, stateType, value)
	if err != nil {
		log.Error().Err(err).
			Str("baby_uid", babyUID).
			Str("state_type", stateType).
			Msg("Failed to record state change")
		return err
	}

	log.Debug().
		Str("baby_uid", babyUID).
		Str("state_type", stateType).
		Bool("value", value).
		Msg("Recorded state change")
		
	return nil
}

// GetSensorReadings retrieves sensor data for a time range
func (t *Tracker) GetSensorReadings(babyUID string, startTime, endTime int64, limit int) ([]SensorReading, error) {
	if !t.enabled {
		return nil, fmt.Errorf("historical tracking disabled")
	}

	query := `
		SELECT id, baby_uid, timestamp, temperature_celsius, humidity_percent, is_night, created_at
		FROM sensor_readings
		WHERE baby_uid = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
		LIMIT ?
	`
	
	rows, err := t.db.Query(query, babyUID, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []SensorReading
	for rows.Next() {
		var r SensorReading
		err := rows.Scan(&r.ID, &r.BabyUID, &r.Timestamp, &r.TemperatureCelsius, 
			&r.HumidityPercent, &r.IsNight, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		readings = append(readings, r)
	}

	return readings, nil
}

// GetSensorReadingsWithSampling retrieves sensor data with intelligent time-based sampling
func (t *Tracker) GetSensorReadingsWithSampling(babyUID string, startTime, endTime int64) ([]SensorReading, error) {
	if !t.enabled {
		return nil, fmt.Errorf("historical tracking disabled")
	}

	// Determine sampling strategy based on timeframe
	var query string
	var args []interface{}
	
	timeframeDuration := endTime - startTime
	timeframeHours := timeframeDuration / 3600
	
	if timeframeHours <= 6 {
		// ≤ 6 hours: Raw data (every reading)
		query = `
			SELECT id, baby_uid, timestamp, temperature_celsius, humidity_percent, is_night, created_at
			FROM sensor_readings
			WHERE baby_uid = ? AND timestamp BETWEEN ? AND ?
			ORDER BY timestamp ASC
		`
		args = []interface{}{babyUID, startTime, endTime}
		
	} else if timeframeHours <= 24 {
		// 6-24 hours: 5-minute averages
		query = `
			SELECT 
				0 as id,
				? as baby_uid,
				(timestamp / 300) * 300 as timestamp,
				AVG(temperature_celsius) as temperature_celsius,
				AVG(humidity_percent) as humidity_percent,
				CASE WHEN AVG(CASE WHEN is_night THEN 1.0 ELSE 0.0 END) > 0.5 THEN 1 ELSE 0 END as is_night,
				MIN(created_at) as created_at
			FROM sensor_readings
			WHERE baby_uid = ? AND timestamp BETWEEN ? AND ?
			GROUP BY (timestamp / 300)
			ORDER BY timestamp ASC
		`
		args = []interface{}{babyUID, babyUID, startTime, endTime}
		
	} else if timeframeHours <= 168 { // 7 days
		// 1-7 days: 1-hour averages  
		query = `
			SELECT 
				0 as id,
				? as baby_uid,
				(timestamp / 3600) * 3600 as timestamp,
				AVG(temperature_celsius) as temperature_celsius,
				AVG(humidity_percent) as humidity_percent,
				CASE WHEN AVG(CASE WHEN is_night THEN 1.0 ELSE 0.0 END) > 0.5 THEN 1 ELSE 0 END as is_night,
				MIN(created_at) as created_at
			FROM sensor_readings
			WHERE baby_uid = ? AND timestamp BETWEEN ? AND ?
			GROUP BY (timestamp / 3600)
			ORDER BY timestamp ASC
		`
		args = []interface{}{babyUID, babyUID, startTime, endTime}
		
	} else {
		// > 7 days: 6-hour averages
		query = `
			SELECT 
				0 as id,
				? as baby_uid,
				(timestamp / 21600) * 21600 as timestamp,
				AVG(temperature_celsius) as temperature_celsius,
				AVG(humidity_percent) as humidity_percent,
				CASE WHEN AVG(CASE WHEN is_night THEN 1.0 ELSE 0.0 END) > 0.5 THEN 1 ELSE 0 END as is_night,
				MIN(created_at) as created_at
			FROM sensor_readings
			WHERE baby_uid = ? AND timestamp BETWEEN ? AND ?
			GROUP BY (timestamp / 21600)
			ORDER BY timestamp ASC
		`
		args = []interface{}{babyUID, babyUID, startTime, endTime}
	}
	
	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []SensorReading
	for rows.Next() {
		var r SensorReading
		
		if timeframeHours <= 6 {
			// Raw data - is_night is boolean
			err := rows.Scan(&r.ID, &r.BabyUID, &r.Timestamp, &r.TemperatureCelsius, 
				&r.HumidityPercent, &r.IsNight, &r.CreatedAt)
			if err != nil {
				return nil, err
			}
		} else {
			// Aggregated data - is_night is integer, convert to boolean
			var isNightInt *int64
			err := rows.Scan(&r.ID, &r.BabyUID, &r.Timestamp, &r.TemperatureCelsius, 
				&r.HumidityPercent, &isNightInt, &r.CreatedAt)
			if err != nil {
				return nil, err
			}
			
			// Convert is_night integer back to boolean pointer
			if isNightInt != nil {
				isNight := *isNightInt == 1
				r.IsNight = &isNight
			}
		}
		
		readings = append(readings, r)
	}

	return readings, nil
}

// GetEvents retrieves events for a time range
func (t *Tracker) GetEvents(babyUID string, startTime, endTime int64, eventType string, limit int) ([]Event, error) {
	if !t.enabled {
		return nil, fmt.Errorf("historical tracking disabled")
	}

	var query string
	var args []interface{}

	if eventType != "" {
		query = `
			SELECT id, baby_uid, timestamp, event_type, created_at
			FROM events
			WHERE baby_uid = ? AND timestamp BETWEEN ? AND ? AND event_type = ?
			ORDER BY timestamp DESC
			LIMIT ?
		`
		args = []interface{}{babyUID, startTime, endTime, eventType, limit}
	} else {
		query = `
			SELECT id, baby_uid, timestamp, event_type, created_at
			FROM events
			WHERE baby_uid = ? AND timestamp BETWEEN ? AND ?
			ORDER BY timestamp DESC
			LIMIT ?
		`
		args = []interface{}{babyUID, startTime, endTime, limit}
	}
	
	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		err := rows.Scan(&e.ID, &e.BabyUID, &e.Timestamp, &e.EventType, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}

// GetSummary provides aggregated statistics for a time period
func (t *Tracker) GetSummary(babyUID string, startTime, endTime int64) (*HistoricalSummary, error) {
	if !t.enabled {
		return nil, fmt.Errorf("historical tracking disabled")
	}

	summary := &HistoricalSummary{
		BabyUID:   babyUID,
		StartTime: startTime,
		EndTime:   endTime,
	}

	// Get sensor statistics
	sensorQuery := `
		SELECT 
			AVG(temperature_celsius) as avg_temp,
			MIN(temperature_celsius) as min_temp,
			MAX(temperature_celsius) as max_temp,
			AVG(humidity_percent) as avg_humidity,
			MIN(humidity_percent) as min_humidity,
			MAX(humidity_percent) as max_humidity
		FROM sensor_readings 
		WHERE baby_uid = ? AND timestamp BETWEEN ? AND ?
		AND (temperature_celsius IS NOT NULL OR humidity_percent IS NOT NULL)
	`
	
	err := t.db.QueryRow(sensorQuery, babyUID, startTime, endTime).Scan(
		&summary.AvgTemperature, &summary.MinTemperature, &summary.MaxTemperature,
		&summary.AvgHumidity, &summary.MinHumidity, &summary.MaxHumidity)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get event counts
	eventQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN event_type = 'motion' THEN 1 ELSE 0 END), 0) as motion_count,
			COALESCE(SUM(CASE WHEN event_type = 'sound' THEN 1 ELSE 0 END), 0) as sound_count
		FROM events 
		WHERE baby_uid = ? AND timestamp BETWEEN ? AND ?
	`
	
	err = t.db.QueryRow(eventQuery, babyUID, startTime, endTime).Scan(
		&summary.MotionEventCount, &summary.SoundEventCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get state change counts
	stateQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN state_type = 'night_light' THEN 1 ELSE 0 END), 0) as night_light_changes,
			COALESCE(SUM(CASE WHEN state_type = 'standby' THEN 1 ELSE 0 END), 0) as standby_changes
		FROM state_changes 
		WHERE baby_uid = ? AND timestamp BETWEEN ? AND ?
	`
	
	err = t.db.QueryRow(stateQuery, babyUID, startTime, endTime).Scan(
		&summary.NightLightChanges, &summary.StandbyChanges)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Calculate day/night mode statistics
	dayNightStats := t.calculateDayNightStats(babyUID, startTime, endTime)
	summary.DayModeMinutes = dayNightStats.DayModeMinutes
	summary.NightModeMinutes = dayNightStats.NightModeMinutes
	summary.DayModePercentage = dayNightStats.DayModePercentage
	summary.NightModePercentage = dayNightStats.NightModePercentage

	return summary, nil
}

// GetDayNightAnalytics provides detailed day/night mode analysis
func (t *Tracker) GetDayNightAnalytics(babyUID string, startTime, endTime int64) (*DayNightAnalytics, error) {
	if !t.enabled {
		return nil, fmt.Errorf("historical tracking disabled")
	}

	analytics := &DayNightAnalytics{
		BabyUID:   babyUID,
		StartTime: startTime,
		EndTime:   endTime,
		TotalMinutes: (endTime - startTime) / 60,
	}

	// Get all sensor readings with is_night data ordered by timestamp
	query := `
		SELECT timestamp, is_night
		FROM sensor_readings
		WHERE baby_uid = ? AND timestamp BETWEEN ? AND ? AND is_night IS NOT NULL
		ORDER BY timestamp ASC
	`

	rows, err := t.db.Query(query, babyUID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []struct {
		timestamp int64
		isNight   bool
	}

	for rows.Next() {
		var reading struct {
			timestamp int64
			isNight   bool
		}
		err := rows.Scan(&reading.timestamp, &reading.isNight)
		if err != nil {
			return nil, err
		}
		readings = append(readings, reading)
	}

	if len(readings) == 0 {
		// No readings in this time period, check for last known state before this period
		lastKnownQuery := `
			SELECT is_night
			FROM sensor_readings
			WHERE baby_uid = ? AND timestamp < ? AND is_night IS NOT NULL
			ORDER BY timestamp DESC
			LIMIT 1
		`
		
		var lastKnownState bool
		err := t.db.QueryRow(lastKnownQuery, babyUID, startTime).Scan(&lastKnownState)
		if err != nil {
			// No previous state found, mark as unknown
			analytics.UnknownModeMinutes = analytics.TotalMinutes
			analytics.UnknownModePercentage = 100.0
			return analytics, nil
		}
		
		// Carry forward the last known state for the entire period
		if lastKnownState {
			analytics.NightModeMinutes = analytics.TotalMinutes
			analytics.NightModePercentage = 100.0
		} else {
			analytics.DayModeMinutes = analytics.TotalMinutes
			analytics.DayModePercentage = 100.0
		}
		
		return analytics, nil
	}

	// Calculate time spent in each mode and transitions
	var dayModeSeconds int64
	var nightModeSeconds int64
	var transitions int64
	var changes []DayNightChange

	// Determine initial mode - check for last known state before this period
	var currentMode bool
	
	lastKnownQuery := `
		SELECT is_night
		FROM sensor_readings
		WHERE baby_uid = ? AND timestamp < ? AND is_night IS NOT NULL
		ORDER BY timestamp DESC
		LIMIT 1
	`
	
	err = t.db.QueryRow(lastKnownQuery, babyUID, startTime).Scan(&currentMode)
	if err != nil {
		// No previous state, use the first reading's state
		currentMode = readings[0].isNight
	}
	
	currentModeStart := startTime

	// Add time from startTime to first reading
	firstReadingDuration := readings[0].timestamp - startTime
	if currentMode {
		nightModeSeconds += firstReadingDuration
	} else {
		dayModeSeconds += firstReadingDuration
	}

	// Process all readings
	for i, reading := range readings {
		// Check for mode transition
		if reading.isNight != currentMode {
			// Record the transition
			change := DayNightChange{
				Timestamp:    reading.timestamp,
				FromNight:    currentMode,
				ToNight:      reading.isNight,
				DurationMins: (reading.timestamp - currentModeStart) / 60,
			}
			changes = append(changes, change)
			
			transitions++
			currentMode = reading.isNight
			currentModeStart = reading.timestamp
		}

		// Calculate duration until next reading (or end time)
		var duration int64
		if i < len(readings)-1 {
			// Duration until next reading
			duration = readings[i+1].timestamp - reading.timestamp
		} else {
			// Duration from last reading to end time
			duration = endTime - reading.timestamp
		}

		// Attribute this duration to the current mode
		if currentMode {
			nightModeSeconds += duration
		} else {
			dayModeSeconds += duration
		}
	}

	// Convert to minutes and calculate percentages
	analytics.DayModeMinutes = dayModeSeconds / 60
	analytics.NightModeMinutes = nightModeSeconds / 60
	analytics.UnknownModeMinutes = analytics.TotalMinutes - analytics.DayModeMinutes - analytics.NightModeMinutes
	analytics.ModeTransitions = transitions
	analytics.DayNightChanges = changes

	if analytics.TotalMinutes > 0 {
		analytics.DayModePercentage = float64(analytics.DayModeMinutes) / float64(analytics.TotalMinutes) * 100
		analytics.NightModePercentage = float64(analytics.NightModeMinutes) / float64(analytics.TotalMinutes) * 100
		analytics.UnknownModePercentage = float64(analytics.UnknownModeMinutes) / float64(analytics.TotalMinutes) * 100
	}

	return analytics, nil
}

// calculateDayNightStats is a helper method for summary calculations
func (t *Tracker) calculateDayNightStats(babyUID string, startTime, endTime int64) *DayNightAnalytics {
	// Use the detailed analytics but only return the basic stats
	analytics, err := t.GetDayNightAnalytics(babyUID, startTime, endTime)
	if err != nil {
		log.Error().Err(err).Str("baby_uid", babyUID).Msg("Failed to calculate day/night stats")
		return &DayNightAnalytics{
			TotalMinutes: (endTime - startTime) / 60,
		}
	}
	return analytics
}

// Cleanup removes old data beyond the specified retention period
func (t *Tracker) Cleanup(retentionDays int) error {
	if !t.enabled {
		return nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays).Unix()
	
	tables := []string{"sensor_readings", "events", "state_changes"}
	totalDeleted := 0
	
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE created_at < ?", table)
		result, err := t.db.Exec(query, cutoffTime)
		if err != nil {
			log.Error().Err(err).Str("table", table).Msg("Failed to cleanup old data")
			continue
		}
		
		if deleted, err := result.RowsAffected(); err == nil {
			totalDeleted += int(deleted)
			log.Debug().Str("table", table).Int64("deleted", deleted).Msg("Cleaned up old records")
		}
	}
	
	if totalDeleted > 0 {
		// Vacuum database to reclaim space
		if _, err := t.db.Exec("VACUUM"); err != nil {
			log.Warn().Err(err).Msg("Failed to vacuum database after cleanup")
		}
		
		log.Info().Int("total_deleted", totalDeleted).Int("retention_days", retentionDays).
			Msg("Historical data cleanup completed")
	}
	
	return nil
}

// ResetData removes all historical data for a specific baby
func (t *Tracker) ResetData(babyUID string) (int, error) {
	if !t.enabled {
		return 0, fmt.Errorf("historical tracking disabled")
	}

	tables := []string{"sensor_readings", "events", "state_changes"}
	totalDeleted := 0
	
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE baby_uid = ?", table)
		result, err := t.db.Exec(query, babyUID)
		if err != nil {
			log.Error().Err(err).Str("table", table).Str("baby_uid", babyUID).Msg("Failed to reset data from table")
			return totalDeleted, err
		}
		
		if deleted, err := result.RowsAffected(); err == nil {
			totalDeleted += int(deleted)
			log.Debug().Str("table", table).Str("baby_uid", babyUID).Int64("deleted", deleted).Msg("Reset data from table")
		}
	}
	
	if totalDeleted > 0 {
		// Vacuum database to reclaim space
		if _, err := t.db.Exec("VACUUM"); err != nil {
			log.Warn().Err(err).Msg("Failed to vacuum database after reset")
		}
		
		log.Info().Str("baby_uid", babyUID).Int("total_deleted", totalDeleted).
			Msg("Historical data reset completed")
	}
	
	return totalDeleted, nil
}

// IsEnabled returns whether historical tracking is enabled
func (t *Tracker) IsEnabled() bool {
	return t.enabled
}