/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useState } from 'react'
import { getPhoneAuthEnabled } from '../api'
import type { PhoneAuthEnabledResponse } from '../types'

const CACHE_TTL_MS = 5 * 60 * 1000

let cachedResult: PhoneAuthEnabledResponse | null = null
let cachedTimestamp: number = 0

function isCacheValid(): boolean {
  return cachedResult !== null && (Date.now() - cachedTimestamp) < CACHE_TTL_MS
}

export function usePhoneAuthEnabled() {
  const [data, setData] = useState<PhoneAuthEnabledResponse | null>(
    isCacheValid() ? cachedResult : null,
  )
  const [isLoading, setIsLoading] = useState(!isCacheValid())

  useEffect(() => {
    if (isCacheValid()) {
      setData(cachedResult)
      setIsLoading(false)
      return
    }

    setIsLoading(true)
    getPhoneAuthEnabled()
      .then((res) => {
        cachedResult = res
        cachedTimestamp = Date.now()
        setData(res)
      })
      .catch(() => {
        // Keep previous data on failure to avoid flickering
      })
      .finally(() => {
        setIsLoading(false)
      })
  }, [])

  return { data, isLoading }
}
