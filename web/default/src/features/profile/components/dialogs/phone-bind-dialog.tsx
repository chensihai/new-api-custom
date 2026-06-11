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
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCountdown } from '@/hooks/use-countdown'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  sendPhoneSmsCode,
  bindPhone,
  rebindPhone,
} from '@/features/auth/phone-auth/api'
import {
  PHONE_MAX_LENGTH,
  PHONE_CODE_LENGTH,
  filterDigits,
} from '@/features/auth/phone-auth/constants'

interface PhoneBindDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  isBound: boolean
  maskedPhone?: string
  onSuccess: () => void
}

export function PhoneBindDialog({
  open,
  onOpenChange,
  isBound,
  maskedPhone,
  onSuccess,
}: PhoneBindDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [sendingCode, setSendingCode] = useState(false)
  const [phone, setPhone] = useState('')
  const [code, setCode] = useState('')
  const {
    secondsLeft,
    isActive,
    start: startCountdown,
    reset: resetCountdown,
  } = useCountdown({
    initialSeconds: 60,
  })

  const handleSendCode = async () => {
    if (!phone || phone.length < PHONE_MAX_LENGTH) {
      toast.error(t('Please enter a valid phone number'))
      return
    }

    try {
      setSendingCode(true)
      const response = await sendPhoneSmsCode({ phone })

      if (response.success) {
        toast.success(t('Verification code sent'))
        startCountdown()
      } else {
        toast.error(response.message || t('Failed to send verification code'))
      }
    } catch (_error) {
      toast.error(t('Failed to send verification code'))
    } finally {
      setSendingCode(false)
    }
  }

  const handleBind = async () => {
    if (!phone || phone.length !== PHONE_MAX_LENGTH) {
      toast.error(t('Please enter a valid phone number'))
      return
    }
    if (!code) {
      toast.error(t('Please enter verification code'))
      return
    }

    try {
      setLoading(true)
      const response = isBound
        ? await rebindPhone({ new_phone: phone, code })
        : await bindPhone({ phone, code })

      if (response.success) {
        toast.success(
          isBound ? t('Phone number changed successfully!') : t('Phone number bound successfully!')
        )
        onOpenChange(false)
        onSuccess()
        setPhone('')
        setCode('')
        resetCountdown()
      } else {
        toast.error(
          response.message || (isBound ? t('Failed to change phone number') : t('Failed to bind phone number'))
        )
      }
    } catch (_error) {
      toast.error(isBound ? t('Failed to change phone number') : t('Failed to bind phone number'))
    } finally {
      setLoading(false)
    }
  }

  const handleOpenChange = (open: boolean) => {
    if (!loading) {
      onOpenChange(open)
      if (!open) {
        setPhone('')
        setCode('')
        resetCountdown()
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>
            {isBound ? t('Change Phone Number') : t('Bind Phone Number')}
          </DialogTitle>
          <DialogDescription>
            {isBound
              ? t('Current phone: {{phone}}. Enter a new phone number to change.', {
                  phone: maskedPhone || '****',
                })
              : t('Bind a phone number to your account.')}
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-4 py-4'>
          <div className='space-y-2'>
            <Label htmlFor='phone'>{t('Phone Number')}</Label>
            <Input
              id='phone'
              type='tel'
              inputMode='numeric'
              value={phone}
              onChange={(e) => setPhone(filterDigits(e.target.value, PHONE_MAX_LENGTH))}
              placeholder={t('Enter 11-digit phone number')}
              disabled={loading}
              maxLength={PHONE_MAX_LENGTH}
            />
          </div>

          <div className='space-y-2'>
            <Label htmlFor='phone-code'>{t('Verification Code')}</Label>
            <div className='flex gap-2'>
              <Input
                id='phone-code'
                value={code}
                onChange={(e) => setCode(filterDigits(e.target.value, PHONE_CODE_LENGTH))}
                placeholder={t('Enter code')}
                disabled={loading}
                maxLength={PHONE_CODE_LENGTH}
                autoComplete='one-time-code'
              />
              <Button
                type='button'
                variant='outline'
                onClick={handleSendCode}
                disabled={sendingCode || isActive || !phone}
              >
                {isActive
                  ? `${secondsLeft}s`
                  : sendingCode
                    ? t('Sending...')
                    : t('Send')}
              </Button>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => handleOpenChange(false)}
            disabled={loading}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={handleBind}
            disabled={loading || !phone || !code}
          >
            {loading && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            {loading
              ? t('Binding...')
              : isBound
                ? t('Change Phone Number')
                : t('Bind Phone Number')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
