package baby

import (
	reflect "reflect"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type StreamRequestState int32

const (
	StreamRequestState_NotRequested StreamRequestState = iota
	StreamRequestState_Requested
	StreamRequestState_RequestFailed
)

type StreamState int32

const (
	StreamState_Unknown StreamState = iota
	StreamState_Unhealthy
	StreamState_Alive
)

// DeviceInfo - struct holding device information from Nanit API responses
type DeviceInfo struct {
	FirmwareVersion     *string `json:"firmware_version,omitempty"`
	HardwareVersion     *string `json:"hardware_version,omitempty"`
	DeviceMode          *string `json:"device_mode,omitempty"`
	MountingMode        *int32  `json:"mounting_mode,omitempty"`
	WiFiNetwork         *string `json:"wifi_network,omitempty"`
	WiFiBand            *string `json:"wifi_band,omitempty"`
	NightVision         *bool   `json:"night_vision,omitempty"`
	Volume              *int32  `json:"volume,omitempty"`
	SleepMode           *bool   `json:"sleep_mode,omitempty"`
	StatusLight         *bool   `json:"status_light,omitempty"`
	MicMute             *bool   `json:"mic_mute,omitempty"`
	AntiFlicker         *string `json:"anti_flicker,omitempty"`
	StreamingError      *string `json:"streaming_error,omitempty"`
	UpgradeDownloaded   *bool   `json:"upgrade_downloaded,omitempty"`
	AvailableSoundtracks []string `json:"available_soundtracks,omitempty"`
	
	// Sensor thresholds
	TempLowThreshold  *int32 `json:"temp_low_threshold,omitempty"`
	TempHighThreshold *int32 `json:"temp_high_threshold,omitempty"`
	HumidityLowThreshold  *int32 `json:"humidity_low_threshold,omitempty"`
	HumidityHighThreshold *int32 `json:"humidity_high_threshold,omitempty"`
	
	// Stream configuration
	MobileBitrate    *int32 `json:"mobile_bitrate,omitempty"`
	MobileFPS        *int32 `json:"mobile_fps,omitempty"`
	DVRBitrate       *int32 `json:"dvr_bitrate,omitempty"`
	DVRFPS           *int32 `json:"dvr_fps,omitempty"`
	AnalyticsBitrate *int32 `json:"analytics_bitrate,omitempty"`
	AnalyticsFPS     *int32 `json:"analytics_fps,omitempty"`
	
	// Metadata
	LastUpdated      *int64  `json:"last_updated,omitempty"`  // Unix timestamp when device info was last updated
}

// State - struct holding information about state of a single baby
type State struct {
	StreamState        *StreamState        `internal:"true"`
	StreamRequestState *StreamRequestState `internal:"true"`
	IsWebsocketAlive   *bool               `internal:"true"`
	LastVideoPacketTime *int64             `internal:"true"` // Unix timestamp of last video packet received

	MotionTimestamp  *int32 // int32 is used to represent UTC timestamp
	SoundTimestamp   *int32 // int32 is used to represent UTC timestamp
	Temperature      *bool
	IsNight          *bool
	TemperatureMilli *int32
	HumidityMilli    *int32
	NightLight       *bool
	Standby          *bool
	
	// Device information cache
	DeviceInfo *DeviceInfo `internal:"true"`
}

// NewState - constructor
func NewState() *State {
	return &State{}
}

// Merge - Merges non-nil values of an argument to the state.
// Returns ptr to new state if changes
// Returns ptr to old state if not changed
func (state *State) Merge(stateUpdate *State) *State {
	newState := &State{}
	changed := false

	currReflect := reflect.ValueOf(state).Elem()
	newReflect := reflect.ValueOf(newState).Elem()
	patchReflect := reflect.ValueOf(stateUpdate).Elem()

	for i := 0; i < currReflect.NumField(); i++ {
		currField := currReflect.Field(i)
		newField := newReflect.Field(i)
		patchField := patchReflect.Field(i)
		fieldName := currReflect.Type().Field(i).Name

		if currField.Type().Kind() == reflect.Ptr {
			// Special handling for DeviceInfo struct
			if fieldName == "DeviceInfo" && !patchField.IsNil() {
				// Merge DeviceInfo fields
				var mergedDeviceInfo *DeviceInfo
				if currField.IsNil() {
					// Current is nil, use patch
					mergedDeviceInfo = patchField.Interface().(*DeviceInfo)
					changed = true
				} else {
					// Merge non-nil fields from patch into current
					mergedDeviceInfo = state.mergeDeviceInfo(currField.Interface().(*DeviceInfo), patchField.Interface().(*DeviceInfo))
					if mergedDeviceInfo != currField.Interface().(*DeviceInfo) {
						changed = true
					}
				}
				newField.Set(reflect.ValueOf(mergedDeviceInfo))
			} else if !patchField.IsNil() && (currField.IsNil() || currField.Elem().Interface() != patchField.Elem().Interface()) {
				changed = true
				ptr := reflect.New(patchField.Type().Elem())
				ptr.Elem().Set(patchField.Elem())
				newField.Set(ptr)
			} else {
				newField.Set(currField)
			}
		}
	}

	if changed {
		return newState
	}

	return state
}

// mergeDeviceInfo merges non-nil fields from patch into current DeviceInfo
func (state *State) mergeDeviceInfo(current *DeviceInfo, patch *DeviceInfo) *DeviceInfo {
	if patch == nil {
		return current
	}
	if current == nil {
		return patch
	}
	
	// Create a copy of current
	merged := *current
	
	// Merge non-nil fields from patch
	if patch.FirmwareVersion != nil {
		merged.FirmwareVersion = patch.FirmwareVersion
	}
	if patch.HardwareVersion != nil {
		merged.HardwareVersion = patch.HardwareVersion
	}
	if patch.DeviceMode != nil {
		merged.DeviceMode = patch.DeviceMode
	}
	if patch.MountingMode != nil {
		merged.MountingMode = patch.MountingMode
	}
	if patch.WiFiNetwork != nil {
		merged.WiFiNetwork = patch.WiFiNetwork
	}
	if patch.WiFiBand != nil {
		merged.WiFiBand = patch.WiFiBand
	}
	if patch.NightVision != nil {
		merged.NightVision = patch.NightVision
	}
	if patch.Volume != nil {
		merged.Volume = patch.Volume
	}
	if patch.SleepMode != nil {
		merged.SleepMode = patch.SleepMode
	}
	if patch.StatusLight != nil {
		merged.StatusLight = patch.StatusLight
	}
	if patch.MicMute != nil {
		merged.MicMute = patch.MicMute
	}
	if patch.AntiFlicker != nil {
		merged.AntiFlicker = patch.AntiFlicker
	}
	if patch.StreamingError != nil {
		merged.StreamingError = patch.StreamingError
	}
	if patch.UpgradeDownloaded != nil {
		merged.UpgradeDownloaded = patch.UpgradeDownloaded
	}
	if patch.AvailableSoundtracks != nil {
		merged.AvailableSoundtracks = patch.AvailableSoundtracks
	}
	if patch.TempLowThreshold != nil {
		merged.TempLowThreshold = patch.TempLowThreshold
	}
	if patch.TempHighThreshold != nil {
		merged.TempHighThreshold = patch.TempHighThreshold
	}
	if patch.HumidityLowThreshold != nil {
		merged.HumidityLowThreshold = patch.HumidityLowThreshold
	}
	if patch.HumidityHighThreshold != nil {
		merged.HumidityHighThreshold = patch.HumidityHighThreshold
	}
	if patch.MobileBitrate != nil {
		merged.MobileBitrate = patch.MobileBitrate
	}
	if patch.MobileFPS != nil {
		merged.MobileFPS = patch.MobileFPS
	}
	if patch.DVRBitrate != nil {
		merged.DVRBitrate = patch.DVRBitrate
	}
	if patch.DVRFPS != nil {
		merged.DVRFPS = patch.DVRFPS
	}
	if patch.AnalyticsBitrate != nil {
		merged.AnalyticsBitrate = patch.AnalyticsBitrate
	}
	if patch.AnalyticsFPS != nil {
		merged.AnalyticsFPS = patch.AnalyticsFPS
	}
	
	return &merged
}

var upperCaseRX = regexp.MustCompile("[A-Z]+")

// AsMap - returns K/V map of non-nil properties
func (state *State) AsMap(includeInternal bool) map[string]interface{} {
	m := make(map[string]interface{})

	r := reflect.ValueOf(state).Elem()
	ts := reflect.TypeOf(*state)
	t := r.Type()
	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)

		if includeInternal || ts.Field(i).Tag.Get("internal") != "true" {

			if !f.IsNil() && f.Type().Kind() == reflect.Ptr {
				name := t.Field(i).Name
				var value interface{}

				if f.Type().Elem().Kind() == reflect.Int32 {
					value = f.Elem().Int()

					if strings.HasSuffix(name, "Milli") {
						name = strings.TrimSuffix(name, "Milli")
						value = float64(value.(int64)) / 1000
					}
				} else {
					value = f.Elem().Interface()
				}

				name = strings.ToLower(name[0:1]) + name[1:]
				name = upperCaseRX.ReplaceAllStringFunc(name, func(m string) string {
					return "_" + strings.ToLower(m)
				})

				m[name] = value
			}
		}
	}

	return m
}

