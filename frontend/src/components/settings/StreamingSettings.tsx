'use client';

import { useState } from 'react';
import StreamingLinks from '@/components/baby/StreamingLinks';
import type { Baby } from '@/types/api';

interface StreamingSettingsProps {
  babies: Baby[];
}

export default function StreamingSettings({ babies }: StreamingSettingsProps) {
  const [activeTab, setActiveTab] = useState(0);

  if (babies.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        No devices found. Please ensure your Nanit account is authenticated and devices are connected.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Global Streaming Information */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h4 className="font-medium text-blue-900 mb-2">📡 Streaming Overview</h4>
        <div className="text-sm text-blue-800 space-y-1">
          <p>• <strong>RTMP:</strong> Best for Home Assistant, OBS, VLC, and other video software</p>
          <p>• <strong>HLS:</strong> Best for web browsers and modern mobile applications</p>
          <p>• Start video streaming on the device before using these URLs</p>
          <p>• Streaming quality depends on your network connection and device settings</p>
        </div>
      </div>

      {/* Device Tabs */}
      {babies.length > 1 && (
        <div className="border-b border-gray-200">
          <nav className="-mb-px flex space-x-8">
            {babies.map((baby, index) => (
              <button
                key={baby.uid}
                onClick={() => setActiveTab(index)}
                className={`py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === index
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                {baby.name}
                {baby.stream_state === 'streaming' && (
                  <span className="ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                    Live
                  </span>
                )}
              </button>
            ))}
          </nav>
        </div>
      )}

      {/* Streaming Links Content */}
      <div className="space-y-4">
        {babies.length === 1 ? (
          <div>
            <div className="flex items-center justify-between mb-4">
              <h4 className="text-lg font-medium text-gray-900">{babies[0].name}</h4>
              <div className="flex items-center gap-2">
                <div className={`w-2 h-2 rounded-full ${
                  babies[0].stream_state === 'streaming' ? 'bg-green-500' : 'bg-gray-400'
                }`} />
                <span className="text-sm text-gray-600">
                  {babies[0].stream_state === 'streaming' ? 'Streaming' : 'Not streaming'}
                </span>
              </div>
            </div>
            <StreamingLinks baby={babies[0]} />
          </div>
        ) : (
          <div>
            <div className="flex items-center justify-between mb-4">
              <h4 className="text-lg font-medium text-gray-900">{babies[activeTab]?.name}</h4>
              <div className="flex items-center gap-2">
                <div className={`w-2 h-2 rounded-full ${
                  babies[activeTab]?.stream_state === 'streaming' ? 'bg-green-500' : 'bg-gray-400'
                }`} />
                <span className="text-sm text-gray-600">
                  {babies[activeTab]?.stream_state === 'streaming' ? 'Streaming' : 'Not streaming'}
                </span>
              </div>
            </div>
            <StreamingLinks baby={babies[activeTab]} />
          </div>
        )}
      </div>

      {/* Streaming Status Summary for Multiple Devices */}
      {babies.length > 1 && (
        <div className="mt-8 bg-gray-50 rounded-lg p-4">
          <h4 className="text-sm font-medium text-gray-900 mb-3">Streaming Status</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {babies.map((baby) => (
              <div key={baby.uid} className="bg-white rounded-lg p-3 border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="font-medium text-gray-900">{baby.name}</p>
                    <p className="text-xs text-gray-500">
                      {baby.websocket_alive ? 'Connected' : 'Disconnected'}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${
                      baby.stream_state === 'streaming' ? 'bg-green-500' : 'bg-gray-400'
                    }`} />
                    <span className="text-xs text-gray-600">
                      {baby.stream_state === 'streaming' ? 'Live' : 'Offline'}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Home Assistant Integration Help */}
      <div className="bg-green-50 border border-green-200 rounded-lg p-4">
        <h4 className="font-medium text-green-900 mb-2">🏠 Home Assistant Integration</h4>
        <div className="text-sm text-green-800 space-y-2">
          <p>To add these streams to Home Assistant, use the RTMP URLs in your camera configuration:</p>
          <div className="bg-green-100 p-3 rounded font-mono text-xs overflow-x-auto">
            <div>camera:</div>
            <div>&nbsp;&nbsp;- platform: ffmpeg</div>
            <div>&nbsp;&nbsp;&nbsp;&nbsp;name: &quot;Nanit Camera&quot;</div>
            <div>&nbsp;&nbsp;&nbsp;&nbsp;input: &quot;rtmp://YOUR_SERVER_IP:1935/camera_uid&quot;</div>
          </div>
          <p className="text-xs text-green-700">
            Replace YOUR_SERVER_IP with the IP address of this server and camera_uid with your device&apos;s UID.
          </p>
        </div>
      </div>
    </div>
  );
}