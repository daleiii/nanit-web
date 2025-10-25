# Nanit Baby Monitor Bridge

A comprehensive Go application that bridges Nanit baby monitors with Home Assistant and provides a feature-rich web dashboard for monitoring your baby.

## Features

- **🏠 Home Assistant Integration**: RTMP streaming and MQTT auto-discovery
- **🌐 Web Dashboard**: React-based real-time monitoring with interactive charts
- **📊 Historical Data**: SQLite-powered tracking of temperature, humidity, and day/night status
- **🎥 Video Streaming**: HLS transcoding for browser-based video playback
- **🔐 2FA Authentication**: Support for Nanit's required two-factor authentication
- **📱 Real-time Updates**: Live sensor data and device control
- **🔒 Optional Web Protection**: Password-protected dashboard access

This is a fork of [indiefan/home_assistant_nanit](https://github.com/indiefan/home_assistant_nanit), which itself is a fork of the original [adam.stanek/nanit](https://gitlab.com/adam.stanek/nanit) project (no longer maintained). This version includes significant enhancements including 2FA support, web dashboard, historical tracking, and modern streaming capabilities.

# Installation & Setup

## Quick Start

1. **Pull and run the container:**
```bash
docker run -d \
  --name=nanit \
  --restart unless-stopped \
  -v /path/to/data:/data \
  -p 8080:8080 \
  -p 1935:1935 \
  -e NANIT_RTMP_ADDR=YOUR_LOCAL_IP:1935 \
  deltathreed/nanit-web
```

2. **Open the web dashboard:**
   - Visit `http://localhost:8080`
   - If not yet configured, you'll be redirected to the setup page

3. **Complete setup via web interface:**
   - Enter your Nanit email and password
   - Enter the 2FA code sent to your email
   - The system will automatically start monitoring your baby
   - Dashboard will display live data, charts, and video stream

**Security Note:** The refresh token provides full access to your Nanit account. Protect your system and proceed at your own risk.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NANIT_RTMP_ADDR` | *Required* | Your local IP and port (e.g., `192.168.1.100:1935`) |
| `NANIT_HTTP_PORT` | `8080` | Web dashboard port |
| `NANIT_DATA_DIR` | `/data` | Directory where all files are stored |
| `NANIT_SESSION_FILE` | | Session file path for storing auth tokens |
| `NANIT_RTMP_AUTO_START` | `true` | Automatically start streaming when baby comes online |
| `NANIT_LOG_LEVEL` | `info` | Logging level: `trace`, `debug`, `info`, `warn`, `error` |
| `NANIT_HISTORY_ENABLED` | `true` | Enable historical data tracking |
| `NANIT_HISTORY_RETENTION_DAYS` | `30` | Days to keep historical data |
| `NANIT_MQTT_ENABLED` | `false` | Enable MQTT for Home Assistant |
| `NANIT_MQTT_BROKER_URL` | | MQTT broker URL (e.g., `tcp://localhost:1883`) |
| `NANIT_MQTT_USERNAME` | | MQTT username |
| `NANIT_MQTT_PASSWORD` | | MQTT password |
| `NANIT_MQTT_CLIENT_ID` | `nanit` | MQTT client identifier |
| `NANIT_MQTT_PREFIX` | `nanit` | MQTT topic prefix |
| `NANIT_EVENTS_POLLING` | `false` | Enable polling for event messages |
| `NANIT_EVENTS_POLLING_INTERVAL` | `30` | Seconds between event polling requests |
| `NANIT_EVENTS_MESSAGE_TIMEOUT` | `300` | Seconds after which to disregard old events |

**Note:** Nanit credentials (email/password) are configured via the web dashboard at `http://localhost:8080`, not through environment variables.

## Docker Deployment Options

### Option 1: Docker Compose (Recommended)

Use the included `docker-compose.yml`.

### Option 2: Docker Run with Full Configuration

```bash
docker run -d \
  --name=nanit \
  --restart unless-stopped \
  -v /path/to/data:/data \
  -p 8080:8080 \
  -p 1935:1935 \
  -e NANIT_RTMP_ADDR=192.168.1.100:1935 \
  -e NANIT_LOG_LEVEL=info \
  -e NANIT_HISTORY_ENABLED=true \
  -e NANIT_MQTT_ENABLED=true \
  -e NANIT_MQTT_BROKER_URL=tcp://homeassistant:1883 \
  -e NANIT_MQTT_USERNAME=mqtt_user \
  -e NANIT_MQTT_PASSWORD=mqtt_pass \
  deltathreed/nanit-web
```

**Important:** Use your local IP address (reachable by the Nanit camera), not `127.0.0.1` or `localhost`.

### Password Reset
Reset the web dashboard password protection:
```bash
docker exec -it nanit /app/nanit --reset-password
``` 

# Web Dashboard Features

The modern React-based dashboard at `http://localhost:8080` provides comprehensive baby monitoring:

## 🎥 Live Video Streaming
- **Browser-native playback**: HLS streaming works directly in any modern browser
- **Real-time video**: Low-latency streaming from your Nanit camera
- **No plugins required**: Pure HTML5 video with Video.js player

## 📊 Interactive Data Visualization
- **Real-time sensor charts**: Live temperature, humidity, and day/night status with Chart.js
- **Historical trends**: Explore data over hours, days, or weeks
- **Environmental monitoring**: Track room conditions and lighting changes over time
- **Day/night patterns**: Visualize environmental and lighting changes throughout the day

## 📱 Device Control Panel
- **Night light control**: Toggle and adjust lighting remotely
- **Standby mode**: Put camera in sleep mode when needed
- **Streaming controls**: Start/stop video streaming on demand
- **Device status**: Real-time connection and health monitoring

## 🔧 Management & Settings
- **Web-based authentication**: Complete 2FA setup without command line
- **Password protection**: Optional dashboard security
- **Configuration management**: Adjust settings through intuitive interface
- **System monitoring**: View logs, connection status, and performance metrics

## 📈 Advanced Analytics
- **Temperature alerts**: Visual indicators for threshold breaches
- **Sleep quality insights**: Track room conditions over time
- **Export capabilities**: Download historical data for analysis
- **Responsive design**: Works perfectly on desktop, tablet, and mobile

# Integrations

## Home Assistant

### MQTT Auto-Discovery (Recommended)

When MQTT is enabled, the application automatically creates Home Assistant entities:

- **Camera**: RTMP video feed
- **Sensors**: Temperature, humidity, day/night status
- **Binary Sensors**: Night mode, streaming status
- **Switches**: Night light, standby mode

Simply enable MQTT in your configuration and devices will appear automatically.

### Manual Camera Setup

Alternatively, add a camera manually to `configuration.yaml`:

```yaml
camera:
  - name: Nanit
    platform: ffmpeg
    input: rtmp://YOUR_LOCAL_IP:1935/local/YOUR_BABY_UID
```

You can find your `baby_uid` in the web dashboard at `http://localhost:8080` or in the application logs.

# Development

For local development with hot reload:

```bash
# Clone the repository
git clone https://github.com/daleiii/nanit-web.git
cd nanit-web

# Start development environment
docker-compose -f docker-compose.dev.yml up frontend backend

# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
```

See `CLAUDE.md` for detailed development instructions.

# License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

# FAQ

## Troubleshooting

### Q: Video streaming fails but sensor data still works?
**A: Connection Limit Errors** - The most common issue is **"too many local connections"** to the Nanit camera:

- **Symptoms**: Video stream fails to connect, but sensor data continues working
- **Cause**: Nanit cameras limit concurrent local streaming connections
- **Behavior**: App continuously retries connection attempts
- **Impact**: Usually doesn't affect the official Nanit mobile app

**Solutions:**
1. **Wait and retry**: Connection limits may clear after a few minutes
2. **Restart the container**: `docker restart nanit`
3. **Check network connectivity**: Ensure camera and app are on same network
4. **Verify RTMP address**: Make sure `NANIT_RTMP_ADDR` uses correct local IP

**Expected Behavior:**
- **Sensor data continues**: Temperature, humidity, day/night status still update
- **Automatic retries**: App persistently attempts to reconnect
- **Mobile app unaffected**: Official Nanit app typically continues working
- **Eventually connects**: Usually succeeds after some time

### Q: How do I check what's going wrong?
**A:** Check the application logs using the Docker commands from the Installation section:
```bash
# Real-time logs
docker logs -f nanit
# Recent logs  
docker logs --tail 100 nanit
```

### Q: Can I access this remotely/over the internet?
**A:** This should **NOT be exposed directly to the public internet**. For remote access:
- Use a VPN to access your home network
- Use a reverse proxy (nginx, Traefik, Caddy) with proper authentication and HTTPS
- Never rely solely on the built-in password protection

## Security & Safety

### Q: Is this safe to use?
**A:** This application accesses your Nanit account and camera feeds. Important considerations:

**Security Risks:**
- **Refresh tokens provide full Nanit account access**
- **Live video streams from your baby's room** 
- **Historical data about your home environment**
- **Device control capabilities**

**Recommendations:**
- Use only within your home network
- Enable the built-in password protection
- Access via local IP addresses (e.g., `http://192.168.1.100:8080`)
- **Never expose directly to the internet**

### Q: What are the legal disclaimers?
**A:** This is a personal project provided as-is for educational and personal use. **I am not responsible for any issues, damages, or consequences that may arise from using this software.**

**Use at your own risk:**
- This software accesses your Nanit account and camera feeds
- No warranty or support is provided
- You are responsible for securing your own installation
- Any misuse or security issues are your responsibility

By using this software, you acknowledge that you understand the risks and accept full responsibility for its use.

## Technical Questions

### Q: What ports does this use?
**A:** 
- **Port 8080**: Web dashboard interface
- **Port 1935**: RTMP streaming (for Home Assistant and other RTMP clients)

### Q: Where do I find my baby_uid?
**A:** Your `baby_uid` is displayed in the web dashboard at `http://localhost:8080` or can be found in the application logs.

### Q: Do I need to be on the same network as my Nanit camera?
**A:** Yes, your Docker host must be on the same network as your Nanit camera, and you must use your local IP address (not `localhost` or `127.0.0.1`) in the `NANIT_RTMP_ADDR` setting.
