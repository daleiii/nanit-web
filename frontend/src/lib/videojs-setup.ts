import videojs from 'video.js'
import 'video.js/dist/video-js.css'

// No external HLS plugin needed - Video.js 8.x has built-in HLS support

// Define Video.js player type
type VideoJSPlayer = ReturnType<typeof videojs>

// Video.js configuration for live streaming
export const createVideoJSOptions = (hlsUrl: string) => ({
  // Basic configuration
  fluid: true,
  responsive: true,
  fill: true,
  playsinline: true,
  controls: true,
  preload: 'none', // Don't preload until we have a stream
  width: '100%',
  height: 'auto',
  
  // Store HLS URL for components to access
  hlsUrl,
  
  // Live streaming specific options
  liveui: true,
  liveTracker: {
    trackingThreshold: 20, // Consider "live" if within 20 seconds
    liveTolerance: 15,     // Show live UI if within 15 seconds of live edge
  },
  
  // HLS source configuration - will be set when streaming starts
  sources: [],
  
  // HTML5 options for better live streaming
  html5: {
    hls: {
      enableLowInitialPlaylist: true,
      smoothQualityChange: true,
      overrideNative: false, // Let Safari use native HLS support
    },
  },
  
  // UI customization
  userActions: {
    hotkeys: true, // Enable keyboard shortcuts
  },
  
  // Error handling
  errorDisplay: false, // We'll handle errors with custom UI
  
  // Performance optimizations
  techOrder: ['html5'], // Prefer HTML5 tech
  
  // Mobile optimizations
  breakpoints: {
    tiny: 210,
    xsmall: 320,
    small: 425,
    medium: 768,
    large: 1024,
    xlarge: 1440,
    huge: 1920,
  },
})

