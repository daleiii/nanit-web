package app

import (
	"github.com/indiefan/home_assistant_nanit/pkg/mqtt"
	"time"
)

// Opts - application run options
type Opts struct {
	NanitCredentials NanitCredentials
	SessionFile      string
	DataDirectories  DataDirectories
	HTTPEnabled      bool
	HTTPPort         int
	MQTT             *mqtt.Opts
	RTMP             *RTMPOpts
	EventPolling     EventPollingOpts
	History          HistoryOpts
	WebAuth          WebAuthOpts
}

// NanitCredentials - user credentials for Nanit account
type NanitCredentials struct {
	Email        string
	Password     string
	RefreshToken string
}

// DataDirectories - dictionary of dir paths
type DataDirectories struct {
	BaseDir    string
	VideoDir   string
	LogDir     string
	HistoryDir string
}

// RTMPOpts - options for RTMP streaming
type RTMPOpts struct {
	// IP:Port of the interface on which we should listen
	ListenAddr string

	// IP:Port under which can Cam reach the RTMP server
	PublicAddr string

	// Automatically start streaming when baby comes online
	AutoStart bool
}

type EventPollingOpts struct {
	Enabled         bool
	PollingInterval time.Duration
	MessageTimeout  time.Duration
}

// HistoryOpts - options for historical data tracking
type HistoryOpts struct {
	Enabled        bool
	RetentionDays  int
	CleanupEnabled bool
}

// WebAuthOpts - options for web interface authentication
type WebAuthOpts struct {
	Enabled      bool
	PasswordFile string
}
