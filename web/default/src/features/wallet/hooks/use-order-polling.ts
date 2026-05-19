import { useState, useEffect, useCallback, useRef } from 'react'
import { getTopUpStatus } from '../api'

export function useOrderPolling(
  tradeNo: string | null,
  initialIntervalMs = 5000,
  maxPolls = 60
) {
  const [status, setStatus] = useState<string | null>(null)
  const [polling, setPolling] = useState(false)
  const stoppedRef = useRef(false)
  const pollingRef = useRef(false)

  const stopPolling = useCallback(() => {
    stoppedRef.current = true
    setPolling(false)
  }, [])

  useEffect(() => {
    if (!tradeNo) {
      setPolling(false)
      return
    }

    setStatus(null)
    setPolling(true)
    stoppedRef.current = false
    pollingRef.current = false

    let pollCount = 0
    let currentInterval = initialIntervalMs
    let timeoutId: ReturnType<typeof setTimeout> | null = null

    const poll = async () => {
      if (stoppedRef.current || pollingRef.current) return

      pollingRef.current = true
      pollCount++

      try {
        const res = await getTopUpStatus(tradeNo)
        const s = res.data?.status

        if (stoppedRef.current) return

        if (s) {
          setStatus(s)
          if (s === 'success' || s === 'failed' || s === 'expired') {
            stoppedRef.current = true
            setPolling(false)
            return
          }
        }
      } catch {
        // ignore errors
      } finally {
        pollingRef.current = false
      }

      if (stoppedRef.current) return

      if (pollCount >= maxPolls) {
        stoppedRef.current = true
        setPolling(false)
        setStatus('timeout')
        return
      }

      // Exponential backoff: increase interval gradually, max 15s
      currentInterval = Math.min(currentInterval * 1.2, 15000)
      timeoutId = setTimeout(poll, currentInterval)
    }

    // Start first poll after a short delay
    timeoutId = setTimeout(poll, 1000)

    return () => {
      stoppedRef.current = true
      if (timeoutId) {
        clearTimeout(timeoutId)
      }
      setPolling(false)
    }
  }, [tradeNo, initialIntervalMs, maxPolls])

  return { status, polling, stopPolling }
}