// Custom Video.js theme for Nanit dashboard
export const injectCustomStyles = () => {
  const style = document.createElement('style')
  style.textContent = `
    /* Nanit Video.js Theme */
    .video-js {
      font-family: inherit;
      border-radius: 0.5rem;
      overflow: hidden;
    }
    
    /* Live badge styling */
    .vjs-live-control {
      background: #dc2626 !important;
      color: white !important;
      border-radius: 0.375rem;
      font-weight: 600;
      font-size: 0.875rem;
      padding: 0.25rem 0.75rem;
      margin-right: 0.5rem;
    }
    
    .vjs-live-control.vjs-control.vjs-button > .vjs-live-display {
      font-size: inherit;
    }
    
    /* Live indicator when at live edge */
    .vjs-live-control.vjs-at-live-edge {
      animation: pulse 2s infinite;
    }
    
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.7; }
    }
    
    /* Control bar styling */
    .video-js .vjs-control-bar {
      background: linear-gradient(180deg, transparent 0%, rgba(0,0,0,0.8) 100%);
      backdrop-filter: blur(4px);
    }
    
    /* Progress bar styling */
    .video-js .vjs-progress-control .vjs-progress-holder {
      height: 0.375rem;
      border-radius: 0.1875rem;
    }
    
    .video-js .vjs-progress-control .vjs-play-progress {
      background: #3b82f6;
      border-radius: 0.1875rem;
    }
    
    /* Live progress bar - red for live content */
    .vjs-live .vjs-progress-control .vjs-play-progress {
      background: #dc2626;
    }
    
    /* Loading spinner */
    .vjs-loading-spinner {
      border-color: #3b82f6 transparent #3b82f6 transparent;
    }
    
    /* Big play button */
    .video-js .vjs-big-play-button {
      background: rgba(59, 130, 246, 0.9);
      border: none;
      border-radius: 50%;
      width: 4rem;
      height: 4rem;
      line-height: 4rem;
      margin-top: -2rem;
      margin-left: -2rem;
      font-size: 1.5rem;
    }
    
    .video-js .vjs-big-play-button:hover {
      background: rgba(59, 130, 246, 1);
    }
    
    /* Volume control */
    .video-js .vjs-volume-panel .vjs-volume-control {
      background: rgba(0, 0, 0, 0.8);
      border-radius: 0.375rem;
    }
    
    /* Fullscreen button */
    .video-js .vjs-fullscreen-control {
      order: 10; /* Move to the end */
    }
    
    /* Error styling */
    .vjs-error .vjs-error-display {
      background: rgba(220, 38, 38, 0.9);
      backdrop-filter: blur(4px);
    }
    
    /* Custom control buttons */
    .vjs-check-stream-control,
    .vjs-load-stream-control,
    .vjs-snapshot-control {
      font-size: 1.2em;
      padding: 0 0.5em;
      cursor: pointer;
      border: none;
      background: transparent;
      color: white;
      opacity: 0.8;
      transition: opacity 0.3s ease;
    }
    
    .vjs-check-stream-control:hover,
    .vjs-load-stream-control:hover,
    .vjs-snapshot-control:hover {
      opacity: 1;
    }
    
    /* Stream status display */
    .vjs-stream-status-display {
      position: absolute;
      top: 10px;
      right: 10px;
      padding: 4px 8px;
      border-radius: 4px;
      font-size: 12px;
      font-weight: 600;
      pointer-events: none;
      z-index: 3;
    }
    
    .vjs-status-info {
      background: rgba(0, 0, 0, 0.7);
      color: white;
    }
    
    .vjs-status-success {
      background: rgba(34, 197, 94, 0.9);
      color: white;
    }
    
    .vjs-status-error {
      background: rgba(239, 68, 68, 0.9);
      color: white;
    }
    
    .vjs-status-loading {
      background: rgba(59, 130, 246, 0.9);
      color: white;
    }
    
    .vjs-status-live {
      background: rgba(220, 38, 38, 0.9);
      color: white;
      animation: pulse 2s infinite;
    }

    /* Responsive adjustments */
    @media (max-width: 768px) {
      .video-js .vjs-control-bar {
        font-size: 1rem;
      }
      
      .video-js .vjs-big-play-button {
        width: 3rem;
        height: 3rem;
        line-height: 3rem;
        margin-top: -1.5rem;
        margin-left: -1.5rem;
        font-size: 1.25rem;
      }
      
      .vjs-stream-status-display {
        top: 5px;
        right: 5px;
        font-size: 10px;
      }
    }
  `
  
  document.head.appendChild(style)
}

// Note: Removed old complex component registration system in favor of simple DOM approach

// Initialize Video.js with custom settings
export const initializeVideoJS = () => {
  // Inject custom styles
  injectCustomStyles()
  
  // Set global Video.js options
  videojs.options.techOrder = ['html5']
  videojs.options.html5 = {
    ...videojs.options.html5,
    hls: {
      enableLowInitialPlaylist: true,
      smoothQualityChange: true,
      overrideNative: false, // Use native HLS when available
    },
  }
  
  return videojs
}

