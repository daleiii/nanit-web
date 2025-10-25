import { useRef, useEffect, useState, useCallback } from 'react'
import videojs from 'video.js'
import type { StreamError } from '@/types/api'

// Define Video.js player type
type VideoJSPlayer = ReturnType<typeof videojs>
import { 
  createVideoJSOptions, 
  initializeVideoJS, 
  disposeVideoJS, 
  setupLiveStreamHandlers,
  addCustomControlsToPlayer 
} from '@/lib/videojs-setup'

interface UseVideoPlayerOptions {
  hlsUrl: string
}

interface UseVideoPlayerReturn {
  videoRef: (element: HTMLVideoElement | null) => void
  player: VideoJSPlayer | null
  isStreaming: boolean
  isLive: boolean
  isLoading: boolean
  error: string | null
}

export function useVideoPlayer({
  hlsUrl,
}: UseVideoPlayerOptions): UseVideoPlayerReturn {
  const playerRef = useRef<VideoJSPlayer | null>(null)
  const liveCheckIntervalRef = useRef<NodeJS.Timeout | null>(null)
  const streamRetryIntervalRef = useRef<NodeJS.Timeout | null>(null)
  const retryStartTimeRef = useRef<number | null>(null)
  
  const [isStreaming, setIsStreaming] = useState(false)
  const [isLive, setIsLive] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Initialize Video.js player when element is mounted
  const initializePlayer = useCallback((videoElement: HTMLVideoElement) => {
    if (!videoElement || playerRef.current) return

    console.log('Initializing Video.js player')

    // Initialize Video.js
    const vjs = initializeVideoJS()
    
    // Create player options with HLS URL
    const options = createVideoJSOptions(hlsUrl)
    
    // Create the player
    const player = vjs(videoElement, options)
    playerRef.current = player
    
    // Add custom controls to the player with HLS URL
    addCustomControlsToPlayer(player, hlsUrl)
    
    console.log('Video.js player created successfully')

    // Player event handlers
    player.ready(() => {
      console.log('Video.js player is ready')
      
      // Start smart retry logic for stream availability
      if (hlsUrl) {
        startStreamRetry()
      }
    })

    player.on('loadstart', () => {
      setIsLoading(true)
      setError(null)
    })

    player.on('canplay', () => {
      setIsLoading(false)
      setIsStreaming(true)
    })

    player.on('play', () => {
      setIsStreaming(true)
    })

    player.on('pause', () => {
      setIsStreaming(false)
    })

    player.on('ended', () => {
      setIsStreaming(false)
      setIsLive(false)
    })

    player.on('error', () => {
      const playerError = player.error()
      setIsLoading(false)
      setIsStreaming(false)
      setError(playerError ? playerError.message : 'Video playback error')
    })

    // Live tracking
    player.on('durationchange', () => {
      const duration = player.duration()
      setIsLive(duration === Infinity)
    })
  }, [])

  // Create callback ref for video element
  const videoRef = useCallback((videoElement: HTMLVideoElement | null) => {
    // Cleanup existing player if element is being removed
    if (!videoElement && playerRef.current) {
      console.log('Cleaning up Video.js player')
      disposeVideoJS(playerRef.current)
      playerRef.current = null
      return
    }

    // Initialize player if element is being added and no player exists
    if (videoElement && !playerRef.current) {
      // Small delay to ensure React has fully mounted the element
      setTimeout(() => {
        if (videoElement.isConnected && !playerRef.current) {
          initializePlayer(videoElement)
        }
      }, 0)
    }
  }, [initializePlayer])

  // Handle live edge detection
  const checkLiveStatus = useCallback(() => {
    if (!playerRef.current) return
    
    try {
      // Check if liveTracker is available (Video.js 7.x+)
      const liveTracker = (playerRef.current as any).liveTracker
      if (liveTracker && typeof liveTracker.atLiveEdge === 'function') {
        const atLiveEdge = liveTracker.atLiveEdge()
        if (atLiveEdge !== isLive) {
          setIsLive(atLiveEdge)
        }
      } else {
        // Fallback: check if we're at the end of seekable range (live edge)
        const duration = playerRef.current.duration()
        const currentTime = playerRef.current.currentTime()
        const seekableEnd = playerRef.current.seekable().end(0)
        
        if (duration === Infinity && seekableEnd > 0 && currentTime !== undefined) {
          const atLiveEdge = Math.abs(currentTime - seekableEnd) < 5 // Within 5 seconds of live
          if (atLiveEdge !== isLive) {
            setIsLive(atLiveEdge)
          }
        }
      }
    } catch (error) {
      // Silent fail if live tracking isn't available
      console.debug('Live tracking not available:', error)
    }
  }, [isLive])

  // Smart retry logic for stream availability
  const checkStreamAvailability = useCallback(async (): Promise<boolean> => {
    if (!hlsUrl || !playerRef.current) return false
    
    try {
      const response = await fetch(hlsUrl, {
        method: 'GET',
        signal: AbortSignal.timeout(5000)
      })
      
      if (response.ok) {
        const text = await response.text()
        if (text.includes('#EXTM3U')) {
          return true
        }
      }
      return false
    } catch (error) {
      return false
    }
  }, [hlsUrl])

  const startStreamRetry = useCallback(() => {
    if (!hlsUrl || !playerRef.current) return
    
    // Stop any existing retry
    stopStreamRetry()
    
    retryStartTimeRef.current = Date.now()
    setError(null)
    
    console.log('Starting smart stream retry...')
    
    const attemptStreamLoad = async () => {
      const player = playerRef.current
      if (!player) return
      
      const isAvailable = await checkStreamAvailability()
      
      if (isAvailable) {
        console.log('Stream is available, loading...')
        player.src({
          src: hlsUrl,
          type: 'application/x-mpegURL'
        })
        player.load()
        player.trigger('streamAvailable')
        stopStreamRetry() // Stop retrying once successful
        return
      }
      
      // Determine retry interval based on elapsed time
      const elapsedMs = Date.now() - (retryStartTimeRef.current || 0)
      const elapsedMinutes = elapsedMs / (1000 * 60)
      
      let nextInterval: number
      if (elapsedMinutes < 1) {
        // First minute: check every 2-3 seconds
        nextInterval = 2500
        player.trigger('streamUnavailable', 'Checking for stream...')
      } else {
        // After first minute: check every 30 seconds  
        nextInterval = 30000
        player.trigger('streamUnavailable', 'Waiting for stream...')
      }
      
      console.log(`Stream not ready, retrying in ${nextInterval/1000}s...`)
      
      streamRetryIntervalRef.current = setTimeout(attemptStreamLoad, nextInterval)
    }
    
    // Start immediate check
    attemptStreamLoad()
  }, [hlsUrl, checkStreamAvailability])

  const stopStreamRetry = useCallback(() => {
    if (streamRetryIntervalRef.current) {
      clearTimeout(streamRetryIntervalRef.current)
      streamRetryIntervalRef.current = null
    }
    retryStartTimeRef.current = null
  }, [])

  // Setup live tracking interval
  useEffect(() => {
    if (playerRef.current && isStreaming) {
      liveCheckIntervalRef.current = setInterval(checkLiveStatus, 1000)
    } else {
      if (liveCheckIntervalRef.current) {
        clearInterval(liveCheckIntervalRef.current)
      }
    }

    // Cleanup on unmount
    return () => {
      if (liveCheckIntervalRef.current) {
        clearInterval(liveCheckIntervalRef.current)
      }
      if (streamRetryIntervalRef.current) {
        clearTimeout(streamRetryIntervalRef.current)
      }
    }
  }, [checkLiveStatus, isStreaming])

  // Cleanup player on unmount
  useEffect(() => {
    return () => {
      stopStreamRetry()
      disposeVideoJS(playerRef.current)
      playerRef.current = null
    }
  }, [stopStreamRetry])


  return {
    videoRef,
    player: playerRef.current,
    isStreaming,
    isLive,
    isLoading,
    error,
  }
}