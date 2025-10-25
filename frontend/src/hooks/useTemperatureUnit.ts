import { useState, useEffect } from 'react'

export type TemperatureUnit = 'celsius' | 'fahrenheit'

export function useTemperatureUnit() {
  const [unit, setUnit] = useState<TemperatureUnit>('celsius')

  useEffect(() => {
    // Load from localStorage on mount
    const saved = localStorage.getItem('temperatureUnit') as TemperatureUnit
    if (saved && (saved === 'celsius' || saved === 'fahrenheit')) {
      setUnit(saved)
    }

    // Listen for localStorage changes from other components
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === 'temperatureUnit' && e.newValue) {
        const newUnit = e.newValue as TemperatureUnit
        if (newUnit === 'celsius' || newUnit === 'fahrenheit') {
          setUnit(newUnit)
        }
      }
    }

    // Listen for custom events (for same-tab changes)
    const handleCustomEvent = (e: CustomEvent) => {
      if (e.detail && (e.detail === 'celsius' || e.detail === 'fahrenheit')) {
        setUnit(e.detail)
      }
    }

    window.addEventListener('storage', handleStorageChange)
    window.addEventListener('temperatureUnitChanged', handleCustomEvent as EventListener)

    return () => {
      window.removeEventListener('storage', handleStorageChange)
      window.removeEventListener('temperatureUnitChanged', handleCustomEvent as EventListener)
    }
  }, [])

  const toggleUnit = () => {
    const newUnit = unit === 'celsius' ? 'fahrenheit' : 'celsius'
    setUnit(newUnit)
    localStorage.setItem('temperatureUnit', newUnit)
    
    // Dispatch custom event to notify other components immediately
    window.dispatchEvent(new CustomEvent('temperatureUnitChanged', { detail: newUnit }))
  }

  const convertTemperature = (celsius: number): number => {
    if (unit === 'fahrenheit') {
      return (celsius * 9/5) + 32
    }
    return celsius
  }

  const formatTemperature = (celsius: number | undefined): string => {
    if (celsius === undefined || celsius <= 0) {
      return `--°${unit === 'celsius' ? 'C' : 'F'}`
    }
    
    const temp = convertTemperature(celsius)
    const unitSymbol = unit === 'celsius' ? '°C' : '°F'
    return `${temp.toFixed(1)}${unitSymbol}`
  }

  return {
    unit,
    toggleUnit,
    convertTemperature,
    formatTemperature,
  }
}