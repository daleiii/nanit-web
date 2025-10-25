import type {
  StatusResponse,
  DeviceInfoResponse,
  SensorDataResponse,
  // EventsDataResponse,  // Disabled motion/sound activity
  HistorySummary,
  DayNightAnalytics,
  ControlRequest,
  ControlResponse,
  LoginRequest,
  LoginResponse,
  Verify2FARequest,
  Verify2FAResponse,
  AuthStatusResponse,
  AuthResetResponse,
  StreamStartRequest,
  StreamStartResponse,
  StreamStatusResponse,
  WebAuthStatusResponse,
  WebAuthResponse,
  HealthResponse,
} from '@/types/api'

// In production, API calls go directly to the same host since Go serves the frontend
const API_BASE = '';

class ApiClient {
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${API_BASE}/api${endpoint}`;
    
    const response = await fetch(url, {
      credentials: 'include', // Include cookies for authentication
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      throw new Error(`API Error: ${response.status} ${response.statusText}`);
    }

    return response.json();
  }

  // Status and Baby Data
  async getStatus(): Promise<StatusResponse> {
    return this.request<StatusResponse>('/status');
  }

  async getDeviceInfo(babyUid: string): Promise<DeviceInfoResponse> {
    return this.request<DeviceInfoResponse>(`/device-info/${babyUid}`);
  }

  // Historical Data
  async getSensorData(
    babyUid: string,
    startTime: number,
    endTime: number,
    limit = 500
  ): Promise<SensorDataResponse> {
    const params = new URLSearchParams({
      start: startTime.toString(),
      end: endTime.toString(),
      limit: limit.toString(),
    });
    return this.request<SensorDataResponse>(`/history/sensor/${babyUid}?${params}`);
  }

  // Motion/Sound Events API - DISABLED
  // async getEventsData(
  //   babyUid: string,
  //   startTime: number,
  //   endTime: number,
  //   eventType?: string,
  //   limit = 200
  // ): Promise<EventsDataResponse> {
  //   const params = new URLSearchParams({
  //     start: startTime.toString(),
  //     end: endTime.toString(),
  //     limit: limit.toString(),
  //   });
  //   
  //   if (eventType) {
  //     params.append('type', eventType);
  //   }
  //   
  //   return this.request<EventsDataResponse>(`/history/events/${babyUid}?${params}`);
  // }

  async getHistorySummary(
    babyUid: string,
    startTime: number,
    endTime: number
  ): Promise<HistorySummary> {
    const params = new URLSearchParams({
      start: startTime.toString(),
      end: endTime.toString(),
    });
    return this.request<HistorySummary>(`/history/summary/${babyUid}?${params}`);
  }

  async getDayNightAnalytics(
    babyUid: string,
    startTime: number,
    endTime: number
  ): Promise<DayNightAnalytics> {
    const params = new URLSearchParams({
      start: startTime.toString(),
      end: endTime.toString(),
    });
    const response = await this.request<{day_night: DayNightAnalytics}>(`/history/day-night/${babyUid}?${params}`);
    return response.day_night;
  }

  async resetHistoricalData(babyUid: string): Promise<{ success: boolean; deleted_count: number }> {
    return this.request(`/history/reset/${babyUid}`, {
      method: 'DELETE',
    });
  }

  // Control Commands
  async toggleNightLight(babyUid: string): Promise<ControlResponse> {
    const payload: ControlRequest = {
      baby_uid: babyUid,
      action: 'toggle',
    };
    return this.request<ControlResponse>('/control/night-light', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async toggleStandby(babyUid: string): Promise<ControlResponse> {
    const payload: ControlRequest = {
      baby_uid: babyUid,
      action: 'toggle',
    };
    return this.request<ControlResponse>('/control/standby', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  // Authentication
  async login(email: string, password: string): Promise<LoginResponse> {
    const payload: LoginRequest = { email, password };
    return this.request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async verify2FA(
    email: string,
    password: string,
    mfaToken: any,
    mfaCode: string
  ): Promise<Verify2FAResponse> {
    const payload: Verify2FARequest = {
      email,
      password,
      mfa_token: mfaToken,
      mfa_code: mfaCode,
    };
    return this.request<Verify2FAResponse>('/auth/verify-2fa', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async getAuthStatus(): Promise<AuthStatusResponse> {
    return this.request<AuthStatusResponse>('/auth/status');
  }

  async resetAuth(): Promise<AuthResetResponse> {
    return this.request<AuthResetResponse>('/auth/reset', {
      method: 'DELETE',
    });
  }

  // Streaming
  async startStream(babyUid: string): Promise<StreamStartResponse> {
    const payload: StreamStartRequest = { baby_uid: babyUid };
    return this.request<StreamStartResponse>('/stream/start/', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async stopStream(babyUid: string): Promise<{ success: boolean; message: string }> {
    const payload = { baby_uid: babyUid };
    return this.request('/stream/stop/', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async getStreamStatus(babyUid: string): Promise<StreamStatusResponse> {
    return this.request<StreamStatusResponse>(`/stream/status/${babyUid}`);
  }

  // Health
  async getHealth(babyUid: string): Promise<HealthResponse> {
    return this.request<HealthResponse>(`/health/${babyUid}`);
  }

  // Utility methods
  getRTMPUrl(babyUid: string): string {
    const host = typeof window !== 'undefined' ? window.location.hostname : 'localhost';
    return `rtmp://${host}:1940/local/${babyUid}`;
  }

  getHLSUrl(babyUid: string): string {
    const host = typeof window !== 'undefined' ? window.location.hostname : 'localhost';
    const port = typeof window !== 'undefined' ? window.location.port : '8080';
    return `http://${host}:${port}/api/stream/hls/${babyUid}/playlist.m3u8`;
  }

  // Web Authentication
  async getWebAuthStatus(): Promise<WebAuthStatusResponse> {
    return this.request<WebAuthStatusResponse>('/webauth/status');
  }

  async loginWeb(password: string): Promise<WebAuthResponse> {
    return this.request<WebAuthResponse>('/webauth/login', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
  }

  async logoutWeb(): Promise<WebAuthResponse> {
    return this.request<WebAuthResponse>('/webauth/logout', {
      method: 'POST',
    });
  }

  async setWebPassword(password: string): Promise<WebAuthResponse> {
    return this.request<WebAuthResponse>('/webauth/set-password', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
  }

  async changeWebPassword(currentPassword: string, newPassword: string): Promise<WebAuthResponse> {
    return this.request<WebAuthResponse>('/webauth/change-password', {
      method: 'POST',
      body: JSON.stringify({ 
        current_password: currentPassword,
        new_password: newPassword 
      }),
    });
  }

  async removeWebPassword(password: string): Promise<WebAuthResponse> {
    return this.request<WebAuthResponse>('/webauth/remove-password', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
  }
}

export const api = new ApiClient();
export default api;