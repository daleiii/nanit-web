/* Motion/Sound Activity Chart - DISABLED
 * This component has been temporarily disabled since motion/sound activity
 * tracking has been removed from the application.
 */

'use client'

// Simplified interface without ActivityEvent dependency
interface ActivityChartProps {
  events?: any[]  // Generic array instead of ActivityEvent[]
  isLoading?: boolean
  timeframe: string
}

export default function ActivityChart({ events, isLoading, timeframe }: ActivityChartProps) {
  // Show a placeholder since motion/sound activity is disabled
  return (
    <div className="h-64 bg-nanit-gray-50 rounded flex items-center justify-center border-2 border-dashed border-nanit-gray-300">
      <div className="text-center text-nanit-gray-500">
        <div className="text-lg font-medium mb-2">📊 Activity Chart</div>
        <div className="text-sm">Motion & Sound activity tracking is currently disabled</div>
        <div className="text-xs mt-1 text-nanit-gray-400">Timeframe: {timeframe}</div>
      </div>
    </div>
  )
}