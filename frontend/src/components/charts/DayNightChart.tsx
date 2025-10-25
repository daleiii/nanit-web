'use client'

import { timelineTooltipConfig } from '@/lib/tooltipSetup'
import type { DayNightAnalytics } from '@/types/api'

interface DayNightChartProps {
  analytics: DayNightAnalytics | null
  isLoading?: boolean
}

interface TimelinePeriod {
  mode: 'day' | 'night'
  startTime: number
  endTime: number
  duration: number // in minutes
  percentage: number
}

function formatTime(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true
  })
}

function formatDuration(minutes: number): string {
  const hours = Math.floor(minutes / 60)
  const mins = Math.round(minutes % 60)
  
  if (hours === 0) {
    return `${mins} min`
  } else if (mins === 0) {
    return `${hours}h`
  } else {
    return `${hours}h ${mins}m`
  }
}

function formatTooltipContent(period: TimelinePeriod): string {
  return `${period.mode.charAt(0).toUpperCase() + period.mode.slice(1)} Mode
Start: ${formatTime(period.startTime)}
End: ${formatTime(period.endTime)}
Duration: ${formatDuration(period.duration)}`
}

function buildTimelinePeriods(analytics: DayNightAnalytics): TimelinePeriod[] {
  const periods: TimelinePeriod[] = []
  const totalDuration = analytics.end_time - analytics.start_time
  
  if (!analytics.day_night_changes || analytics.day_night_changes.length === 0) {
    // No transitions - single period based on dominant mode
    const mode = analytics.night_mode_percentage > analytics.day_mode_percentage ? 'night' : 'day'
    periods.push({
      mode,
      startTime: analytics.start_time,
      endTime: analytics.end_time,
      duration: totalDuration / 60,
      percentage: 100
    })
    return periods
  }
  
  // Determine initial mode from first transition
  let currentTime = analytics.start_time
  let currentMode: 'day' | 'night' = analytics.day_night_changes[0].from_night ? 'night' : 'day'
  
  analytics.day_night_changes.forEach(change => {
    // Add period before this transition
    const duration = change.timestamp - currentTime
    periods.push({
      mode: currentMode,
      startTime: currentTime,
      endTime: change.timestamp,
      duration: duration / 60,
      percentage: (duration / totalDuration) * 100
    })
    
    currentTime = change.timestamp
    currentMode = change.to_night ? 'night' : 'day'
  })
  
  // Add final period
  const finalDuration = analytics.end_time - currentTime
  if (finalDuration > 0) {
    periods.push({
      mode: currentMode,
      startTime: currentTime,
      endTime: analytics.end_time,
      duration: finalDuration / 60,
      percentage: (finalDuration / totalDuration) * 100
    })
  }
  
  return periods
}

export default function DayNightChart({ analytics, isLoading }: DayNightChartProps) {

  if (isLoading) {
    return (
      <div className="h-64 bg-gray-50 rounded flex items-center justify-center">
        <div className="text-gray-500">Loading day/night data...</div>
      </div>
    )
  }

  if (!analytics) {
    return (
      <div className="h-64 bg-gray-50 rounded flex items-center justify-center">
        <div className="text-gray-500">No day/night pattern data available</div>
      </div>
    )
  }

  const periods = buildTimelinePeriods(analytics)

  return (
    <div className="space-y-4">
      {/* Timeline Chart */}
      <div className="relative">
        <div className="h-16 bg-gray-50 rounded-lg border overflow-hidden">
          <div className="flex h-full">
            {periods.map((period, index) => (
              <div
                key={index}
                className={`h-full cursor-pointer transition-all duration-200 hover:brightness-110 hover:scale-105 hover:shadow-lg relative ${
                  period.mode === 'day' 
                    ? 'bg-gradient-to-r from-yellow-400 to-orange-400' 
                    : 'bg-gradient-to-r from-indigo-600 to-purple-700'
                }`}
                style={{ 
                  width: `${period.percentage}%`,
                  minWidth: '8px' // Ensure even small periods are hoverable
                }}
                data-tooltip-id="app-tooltip"
                data-tooltip-content={formatTooltipContent(period)}
                data-tooltip-place={timelineTooltipConfig.place}
                data-tooltip-delay-show={timelineTooltipConfig.delayShow}
              />
            ))}
          </div>
        </div>
        
        {/* Time axis labels */}
        <div className="flex justify-between text-xs text-gray-500 mt-2 px-1">
          <span>{formatTime(analytics.start_time)}</span>
          <span>{formatTime(analytics.end_time)}</span>
        </div>
      </div>
    </div>
  )
}