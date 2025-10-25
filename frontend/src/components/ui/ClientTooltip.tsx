'use client'

import { Tooltip } from 'react-tooltip'
import { tooltipConfig } from '@/lib/tooltipSetup'

export default function ClientTooltip() {
  return (
    <Tooltip 
      id="app-tooltip" 
      style={tooltipConfig.style}
      border={tooltipConfig.border}
      delayShow={tooltipConfig.delayShow}
      delayHide={tooltipConfig.delayHide}
      offset={tooltipConfig.offset}
    />
  )
}