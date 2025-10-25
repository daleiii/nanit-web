'use client';

import { useState } from 'react';
import DeviceInfo from '@/components/baby/DeviceInfo';
import type { Baby } from '@/types/api';

interface DeviceSettingsProps {
  babies: Baby[];
}

export default function DeviceSettings({ babies }: DeviceSettingsProps) {
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
              </button>
            ))}
          </nav>
        </div>
      )}

      {/* Device Content */}
      <div className="space-y-4">
        {babies.length === 1 ? (
          <div>
            <h4 className="text-lg font-medium text-gray-900 mb-4">{babies[0].name}</h4>
            <DeviceInfo baby={babies[0]} />
          </div>
        ) : (
          <div>
            <h4 className="text-lg font-medium text-gray-900 mb-4">{babies[activeTab]?.name}</h4>
            <DeviceInfo baby={babies[activeTab]} />
          </div>
        )}
      </div>

      {/* Device Summary for Multiple Devices */}
      {babies.length > 1 && (
        <div className="mt-8 bg-gray-50 rounded-lg p-4">
          <h4 className="text-sm font-medium text-gray-900 mb-3">Device Summary</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {babies.map((baby) => (
              <div key={baby.uid} className="bg-white rounded-lg p-3 border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="font-medium text-gray-900">{baby.name}</p>
                    <p className="text-xs text-gray-500">{baby.uid}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${
                      baby.websocket_alive ? 'bg-green-500' : 'bg-red-500'
                    }`} />
                    <span className="text-xs text-gray-600">
                      {baby.websocket_alive ? 'Online' : 'Offline'}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}