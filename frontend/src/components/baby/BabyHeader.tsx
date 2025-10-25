import { useState, useEffect } from 'react'
import { api } from '@/lib/api'
import { errorTooltipConfig } from '@/lib/tooltipSetup'
import type { Baby, StreamStatusResponse, HealthResponse } from '@/types/api'

interface BabyHeaderProps {
  baby: Baby
}

export default function BabyHeader({ baby }: BabyHeaderProps) {
  const [streamStatus, setStreamStatus] = useState<StreamStatusResponse | null>(null)
  const [health, setHealth] = useState<HealthResponse | null>(null)
  
  useEffect(() => {
    let pollInterval: NodeJS.Timeout
    
    const pollHealth = async () => {
      try {
        const healthStatus = await api.getHealth(baby.uid)
        setHealth(healthStatus)
      } catch (error) {
        // Health status not available
        setHealth(null)
      }
    }
    
    const pollStreamStatus = async () => {
      try {
        const status = await api.getStreamStatus(baby.uid)
        setStreamStatus(status)
      } catch (error) {
        // Stream status not available
        setStreamStatus(null)
      }
    }
    
    // Always poll health, regardless of websocket status
    pollHealth() // Initial poll
    pollInterval = setInterval(pollHealth, 15000) // Poll every 15 seconds
    
    // Also poll stream status if websocket is alive for backward compatibility
    if (baby.websocket_alive) {
      pollStreamStatus()
    }
    
    return () => {
      if (pollInterval) {
        clearInterval(pollInterval)
      }
    }
  }, [baby.uid, baby.websocket_alive])
  
  const getCameraStatusInfo = () => {
    if (!health) {
      return { text: 'Checking...', color: 'bg-gray-500', tooltip: 'Loading camera status...' }
    }
    
    const details = health.details
    const isStreaming = streamStatus?.status === 'running'
    
    // Determine primary status based on health and streaming state
    switch (health.overall_health) {
      case 'healthy':
        if (isStreaming) {
          return { 
            text: 'Online & Streaming', 
            color: 'bg-green-500', 
            tooltip: 'Camera online and actively streaming video'
          }
        }
        return { 
          text: 'Camera Online', 
          color: 'bg-green-500', 
          tooltip: 'Camera connected and ready to stream'
        }
      case 'degraded':
        return { 
          text: 'Camera Issues', 
          color: 'bg-yellow-500', 
          tooltip: 'Camera connected but has warnings. Click Settings → Devices for details.'
        }
      case 'starting':
        return { 
          text: 'Camera Starting', 
          color: 'bg-blue-500', 
          tooltip: 'Camera initializing, please wait...'
        }
      case 'unhealthy':
      default:
        if (!baby.websocket_alive) {
          return { 
            text: 'Camera Offline', 
            color: 'bg-red-500', 
            tooltip: 'Camera disconnected from Nanit servers'
          }
        }
        return { 
          text: 'Camera Error', 
          color: 'bg-red-500', 
          tooltip: 'Camera has critical issues. Check Settings → Devices for troubleshooting.'
        }
    }
  }

  const cameraStatus = getCameraStatusInfo()
  
  return (
    <div className="bg-gradient-nanit text-white p-6">
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold">{baby.name}</h2>
        
        <div className="flex items-center">
          {/* Single Camera Status */}
          <div 
            className="flex items-center gap-2 bg-white/20 px-4 py-2 rounded-full text-sm cursor-help"
            data-tooltip-id="app-tooltip"
            data-tooltip-content={cameraStatus.tooltip}
            data-tooltip-place={errorTooltipConfig.place}
            data-tooltip-delay-show={errorTooltipConfig.delayShow}
          >
            <div className={`w-2.5 h-2.5 rounded-full ${cameraStatus.color}`} />
            <span className="font-medium">{cameraStatus.text}</span>
          </div>
        </div>
      </div>
    </div>
  )
}