import useSWR from 'swr'
import { api } from '@/lib/api'
import { getTimeRange } from '@/lib/utils'
import type { 
  SensorDataResponse, 
  HistorySummary,
  DayNightAnalytics 
} from '@/types/api'

export function useSensorData(babyUid: string, timeframe: string) {
  const { start, end } = getTimeRange(timeframe)
  
  const { data, error, isLoading, mutate } = useSWR<SensorDataResponse>(
    [`/history/sensor/${babyUid}`, timeframe],
    () => api.getSensorData(babyUid, start, end, 500),
    {
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    }
  )

  return {
    sensorData: data?.readings ?? [],
    isLoading,
    isError: !!error,
    error,
    refresh: mutate,
  }
}


export function useHistorySummary(babyUid: string, timeframe: string) {
  const { start, end } = getTimeRange(timeframe)
  
  const { data, error, isLoading, mutate } = useSWR<HistorySummary>(
    [`/history/summary/${babyUid}`, timeframe],
    () => api.getHistorySummary(babyUid, start, end),
    {
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    }
  )

  return {
    summary: data,
    isLoading,
    isError: !!error,
    error,
    refresh: mutate,
  }
}

export function useDayNightAnalytics(babyUid: string, timeframe: string) {
  const { start, end } = getTimeRange(timeframe)
  
  const { data, error, isLoading, mutate } = useSWR<DayNightAnalytics>(
    [`/history/day-night/${babyUid}`, timeframe],
    () => api.getDayNightAnalytics(babyUid, start, end),
    {
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    }
  )

  return {
    analytics: data,
    isLoading,
    isError: !!error,
    error,
    refresh: mutate,
  }
}

export function useHistoricalData(babyUid: string, timeframe: string) {
  const sensorData = useSensorData(babyUid, timeframe)
  const summary = useHistorySummary(babyUid, timeframe)
  const dayNightAnalytics = useDayNightAnalytics(babyUid, timeframe)

  const refreshAll = () => {
    sensorData.refresh()
    summary.refresh()
    dayNightAnalytics.refresh()
  }

  return {
    ...sensorData,
    summary: summary.summary,
    analytics: dayNightAnalytics.analytics,
    isLoading: sensorData.isLoading || summary.isLoading || dayNightAnalytics.isLoading,
    isError: sensorData.isError || summary.isError || dayNightAnalytics.isError,
    refreshAll,
  }
}