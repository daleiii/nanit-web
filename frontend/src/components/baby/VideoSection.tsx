'use client'

import { useState } from 'react'
import { api } from '@/lib/api'
import type { Baby } from '@/types/api'
import { useVideoPlayer } from '@/hooks/useVideoPlayer'

interface VideoSectionProps {
  baby: Baby
}

export default function VideoSection({ baby }: VideoSectionProps) {
  // Get HLS URL for this baby
  const hlsUrl = api.getHLSUrl(baby.uid)
  
  // Use Video.js player hook with integrated controls
  const {
    videoRef,
    player,
    isStreaming,
    isLive,
    isLoading,
    error,
  } = useVideoPlayer({ hlsUrl })

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold text-nanit-gray-800">Live Video Stream</h3>
      
      {/* Error display */}
      {error && (
        <div className="p-3 bg-red-100 border border-red-300 text-red-700 rounded">
          Error: {error}
        </div>
      )}
      
      
      {/* Video.js Player with integrated controls */}
      <div className="bg-black rounded-lg overflow-hidden">
        <video
          ref={videoRef}
          className="video-js vjs-default-skin w-full h-auto"
          controls
          preload="none"
          style={{ minHeight: '300px' }}
          data-setup="{}"
        >
          <p className="vjs-no-js">
            To view this video please enable JavaScript, and consider upgrading to a web browser that
            <a href="https://videojs.com/html5-video-support/" target="_blank" rel="noopener noreferrer">
              supports HTML5 video
            </a>.
          </p>
        </video>
      </div>
    </div>
  )
}