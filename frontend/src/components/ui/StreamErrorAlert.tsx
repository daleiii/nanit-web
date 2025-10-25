'use client'

import type { StreamError } from '@/types/api'

interface StreamErrorAlertProps {
  error: StreamError | null
  baby_uid: string
  onRetry?: () => void
  className?: string
}

function getErrorTitle(errorType: string): string {
  switch (errorType) {
    case 'rtmp_connection':
      return 'RTMP Connection Failed'
    case 'p2p_punch':
      return 'Network Connection Issue'
    case 'camera_offline':
      return 'Camera Offline'
    case 'transcoding':
      return 'Video Processing Error'
    case 'connection_limit':
      return 'Too Many Apps Connected'
    default:
      return 'Streaming Error'
  }
}

function getErrorExplanation(errorType: string, message: string): string {
  switch (errorType) {
    case 'rtmp_connection':
      return 'The baby monitor cannot establish an RTMP connection to stream video. This is usually a network connectivity issue.'
    case 'p2p_punch':
      return 'Cannot establish peer-to-peer connection between the camera and streaming server. Check your network firewall settings.'
    case 'camera_offline':
      return 'The baby monitor appears to be offline or unreachable.'
    case 'transcoding':
      return 'Video processing failed. The camera may be sending corrupted video data.'
    case 'connection_limit':
      return 'Nanit limits the number of mobile apps that can stream simultaneously. The official Nanit app on your phone or tablet is currently using this connection.'
    default:
      return message || 'An unknown streaming error occurred.'
  }
}

function getErrorSolutions(errorType: string): string[] {
  switch (errorType) {
    case 'rtmp_connection':
    case 'p2p_punch':
      return [
        'Check that your RTMP server port (1940) is accessible',
        'Verify firewall settings allow incoming connections',
        'Ensure the camera and server are on the same network',
        'Try restarting the camera'
      ]
    case 'camera_offline':
      return [
        'Check camera power connection',
        'Verify Wi-Fi connectivity',
        'Try power cycling the camera',
        'Check the Nanit app for camera status'
      ]
    case 'transcoding':
      return [
        'Try stopping and restarting the stream',
        'Check server disk space and resources',
        'Camera may need to be restarted'
      ]
    case 'connection_limit':
      return [
        'Close the official Nanit app on your phone/tablet',
        'Force-close the Nanit app (don\'t just minimize it)',
        'Wait 30-60 seconds after closing the app, then try streaming again',
        'Make sure no one else is using the Nanit app on another device',
        'If the issue persists, try restarting your phone/tablet'
      ]
    default:
      return [
        'Try refreshing the page',
        'Restart the streaming service',
        'Check server logs for more details'
      ]
  }
}

export default function StreamErrorAlert({ error, baby_uid, onRetry, className = '' }: StreamErrorAlertProps) {
  if (!error) return null

  const title = getErrorTitle(error.type)
  const explanation = getErrorExplanation(error.type, error.message)
  const solutions = getErrorSolutions(error.type)

  return (
    <div className={`bg-red-50 border border-red-200 rounded-lg p-4 ${className}`}>
      <div className="flex items-start gap-3">
        <span className="flex-shrink-0 text-red-500 text-lg">❌</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2 mb-2">
            <h4 className="font-semibold text-red-800">{title}</h4>
            {onRetry && (
              <button
                onClick={onRetry}
                className="px-3 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500"
              >
                Retry
              </button>
            )}
          </div>
          
          <p className="text-red-700 text-sm mb-3">
            {explanation}
          </p>
          
          <details className="text-sm">
            <summary className="cursor-pointer text-red-600 hover:text-red-800 font-medium mb-2">
              Troubleshooting Steps
            </summary>
            <ul className="list-disc list-inside space-y-1 text-red-700 pl-2">
              {solutions.map((solution, index) => (
                <li key={index}>{solution}</li>
              ))}
            </ul>
          </details>
          
          <div className="mt-3 text-xs text-red-600 font-mono bg-red-100 rounded p-2">
            Error: {error.type} - {error.message}
          </div>
        </div>
      </div>
    </div>
  )
}