// EnhanceLogEvent - appends non-nil properties to a log event
func (state *State) EnhanceLogEvent(e *zerolog.Event) *zerolog.Event {
	for key, value := range state.AsMap(true) {
		e.Interface(key, value)
	}

	return e
}

// SetTemperatureMilli - mutates field, returns itself
func (state *State) SetTemperatureMilli(value int32) *State {
	state.TemperatureMilli = &value
	return state
}

// GetTemperature - returns temperature as floating point
func (state *State) GetTemperature() float64 {
	if state.TemperatureMilli != nil {
		return float64(*state.TemperatureMilli) / 1000
	}

	return 0
}

// SetHumidityMilli - mutates field, returns itself
func (state *State) SetHumidityMilli(value int32) *State {
	state.HumidityMilli = &value
	return state
}

// GetHumidity - returns humidity as floating point
func (state *State) GetHumidity() float64 {
	if state.HumidityMilli != nil {
		return float64(*state.HumidityMilli) / 1000
	}

	return 0
}

// SetStreamRequestState - mutates field, returns itself
func (state *State) SetStreamRequestState(value StreamRequestState) *State {
	state.StreamRequestState = &value
	return state
}

// GetStreamRequestState - safely returns value
func (state *State) GetStreamRequestState() StreamRequestState {
	if state.StreamRequestState != nil {
		return *state.StreamRequestState
	}

	return StreamRequestState_NotRequested
}

