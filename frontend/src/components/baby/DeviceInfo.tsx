'use client'

import { useState } from 'react'
import useSWR from 'swr'
import { api } from '@/lib/api'
import type { Baby, DeviceInfoResponse } from '@/types/api'
import LoadingSpinner from '@/components/ui/LoadingSpinner'

interface DeviceInfoProps {
  baby: Baby
}

export default function DeviceInfo({ baby }: DeviceInfoProps) {
  const [isExpanded, setIsExpanded] = useState(false)

  const { data, error, isLoading } = useSWR<DeviceInfoResponse>(
    isExpanded ? `/device-info/${baby.uid}` : null,
    () => api.getDeviceInfo(baby.uid),
    {
      revalidateOnFocus: false,
    }
  )

  // Debug logging
  if (data) {
    console.log('🔍 DeviceInfo API Response:', data)
    console.log('🔍 device_info object:', data.device_info)
    console.log('🔍 connection_status object:', data.connection_status)
  }

  const alerts = data?.alerts || []
  const hasAlerts = alerts.length > 0

  return (
    <div className="border border-nanit-gray-200 rounded-lg overflow-hidden">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full px-4 py-3 bg-nanit-gray-50 hover:bg-nanit-gray-100 transition-colors duration-200 flex items-center justify-between text-left"
      >
        <h3 className="font-semibold text-nanit-gray-800 flex items-center gap-2">
          🔧 Device Details & Troubleshooting
          {hasAlerts && (
            <span className="bg-red-500 text-white text-xs px-2 py-1 rounded-full">
              {alerts.length}
            </span>
          )}
        </h3>
        <span className={`transform transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}>
          ▶
        </span>
      </button>
      
      {isExpanded && (
        <div className="p-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <LoadingSpinner size="md" />
            </div>
          ) : error ? (
            <div className="text-center py-8 text-red-600">
              ❌ Failed to load device information
              <button
                onClick={() => setIsExpanded(false)}
                className="block mx-auto mt-2 text-sm underline hover:no-underline"
              >
                Retry
              </button>
            </div>
          ) : data ? (
            <div className="space-y-6">
              {/* Status Summary */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4 p-4 bg-nanit-gray-50 rounded-lg">
                <div className="text-center">
                  <div className="text-sm text-nanit-gray-600 mb-1">Firmware</div>
                  <div className="font-semibold text-nanit-gray-800">
                    {data.device_info?.firmware_version || '--'}
                  </div>
                </div>
                <div className="text-center">
                  <div className="text-sm text-nanit-gray-600 mb-1">Connection</div>
                  <div className="font-semibold text-nanit-gray-800">
                    {data.connection_status?.websocket_alive ? 'Connected' : 'Disconnected'}
                  </div>
                </div>
                <div className="text-center">
                  <div className="text-sm text-nanit-gray-600 mb-1">Streaming</div>
                  <div className="font-semibold text-nanit-gray-800">
                    {data.device_info?.streaming_error ? 'Error' : 'Ready'}
                  </div>
                </div>
                <div className="text-center">
                  <div className="text-sm text-nanit-gray-600 mb-1">Last Updated</div>
                  <div className="font-semibold text-nanit-gray-800">
                    {data.device_info?.last_updated 
                      ? new Date(data.device_info.last_updated * 1000).toLocaleString()
                      : '--'
                    }
                  </div>
                </div>
              </div>

              {/* Alerts & Troubleshooting */}
              {hasAlerts && (
                <div className="space-y-3">
                  <h4 className="font-semibold text-nanit-gray-800">⚠️ Issues & Solutions</h4>
                  {alerts.map((alert, index) => (
                    <div
                      key={index}
                      className={`p-4 rounded-lg border-l-4 ${
                        alert.type === 'error'
                          ? 'bg-red-50 border-l-red-500'
                          : 'bg-yellow-50 border-l-yellow-500'
                      }`}
                    >
                      <div className="flex items-start gap-3">
                        <span className="flex-shrink-0 text-lg">
                          {alert.type === 'error' ? '❌' : '⚠️'}
                        </span>
                        <div className="flex-1">
                          <div className={`font-medium mb-1 ${
                            alert.type === 'error' ? 'text-red-800' : 'text-yellow-800'
                          }`}>
                            {alert.message}
                          </div>
                          <div className={`text-sm mb-2 ${
                            alert.type === 'error' ? 'text-red-600' : 'text-yellow-600'
                          }`}>
                            Category: {alert.category}
                          </div>
                          
                          {/* Troubleshooting steps based on category */}
                          {alert.category === 'connection_limit' && (
                            <details className="text-sm">
                              <summary className={`cursor-pointer font-medium mb-2 ${
                                alert.type === 'error' ? 'text-red-700' : 'text-yellow-700'
                              }`}>
                                💡 How to fix this
                              </summary>
                              <ul className={`list-disc list-inside space-y-1 ml-2 ${
                                alert.type === 'error' ? 'text-red-600' : 'text-yellow-600'
                              }`}>
                                <li>Close the official Nanit app on your phone/tablet</li>
                                <li>Force-close the app (don&apos;t just minimize it)</li>
                                <li>Wait 30-60 seconds, then try streaming again</li>
                                <li>Make sure no one else is using the Nanit app</li>
                              </ul>
                            </details>
                          )}
                          
                          {alert.category === 'connectivity' && (
                            <details className="text-sm">
                              <summary className={`cursor-pointer font-medium mb-2 ${
                                alert.type === 'error' ? 'text-red-700' : 'text-yellow-700'
                              }`}>
                                💡 How to fix this
                              </summary>
                              <ul className={`list-disc list-inside space-y-1 ml-2 ${
                                alert.type === 'error' ? 'text-red-600' : 'text-yellow-600'
                              }`}>
                                <li>Check camera power connection</li>
                                <li>Verify WiFi connectivity on camera</li>
                                <li>Try power cycling the camera (unplug for 10 seconds)</li>
                                <li>Check the official Nanit app for camera status</li>
                              </ul>
                            </details>
                          )}
                          
                          {alert.category === 'streaming' && (
                            <details className="text-sm">
                              <summary className={`cursor-pointer font-medium mb-2 ${
                                alert.type === 'error' ? 'text-red-700' : 'text-yellow-700'
                              }`}>
                                💡 How to fix this
                              </summary>
                              <ul className={`list-disc list-inside space-y-1 ml-2 ${
                                alert.type === 'error' ? 'text-red-600' : 'text-yellow-600'
                              }`}>
                                <li>Try stopping and restarting the stream</li>
                                <li>Check server resources and disk space</li>
                                <li>Verify RTMP server port (1940) is accessible</li>
                                <li>Camera may need to be restarted</li>
                              </ul>
                            </details>
                          )}
                          
                          {alert.category === 'firmware' && (
                            <details className="text-sm">
                              <summary className={`cursor-pointer font-medium mb-2 ${
                                alert.type === 'error' ? 'text-red-700' : 'text-yellow-700'
                              }`}>
                                💡 How to fix this
                              </summary>
                              <ul className={`list-disc list-inside space-y-1 ml-2 ${
                                alert.type === 'error' ? 'text-red-600' : 'text-yellow-600'
                              }`}>
                                <li>Use the official Nanit app to install the update</li>
                                <li>Ensure camera is connected to WiFi during update</li>
                                <li>Do not power off camera during firmware update</li>
                                <li>Update may take 10-15 minutes to complete</li>
                              </ul>
                            </details>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Device Details Grid */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {/* Device Status */}
                <div className="bg-white border border-nanit-gray-200 rounded-lg p-4">
                  <h4 className="font-semibold text-nanit-gray-800 mb-3 border-b border-nanit-gray-200 pb-2">
                    📱 Device Status
                  </h4>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-nanit-gray-600">Hardware:</span>
                      <span className="font-medium">{data.device_info?.hardware_version || '--'}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-nanit-gray-600">Mode:</span>
                      <span className="font-medium">{data.device_info?.device_mode || '--'}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-nanit-gray-600">Volume:</span>
                      <span className="font-medium">
                        {data.device_info?.volume !== undefined ? `${data.device_info?.volume}%` : '--'}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Network Status */}
                <div className="bg-white border border-nanit-gray-200 rounded-lg p-4">
                  <h4 className="font-semibold text-nanit-gray-800 mb-3 border-b border-nanit-gray-200 pb-2">
                    📡 Network
                  </h4>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-nanit-gray-600">WiFi:</span>
                      <span className="font-medium">{data.device_info?.wifi_network || '--'}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-nanit-gray-600">Band:</span>
                      <span className="font-medium">{data.device_info?.wifi_band || '--'}</span>
                    </div>
                  </div>
                </div>

                {/* Stream Config */}
                <div className="bg-white border border-nanit-gray-200 rounded-lg p-4">
                  <h4 className="font-semibold text-nanit-gray-800 mb-3 border-b border-nanit-gray-200 pb-2">
                    📹 Stream Config
                  </h4>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-nanit-gray-600">Mobile:</span>
                      <span className="font-medium">
                        {data.device_info?.mobile_bitrate && data.device_info?.mobile_fps
                          ? `${Math.round(data.device_info?.mobile_bitrate / 1024)}KB/s @ ${data.device_info?.mobile_fps}fps`
                          : '--'}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-nanit-gray-600">DVR:</span>
                      <span className="font-medium">
                        {data.device_info?.dvr_bitrate && data.device_info?.dvr_fps
                          ? `${Math.round(data.device_info?.dvr_bitrate / 1024)}KB/s @ ${data.device_info?.dvr_fps}fps`
                          : '--'}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ) : null}
        </div>
      )}
    </div>
  )
}