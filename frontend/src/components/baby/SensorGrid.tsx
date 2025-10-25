'use client'

import { useTemperatureUnit } from '@/hooks/useTemperatureUnit'
import { formatRelativeTime } from '@/lib/utils'
import { sensorTooltipConfig } from '@/lib/tooltipSetup'
import type { Baby } from '@/types/api'

interface SensorCardProps {
  title: string
  value: string
  type: 'temperature' | 'humidity' | 'night-mode' | 'night-light'
  tooltip?: string
  onClick?: () => void
  className?: string
}

function SensorCard({ title, value, type, tooltip, onClick, className }: SensorCardProps) {
  return (
    <div 
      className={`sensor-card ${type} ${onClick ? 'cursor-pointer' : ''} ${className}`}
      onClick={onClick}
      data-tooltip-id="app-tooltip"
      data-tooltip-content={tooltip}
      data-tooltip-place={sensorTooltipConfig.place}
      data-tooltip-delay-show={sensorTooltipConfig.delayShow}
    >
      <div className="text-sm font-medium text-nanit-gray-600 mb-1">
        {title}
      </div>
      <div className="text-xl font-bold text-nanit-gray-800">
        {value}
      </div>
    </div>
  )
}

interface SensorGridProps {
  baby: Baby
}

export default function SensorGrid({ baby }: SensorGridProps) {
  const { formatTemperature, toggleUnit } = useTemperatureUnit()

  // Debug logging to troubleshoot data issues
  console.log('🔧 SensorGrid received baby data:', baby)
  console.log('🌡️ Temperature value:', baby.temperature, 'type:', typeof baby.temperature)
  console.log('💧 Humidity value:', baby.humidity, 'type:', typeof baby.humidity)

  const formatHumidity = (humidity: number | undefined): string => {
    console.log('🔧 formatHumidity called with:', humidity, 'type:', typeof humidity)
    if (humidity === undefined || humidity === null || humidity <= 0) {
      console.log('🔧 formatHumidity returning -- due to invalid value')
      return '--%'
    }
    const result = `${humidity.toFixed(1)}%`
    console.log('🔧 formatHumidity returning:', result)
    return result
  }

  const formatNightMode = (isNight: boolean | undefined): string => {
    console.log('🔧 formatNightMode called with:', isNight, 'type:', typeof isNight)
    if (isNight === undefined || isNight === null) {
      return '--'
    }
    return isNight ? 'Night' : 'Day'
  }

  const formatNightLight = (nightLight: boolean | undefined): string => {
    console.log('🔧 formatNightLight called with:', nightLight, 'type:', typeof nightLight)
    if (nightLight === undefined || nightLight === null) {
      return '--'
    }
    return nightLight ? 'On' : 'Off'
  }

  // Safety check: if baby object is completely undefined
  if (!baby) {
    console.error('❌ SensorGrid: baby object is undefined!')
    return (
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="text-red-500 text-center col-span-full">
          Error: Baby data not available
        </div>
      </div>
    )
  }


  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <SensorCard
        title="Temperature"
        value={formatTemperature(baby.temperature)}
        type="temperature"
        onClick={toggleUnit}
        tooltip="Click to toggle °C/°F"
      />
      
      <SensorCard
        title="Humidity"
        value={formatHumidity(baby.humidity)}
        type="humidity"
      />
      
      <SensorCard
        title="Night Mode"
        value={formatNightMode(baby.is_night)}
        type="night-mode"
        className={baby.is_night ? 'bg-purple-50 border-l-purple-600' : ''}
      />
      
      <SensorCard
        title="Night Light"
        value={formatNightLight(baby.night_light)}
        type="night-light"
        className={baby.night_light ? 'bg-cyan-50 border-l-cyan-600' : ''}
      />
    </div>
  )
}