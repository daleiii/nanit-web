'use client'

import { useState } from 'react'
import dynamic from 'next/dynamic'
import { api } from '@/lib/api'
import { useHistoricalData } from '@/hooks/useHistoricalData'
import { useTemperatureUnit } from '@/hooks/useTemperatureUnit'
import type { Baby } from '@/types/api'
import LoadingSpinner from '@/components/ui/LoadingSpinner'

// Import Chart.js setup
import '@/lib/chartSetup'

// Dynamically import chart components to avoid SSR issues
const TemperatureHumidityChart = dynamic(
  () => import('@/components/charts/TemperatureHumidityChart'),
  { ssr: false }
)


const DayNightChart = dynamic(
  () => import('@/components/charts/DayNightChart'),
  { ssr: false }
)

interface HistoricalDataProps {
  baby: Baby
}

export default function HistoricalData({ baby }: HistoricalDataProps) {
  const [selectedTimeframe, setSelectedTimeframe] = useState('24h')
  const [isResetting, setIsResetting] = useState(false)
  const { formatTemperature, unit } = useTemperatureUnit()

  const {
    sensorData,
    summary,
    analytics,
    isLoading,
    isError,
    refreshAll,
  } = useHistoricalData(baby.uid, selectedTimeframe)

  const timeframeOptions = [
    { value: '1h', label: 'Last Hour' },
    { value: '6h', label: 'Last 6 Hours' },
    { value: '24h', label: 'Last 24 Hours' },
    { value: '7d', label: 'Last 7 Days' },
    { value: '30d', label: 'Last 30 Days' },
  ]

  const handleReset = async () => {
    if (confirm('Are you sure you want to reset all historical data for this baby? This action cannot be undone.')) {
      setIsResetting(true)
      try {
        await api.resetHistoricalData(baby.uid)
        refreshAll()
      } catch (error) {
        console.error('Failed to reset data:', error)
        alert('Failed to reset data. Please try again.')
      } finally {
        setIsResetting(false)
      }
    }
  }

  return (
    <div className="space-y-6">
      <h3 className="text-lg font-semibold text-nanit-gray-800">Historical Data</h3>
      
      {/* Controls */}
      <div className="flex flex-wrap items-center gap-3">
        <select
          value={selectedTimeframe}
          onChange={(e) => setSelectedTimeframe(e.target.value)}
          className="px-3 py-2 border border-nanit-gray-300 rounded-md bg-white text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {timeframeOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        
        <button
          onClick={refreshAll}
          disabled={isLoading}
          className="btn btn-primary text-sm"
        >
          {isLoading ? (
            <div className="flex items-center gap-2">
              <LoadingSpinner size="sm" />
              <span>Loading...</span>
            </div>
          ) : (
            '🔄 Refresh'
          )}
        </button>
        
        <button
          onClick={handleReset}
          disabled={isResetting || isLoading}
          className="btn btn-danger text-sm"
        >
          {isResetting ? (
            <div className="flex items-center gap-2">
              <LoadingSpinner size="sm" />
              <span>Resetting...</span>
            </div>
          ) : (
            '🗑️ Reset Data'
          )}
        </button>
      </div>

      {/* Error State */}
      {isError && (
        <div className="bg-red-50 border-l-4 border-red-500 p-3 rounded">
          <div className="text-sm text-red-700">
            Failed to load historical data. Please try refreshing.
          </div>
        </div>
      )}

      {/* Charts */}
      <div className="space-y-6">
        {/* Temperature & Humidity Chart */}
        <div className="bg-white border border-nanit-gray-200 rounded-lg p-4">
          <h4 className="font-semibold text-nanit-gray-800 mb-4">Temperature & Humidity</h4>
          <div className="h-64">
            <TemperatureHumidityChart key={unit} data={sensorData} isLoading={isLoading} />
          </div>
        </div>


        {/* Day/Night Pattern */}
        <div className="bg-white border border-nanit-gray-200 rounded-lg p-4">
          <h4 className="font-semibold text-nanit-gray-800 mb-4">Day/Night Pattern</h4>
          <DayNightChart analytics={analytics || null} isLoading={isLoading} />
        </div>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white border border-nanit-gray-200 rounded-lg p-4 text-center">
          <h5 className="text-sm font-semibold text-nanit-gray-600 mb-2">Temperature</h5>
          <div className="space-y-1 text-sm text-nanit-gray-600">
            <div>Avg: <span className="font-medium">
              {summary?.avg_temperature ? formatTemperature(summary.avg_temperature) : '--'}
            </span></div>
            <div>Min: <span className="font-medium">
              {summary?.min_temperature ? formatTemperature(summary.min_temperature) : '--'}
            </span></div>
            <div>Max: <span className="font-medium">
              {summary?.max_temperature ? formatTemperature(summary.max_temperature) : '--'}
            </span></div>
          </div>
        </div>

        <div className="bg-white border border-nanit-gray-200 rounded-lg p-4 text-center">
          <h5 className="text-sm font-semibold text-nanit-gray-600 mb-2">Humidity</h5>
          <div className="space-y-1 text-sm text-nanit-gray-600">
            <div>Avg: <span className="font-medium">
              {summary?.avg_humidity !== undefined ? `${summary.avg_humidity.toFixed(1)}%` : '--'}
            </span></div>
            <div>Min: <span className="font-medium">
              {summary?.min_humidity !== undefined ? `${summary.min_humidity.toFixed(1)}%` : '--'}
            </span></div>
            <div>Max: <span className="font-medium">
              {summary?.max_humidity !== undefined ? `${summary.max_humidity.toFixed(1)}%` : '--'}
            </span></div>
          </div>
        </div>

        <div className="bg-white border border-nanit-gray-200 rounded-lg p-4 text-center">
          <h5 className="text-sm font-semibold text-nanit-gray-600 mb-2">Day/Night</h5>
          <div className="space-y-1 text-sm text-nanit-gray-600">
            <div>Day: <span className="font-medium">
              {summary?.day_mode_percentage !== undefined ? `${summary.day_mode_percentage.toFixed(1)}%` : '--%'}
            </span></div>
            <div>Night: <span className="font-medium">
              {summary?.night_mode_percentage !== undefined ? `${summary.night_mode_percentage.toFixed(1)}%` : '--%'}
            </span></div>
            <div>Transitions: <span className="font-medium">
              {analytics?.mode_transitions !== undefined ? analytics.mode_transitions : '--'}
            </span></div>
          </div>
        </div>
      </div>
    </div>
  )
}