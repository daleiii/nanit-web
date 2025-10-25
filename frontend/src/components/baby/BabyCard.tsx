'use client'

import { useState } from 'react'
import type { Baby } from '@/types/api'
import BabyHeader from './BabyHeader'
import VideoSection from './VideoSection'
import SensorGrid from './SensorGrid'
import ControlPanel from './ControlPanel'
import HistoricalData from './HistoricalData'

interface BabyCardProps {
  baby: Baby
}

export default function BabyCard({ baby }: BabyCardProps) {
  return (
    <div className="card">
      <BabyHeader baby={baby} />
      
      <div className="p-6 space-y-8">
        <VideoSection baby={baby} />
        
        <SensorGrid baby={baby} />
        
        <ControlPanel baby={baby} />
        
        <HistoricalData baby={baby} />
      </div>
    </div>
  )
}