// Add custom controls to Video.js player (matching working test HTML approach)
export const addCustomControlsToPlayer = (player: VideoJSPlayer, hlsUrl: string) => {
  // Create status display overlay
  const statusDisplay = document.createElement('div')
  statusDisplay.className = 'vjs-stream-status-display vjs-status-info'
  statusDisplay.textContent = 'Ready'
  player.el().appendChild(statusDisplay)
  
  // Function to update status
  function updateStatus(text: string, type: string) {
    statusDisplay.textContent = text
    statusDisplay.className = `vjs-stream-status-display vjs-status-${type}`
  }
  
  // Add event listeners for status updates
  player.on('streamAvailable', () => updateStatus('Stream Available', 'success'))
  player.on('streamUnavailable', (event: any, error: string) => updateStatus(`Stream Error: ${error}`, 'error'))
  player.on('streamLoaded', () => updateStatus('Stream Loaded', 'success'))
  player.on('loadstart', () => updateStatus('Loading...', 'loading'))
  player.on('canplay', () => updateStatus('Ready to Play', 'success'))
  player.on('play', () => updateStatus('Playing', 'success'))
  player.on('pause', () => updateStatus('Paused', 'info'))
  player.on('error', () => updateStatus('Playback Error', 'error'))
  player.on('wentLive', () => updateStatus('Live Edge', 'live'))
  player.on('snapshotTaken', (event: any, filename: string) => updateStatus('Snapshot Saved', 'success'))
  player.on('snapshotFailed', (event: any, error: string) => updateStatus(`Snapshot Failed: ${error}`, 'error'))
  
  // Function to add buttons with retry
  function addButtonsToControlBar() {
    const controlBar = player.el().querySelector('.vjs-control-bar')
    
    if (!controlBar) {
      console.log('Control bar not found, retrying in 100ms...')
      setTimeout(addButtonsToControlBar, 100)
      return
    }
    
    console.log('Adding custom buttons to control bar')
    
    // Create Check Stream button
    const checkBtn = document.createElement('button')
    checkBtn.className = 'vjs-check-stream-control vjs-control vjs-button'
    checkBtn.innerHTML = '<span class="vjs-control-text">Check Stream</span>📡'
    checkBtn.title = 'Check Stream'
    checkBtn.style.fontSize = '1.2em'
    checkBtn.style.padding = '0 0.5em'
    checkBtn.addEventListener('click', function(e) {
      e.preventDefault()
      e.stopPropagation()
      
      if (!hlsUrl) return
      
      console.log('Checking stream availability...')
      
      fetch(hlsUrl, {
        method: 'GET',
        signal: AbortSignal.timeout(5000)
      })
      .then(response => {
        if (response.ok) {
          return response.text()
        }
        throw new Error(`${response.status} ${response.statusText}`)
      })
      .then(text => {
        if (text.includes('#EXTM3U')) {
          console.log('Stream is available (valid M3U8)')
          player.trigger('streamAvailable')
        } else {
          console.log('Invalid M3U8 response')
          player.trigger('streamUnavailable', 'Invalid M3U8')
        }
      })
      .catch(error => {
        console.log(`Stream check failed: ${error.message}`)
        player.trigger('streamUnavailable', error.message)
      })
    })
    
    // Create Load Stream button
    const loadBtn = document.createElement('button')
    loadBtn.className = 'vjs-load-stream-control vjs-control vjs-button'
    loadBtn.innerHTML = '<span class="vjs-control-text">Load Stream</span>📺'
    loadBtn.title = 'Load Stream'
    loadBtn.style.fontSize = '1.2em'
    loadBtn.style.padding = '0 0.5em'
    loadBtn.addEventListener('click', function(e) {
      e.preventDefault()
      e.stopPropagation()
      
      if (!hlsUrl) return
      
      console.log('Loading stream source:', hlsUrl)
      
      player.src({
        src: hlsUrl,
        type: 'application/x-mpegURL'
      })
      
      player.load()
      player.trigger('streamLoaded')
    })
    
    // Create Snapshot button
    const snapshotBtn = document.createElement('button')
    snapshotBtn.className = 'vjs-snapshot-control vjs-control vjs-button'
    snapshotBtn.innerHTML = '<span class="vjs-control-text">Take Snapshot</span>📷'
    snapshotBtn.title = 'Take Snapshot'
    snapshotBtn.style.fontSize = '1.2em'
    snapshotBtn.style.padding = '0 0.5em'
    snapshotBtn.addEventListener('click', function(e) {
      e.preventDefault()
      e.stopPropagation()
      
      takeSnapshot()
    })
    
    // Snapshot functionality
    function takeSnapshot() {
      const video = player.el().querySelector('video')
      if (!video || video.readyState < 2) {
        console.log('Video not ready for snapshot')
        player.trigger('snapshotFailed', 'Video not ready')
        return
      }
      
      try {
        // Create canvas element
        const canvas = document.createElement('canvas')
        const ctx = canvas.getContext('2d')
        
        if (!ctx) {
          throw new Error('Could not get canvas context')
        }
        
        // Set canvas size to video dimensions
        canvas.width = video.videoWidth || video.clientWidth
        canvas.height = video.videoHeight || video.clientHeight
        
        // Draw current video frame to canvas
        ctx.drawImage(video, 0, 0, canvas.width, canvas.height)
        
        // Convert to blob and download
        canvas.toBlob((blob) => {
          if (!blob) {
            console.log('Failed to create image blob')
            player.trigger('snapshotFailed', 'Failed to create image')
            return
          }
          
          // Create download link
          const url = URL.createObjectURL(blob)
          const link = document.createElement('a')
          link.href = url
          link.download = `snapshot-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.png`
          
          // Trigger download
          document.body.appendChild(link)
          link.click()
          document.body.removeChild(link)
          
          // Clean up
          URL.revokeObjectURL(url)
          
          console.log('Snapshot saved:', link.download)
          player.trigger('snapshotTaken', link.download)
        }, 'image/png')
        
      } catch (error: any) {
        console.error('Snapshot failed:', error)
        player.trigger('snapshotFailed', error.message)
      }
    }
    
    // Add buttons to control bar (insert before fullscreen button)
    const fullscreenBtn = controlBar.querySelector('.vjs-fullscreen-control')
    if (fullscreenBtn) {
      controlBar.insertBefore(checkBtn, fullscreenBtn)
      controlBar.insertBefore(loadBtn, fullscreenBtn)
      controlBar.insertBefore(snapshotBtn, fullscreenBtn)
    } else {
      controlBar.appendChild(checkBtn)
      controlBar.appendChild(loadBtn)
      controlBar.appendChild(snapshotBtn)
    }
    
    console.log('Custom buttons added successfully')
  }
  
  // Add buttons after a short delay to ensure control bar is ready
  setTimeout(addButtonsToControlBar, 250)
}

