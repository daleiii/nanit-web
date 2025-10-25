-- Historical data tracking schema for Nanit baby monitor data

-- Table for storing sensor readings (temperature, humidity)
CREATE TABLE IF NOT EXISTS sensor_readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    baby_uid TEXT NOT NULL,
    timestamp INTEGER NOT NULL, -- Unix timestamp
    temperature_celsius REAL,    -- Temperature in Celsius
    humidity_percent REAL,       -- Humidity percentage
    is_night BOOLEAN,           -- Day/night mode
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Table for storing event data (motion, sound)
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    baby_uid TEXT NOT NULL,
    timestamp INTEGER NOT NULL, -- Unix timestamp from camera
    event_type TEXT NOT NULL,   -- 'motion' or 'sound'
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Table for storing state changes (night light, standby)
CREATE TABLE IF NOT EXISTS state_changes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    baby_uid TEXT NOT NULL,
    timestamp INTEGER NOT NULL, -- Unix timestamp
    state_type TEXT NOT NULL,   -- 'night_light' or 'standby'
    state_value BOOLEAN NOT NULL, -- true/false
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_sensor_readings_baby_time ON sensor_readings(baby_uid, timestamp);
CREATE INDEX IF NOT EXISTS idx_events_baby_time ON events(baby_uid, timestamp);
CREATE INDEX IF NOT EXISTS idx_state_changes_baby_time ON state_changes(baby_uid, timestamp);

-- Indexes for cleanup operations (based on created_at)
CREATE INDEX IF NOT EXISTS idx_sensor_readings_created ON sensor_readings(created_at);
CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at);
CREATE INDEX IF NOT EXISTS idx_state_changes_created ON state_changes(created_at);