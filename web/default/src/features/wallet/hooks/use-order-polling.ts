import { useState, useEffect, useCallback } from 'react'
import { getTopUpStatus } from '../api'

export function useOrderPolling(
  tradeNo: string | null,
  intervalMs = 2000,
  maxPolls = 30
) {
  const [status, setStatus] = useState<string | null>(null)
  const [polling, setPolling] = useState(false)

  const stopPolling = useCallback(() => {
    setPolling(false)
  }, [])

  useEffect(() => {
    if (!tradeNo) {
      setPolling(false)
      return
    }

    setStatus(null)
    setPolling(true)
    let pollCount = 0
    let stopped = false

    const timer = setInterval(async () => {
      if (stopped) return
      pollCount++
      try {
        const res = await getTopUpStatus(tradeNo)
        const s = res.data?.status
        if (s) {
          setStatus(s)
          if (
            s === 'success' ||
            s === 'failed' ||
            s === 'expired'
          ) {
            stopped = true
            setPolling(false)
            clearInterval(timer)
            return
          }
        }
      } catch {
        // ignore
      }
      if (pollCount >= maxPolls) {
        stopped = true
        setPolling(false)
        setStatus('timeout')
        clearInterval(timer)
      }
    }, intervalMs)

    return () => {
      stopped = true
      clearInterval(timer)
      setPolling(false)
    }
  }, [tradeNo, intervalMs, maxPolls])

  return { status, polling, stopPolling }
}