// Clean up Video.js instance
export const disposeVideoJS = (player: VideoJSPlayer | null) => {
  if (player && !player.isDisposed()) {
    try {
      player.dispose()
    } catch (error) {
      console.warn('Error disposing Video.js player:', error)
    }
  }
}

// Event handlers for live streaming
export const setupLiveStreamHandlers = (player: VideoJSPlayer) => {
  // Handle live edge seeking
  player.on('seeked', () => {
    try {
      const liveTracker = (player as any).liveTracker
      if (liveTracker && typeof liveTracker.atLiveEdge === 'function') {
        if (liveTracker.isLive() && liveTracker.atLiveEdge()) {
          // User is at live edge
          player.addClass('vjs-at-live-edge')
        } else {
          player.removeClass('vjs-at-live-edge')
        }
      } else {
        // Fallback live edge detection
        const duration = player.duration()
        const currentTime = player.currentTime()
        const seekableEnd = player.seekable().end(0)
        
        if (duration === Infinity && seekableEnd > 0 && currentTime !== undefined) {
          const atLiveEdge = Math.abs(currentTime - seekableEnd) < 5
          if (atLiveEdge) {
            player.addClass('vjs-at-live-edge')
          } else {
            player.removeClass('vjs-at-live-edge')
          }
        }
      }
    } catch (error) {
      console.debug('Live edge detection failed:', error)
    }
  })
  
  // Handle errors
  player.on('error', () => {
    const error = player.error()
    console.error('Video.js error:', error)
    
    // Custom error handling can be added here
    // This integrates with the existing StreamErrorAlert component
  })
  
  // Handle live tracking
  player.on('liveresync', () => {
    console.log('Live stream resynced')
  })
  
  // Handle when stream goes live/offline
  player.on('durationchange', () => {
    const duration = player.duration()
    if (duration === Infinity) {
      player.addClass('vjs-live')
    } else {
      player.removeClass('vjs-live')
    }
  })
}