// API Response Types
export interface Baby {
  uid: string;
  name: string;
  camera_uid: string;
  temperature?: number;
  humidity?: number;
  is_night?: boolean;
  night_light?: boolean;
  standby?: boolean;
  websocket_alive: boolean;
  stream_state?: string;
}

export interface StatusResponse {
  timestamp: number;
  babies: Baby[];
}

export interface DeviceInfo {
  firmware_version?: string;
  hardware_version?: string;
  device_mode?: string;
  volume?: number;
  night_vision?: boolean;
  sleep_mode?: boolean;
  mic_mute?: boolean;
  wifi_network?: string;
  wifi_band?: string;
  anti_flicker?: string;
  temp_low_threshold?: number;
  temp_high_threshold?: number;
  humidity_low_threshold?: number;
  humidity_high_threshold?: number;
  mobile_bitrate?: number;
  mobile_fps?: number;
  dvr_bitrate?: number;
  dvr_fps?: number;
  analytics_bitrate?: number;
  analytics_fps?: number;
  streaming_error?: string;
  last_updated?: number;
}

export interface DeviceAlert {
  type: 'error' | 'warning';
  message: string;
  category: string;
}

export interface DeviceInfoResponse {
  baby_uid: string;
  baby_name: string;
  camera_uid: string;
  timestamp: number;
  device_info: DeviceInfo;
  connection_status: {
    websocket_alive: boolean;
    stream_state: string;
  };
  alerts: DeviceAlert[];
}

export interface SensorReading {
  timestamp: number;
  temperature_celsius?: number;
  humidity_percent?: number;
  is_night?: boolean;
}

export interface SensorDataResponse {
  baby_uid: string;
  start_time: number;
  end_time: number;
  readings: SensorReading[];
  count: number;
}

// Motion/Sound activity interfaces - disabled for now
// export interface ActivityEvent {
//   timestamp: number;
//   event_type: 'motion' | 'sound';
//   value?: number;
// }

// export interface EventsDataResponse {
//   baby_uid: string;
//   start_time: number;
//   end_time: number;
//   event_type?: string;
//   events: ActivityEvent[];
//   count: number;
// }

export interface HistorySummary {
  baby_uid: string;
  start_time: number;
  end_time: number;
  avg_temperature?: number;
  min_temperature?: number;
  max_temperature?: number;
  avg_humidity?: number;
  min_humidity?: number;
  max_humidity?: number;
  // motion_event_count: number;  // Disabled motion/sound activity
  // sound_event_count: number;   // Disabled motion/sound activity
  day_mode_percentage: number;
  night_mode_percentage: number;
}

export interface DayNightChange {
  timestamp: number;
  from_night: boolean;
  to_night: boolean;
}

export interface DayNightAnalytics {
  baby_uid: string;
  start_time: number;
  end_time: number;
  day_mode_percentage: number;
  night_mode_percentage: number;
  mode_transitions: number;
  day_night_changes: DayNightChange[];
}

// Control Request Types
export interface ControlRequest {
  baby_uid: string;
  action: 'toggle';
}

export interface ControlResponse {
  success: boolean;
  baby_uid: string;
  control: string;
  action: string;
  timestamp: number;
}

// Authentication Types
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  success: boolean;
  mfa_token?: any;
  message: string;
  error?: string;
}

export interface Verify2FARequest {
  email: string;
  password: string;
  mfa_token: any;
  mfa_code: string;
}

export interface Verify2FAResponse {
  success: boolean;
  message: string;
  error?: string;
}

export interface AuthStatusResponse {
  authenticated: boolean;
  message: string;
  email?: string;
  babies_count?: number;
  services_running?: boolean;
  auth_time?: number;
}

export interface AuthResetResponse {
  success: boolean;
  message: string;
}

// Stream Types
export interface StreamStartRequest {
  baby_uid: string;
}

export interface StreamStartResponse {
  success: boolean;
  baby_uid: string;
  hls_url: string;
  message: string;
}

export interface StreamError {
  type: string;
  message: string;
}

export interface StreamStatusResponse {
  baby_uid: string;
  status: string;
  message: string;
  stream_error?: StreamError;
}

// Web Authentication Types
export interface WebAuthStatusResponse {
  password_protection_enabled: boolean;
  password_set: boolean;
  authenticated: boolean;
}

export interface WebAuthResponse {
  success: boolean;
  message: string;
  error?: string;
}

// Health Types
export interface HealthComponentStatus {
  status: string;
  [key: string]: any;
}

export interface HealthDetails {
  websocket: HealthComponentStatus;
  rtmp: HealthComponentStatus;
  hls: HealthComponentStatus;
}

export interface HealthResponse {
  baby_uid: string;
  overall_health: 'healthy' | 'degraded' | 'unhealthy' | 'starting';
  details: HealthDetails;
  timestamp: number;
}