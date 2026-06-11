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
import { useState } from 'react'
import i18next from 'i18next'
import { toast } from 'sonner'
import { useCountdown } from '@/hooks/use-countdown'
import { sendPhoneSmsCode } from '../api'
import { SMS_COUNTDOWN_SECONDS } from '../constants'

interface UseSmsCountdownOptions {
  turnstileToken?: string
  validateTurnstile?: () => boolean
}

export function useSmsCountdown(options?: UseSmsCountdownOptions) {
  const [isSending, setIsSending] = useState(false)
  const {
    secondsLeft,
    isActive: isCountdownActive,
    start: startCountdown,
  } = useCountdown({ initialSeconds: SMS_COUNTDOWN_SECONDS })

  const sendCode = async (phone: string, turnstileToken?: string) => {
    if (!phone) {
      toast.error(i18next.t('Please enter your phone number first'))
      return false
    }

    if (options?.validateTurnstile && !options.validateTurnstile()) {
      return false
    }

    setIsSending(true)
    try {
      const token = turnstileToken ?? options?.turnstileToken
      const res = await sendPhoneSmsCode({
        phone,
        turnstile_token: token || undefined,
      })
      if (res?.success) {
        startCountdown()
        toast.success(i18next.t('Verification code sent'))
        return true
      }
      toast.error(res?.message || i18next.t('Failed to send verification code'))
      return false
    } catch (_error) {
      return false
    } finally {
      setIsSending(false)
    }
  }

  return {
    isSending,
    secondsLeft,
    isCountdownActive,
    sendCode,
  }
}