// SetStreamState - mutates field, returns itself
func (state *State) SetStreamState(value StreamState) *State {
	state.StreamState = &value
	return state
}

// GetStreamState - safely returns value
func (state *State) GetStreamState() StreamState {
	if state.StreamState != nil {
		return *state.StreamState
	}

	return StreamState_Unknown
}

// SetLastVideoPacketTime - mutates field, returns itself
func (state *State) SetLastVideoPacketTime(value int64) *State {
	state.LastVideoPacketTime = &value
	return state
}

func (state *State) GetLastVideoPacketTime() *int64 {
	return state.LastVideoPacketTime
}

// IsActivelyStreaming checks if video packets were received recently (within 10 seconds)
func (state *State) IsActivelyStreaming() bool {
	if state.LastVideoPacketTime == nil {
		return false
	}
	
	lastPacketTime := time.Unix(*state.LastVideoPacketTime, 0)
	return time.Since(lastPacketTime) < 10*time.Second
}

// SetIsNight - mutates field, returns itself
func (state *State) SetIsNight(value bool) *State {
	state.IsNight = &value
	return state
}

func (state *State) SetMotionTimestamp(value int32) *State {
	state.MotionTimestamp = &value
	return state
}

func (state *State) SetSoundTimestamp(value int32) *State {
	state.SoundTimestamp = &value
	return state
}

func (state *State) SetTemperature(value bool) *State {
	state.Temperature = &value
	return state
}

// GetIsWebsocketAlive - safely returns value
func (state *State) GetIsWebsocketAlive() bool {
	if state.IsWebsocketAlive != nil {
		return *state.IsWebsocketAlive
	}

	return false
}

// SetWebsocketAlive - mutates field, returns itself
func (state *State) SetWebsocketAlive(value bool) *State {
	state.IsWebsocketAlive = &value
	return state
}

func (s *State) SetNightLight(enabled bool) *State {
	s.NightLight = &enabled
	return s
}

func (s *State) GetNightLight() bool {
	return s.NightLight != nil && *s.NightLight
}

func (s *State) SetStandby(enabled bool) *State {
	s.Standby = &enabled
	return s
}

func (s *State) GetStandby() bool {
	return s.Standby != nil && *s.Standby
}

// SetDeviceInfo - mutates device info field, returns itself
func (s *State) SetDeviceInfo(info *DeviceInfo) *State {
	s.DeviceInfo = info
	return s
}

// GetDeviceInfo - safely returns device info
func (s *State) GetDeviceInfo() *DeviceInfo {
	if s.DeviceInfo == nil {
		s.DeviceInfo = &DeviceInfo{}
	}
	return s.DeviceInfo
}

// UpdateDeviceInfoField - helper to update specific device info fields
func (s *State) UpdateDeviceInfoField(field string, value interface{}) *State {
	if s.DeviceInfo == nil {
		s.DeviceInfo = &DeviceInfo{}
	}
	
	switch field {
	case "firmware_version":
		if v, ok := value.(string); ok {
			s.DeviceInfo.FirmwareVersion = &v
		}
	case "hardware_version":
		if v, ok := value.(string); ok {
			s.DeviceInfo.HardwareVersion = &v
		}
	case "device_mode":
		if v, ok := value.(string); ok {
			s.DeviceInfo.DeviceMode = &v
		}
	case "wifi_network":
		if v, ok := value.(string); ok {
			s.DeviceInfo.WiFiNetwork = &v
		}
	case "night_vision":
		if v, ok := value.(bool); ok {
			s.DeviceInfo.NightVision = &v
		}
	case "volume":
		if v, ok := value.(int32); ok {
			s.DeviceInfo.Volume = &v
		}
	case "sleep_mode":
		if v, ok := value.(bool); ok {
			s.DeviceInfo.SleepMode = &v
		}
	case "streaming_error":
		if v, ok := value.(string); ok {
			s.DeviceInfo.StreamingError = &v
		}
	}
	
	return s
}
