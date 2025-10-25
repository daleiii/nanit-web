// Global tooltip configuration for react-tooltip
export const tooltipConfig = {
  // Default styling that matches the app design
  place: 'top' as const,
  style: {
    backgroundColor: 'rgba(0, 0, 0, 0.9)',
    color: 'white',
    borderRadius: '8px',
    padding: '8px 12px',
    fontSize: '14px',
    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
    zIndex: 50,
  },
  // Border as separate prop
  border: '1px solid rgba(255, 255, 255, 0.2)',
  // Animation settings
  delayShow: 500,
  delayHide: 0,
  // Responsive positioning
  offset: 8,
}

// Specialized configs for different tooltip types
export const sensorTooltipConfig = {
  ...tooltipConfig,
  place: 'bottom' as const,
}

export const timelineTooltipConfig = {
  ...tooltipConfig,
  place: 'top' as const,
  style: {
    ...tooltipConfig.style,
    minWidth: '180px',
    textAlign: 'left' as const,
  },
}

export const errorTooltipConfig = {
  ...tooltipConfig,
  place: 'bottom' as const,
  style: {
    ...tooltipConfig.style,
    backgroundColor: 'rgba(239, 68, 68, 0.9)', // Red background for errors
    maxWidth: '300px',
  },
}