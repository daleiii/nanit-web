'use client'

import { Line } from 'react-chartjs-2'
import { temperatureHumidityOptions } from '@/lib/chartSetup'
import { useTemperatureUnit } from '@/hooks/useTemperatureUnit'
import type { SensorReading } from '@/types/api'

interface TemperatureHumidityChartProps {
  data: SensorReading[]
  isLoading?: boolean
}

export default function TemperatureHumidityChart({ data, isLoading }: TemperatureHumidityChartProps) {
  const { unit, convertTemperature } = useTemperatureUnit()

  if (isLoading) {
    return (
      <div className="h-64 bg-nanit-gray-50 rounded flex items-center justify-center">
        <div className="text-nanit-gray-500">Loading chart data...</div>
      </div>
    )
  }

  if (!data || data.length === 0) {
    return (
      <div className="h-64 bg-nanit-gray-50 rounded flex items-center justify-center">
        <div className="text-nanit-gray-500">No data available for selected timeframe</div>
      </div>
    )
  }

  // Filter and prepare data
  const temperatureData = data
    .filter(reading => reading.temperature_celsius !== undefined && reading.temperature_celsius > 0)
    .map(reading => ({
      x: reading.timestamp * 1000, // Convert to milliseconds
      y: convertTemperature(reading.temperature_celsius!),
    }))

  const humidityData = data
    .filter(reading => reading.humidity_percent !== undefined && reading.humidity_percent > 0)
    .map(reading => ({
      x: reading.timestamp * 1000,
      y: reading.humidity_percent!,
    }))

  const chartData = {
    datasets: [
      {
        label: `Temperature (°${unit === 'celsius' ? 'C' : 'F'})`,
        data: temperatureData,
        borderColor: '#ef4444',
        backgroundColor: 'rgba(239, 68, 68, 0.1)',
        yAxisID: 'y',
        tension: 0.3,
        pointRadius: 2,
        pointHoverRadius: 4,
        borderWidth: 2,
      },
      {
        label: 'Humidity (%)',
        data: humidityData,
        borderColor: '#06b6d4',
        backgroundColor: 'rgba(6, 182, 212, 0.1)',
        yAxisID: 'y1',
        tension: 0.3,
        pointRadius: 2,
        pointHoverRadius: 4,
        borderWidth: 2,
      },
    ],
  }

  // Update Y-axis title based on temperature unit
  const options = {
    ...temperatureHumidityOptions,
    scales: {
      ...temperatureHumidityOptions.scales,
      y: {
        ...temperatureHumidityOptions.scales.y,
        title: {
          ...temperatureHumidityOptions.scales.y.title,
          text: `Temperature (°${unit === 'celsius' ? 'C' : 'F'})`,
        },
      },
    },
  }

  return <Line data={chartData} options={options} />
}