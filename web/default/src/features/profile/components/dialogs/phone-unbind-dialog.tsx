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
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Loader2, MessageSquare } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { unbindPhone, sendPhoneSmsCode } from '@/features/auth/phone-auth/api'
import { PHONE_CODE_LENGTH, PHONE_MAX_LENGTH, filterDigits } from '@/features/auth/phone-auth/constants'
import { useSmsCountdown } from '@/features/auth/phone-auth/hooks/use-sms-countdown'

interface PhoneUnbindDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  maskedPhone?: string
  onSuccess: () => void
}

export function PhoneUnbindDialog({
  open,
  onOpenChange,
  maskedPhone,
  onSuccess,
}: PhoneUnbindDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [phone, setPhone] = useState('')
  const [code, setCode] = useState('')
  const {
    isSending,
    secondsLeft,
    isCountdownActive,
    sendCode,
  } = useSmsCountdown({})

  const handleSendCode = async () => {
    if (!phone || isSending || isCountdownActive) return
    await sendCode(phone)
  }

  const handleConfirm = async () => {
    if (!phone) {
      toast.error(t('Please enter your phone number'))
      return
    }
    if (!code) {
      toast.error(t('Please enter the verification code'))
      return
    }
    setLoading(true)
    try {
      const res = await unbindPhone({ phone, code })
      if (res.success) {
        toast.success(t('Phone unbound successfully'))
        onOpenChange(false)
        setPhone('')
        setCode('')
        onSuccess()
      } else {
        toast.error(res.message || t('Failed to unbind phone number'))
      }
    } catch (_error) {
      toast.error(t('Failed to unbind phone number'))
    } finally {
      setLoading(false)
    }
  }

  const handleClose = (val: boolean) => {
    if (!val) {
      setPhone('')
      setCode('')
    }
    onOpenChange(val)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className='max-w-sm'>
        <DialogHeader>
          <DialogTitle>{t('Confirm Unbind Phone Number')}</DialogTitle>
          <DialogDescription>
            {t(
              'To unbind phone number {{phone}}, please enter the phone number and verification code.',
              { phone: maskedPhone || '****' }
            )}
          </DialogDescription>
        </DialogHeader>

        <Input
          placeholder={t('Enter 11-digit phone number')}
          inputMode='numeric'
          maxLength={PHONE_MAX_LENGTH}
          autoComplete='tel'
          value={phone}
          onChange={(e) => {
            const filtered = filterDigits(e.target.value, PHONE_MAX_LENGTH)
            setPhone(filtered)
          }}
        />

        <div className='flex items-end gap-2'>
          <div className='flex-1'>
            <Input
              placeholder={t('Enter 6-digit code')}
              inputMode='numeric'
              maxLength={PHONE_CODE_LENGTH}
              autoComplete='one-time-code'
              value={code}
              onChange={(e) => {
                const filtered = filterDigits(e.target.value, PHONE_CODE_LENGTH)
                setCode(filtered)
              }}
            />
          </div>
          <Button
            variant='outline'
            type='button'
            disabled={isSending || isCountdownActive || !phone}
            onClick={handleSendCode}
            className='gap-1'
          >
            {isCountdownActive ? (
              t('Resend ({{seconds}}s)', { seconds: secondsLeft })
            ) : isSending ? (
              <Loader2 className='h-4 w-4 animate-spin' />
            ) : (
              <>
                <MessageSquare className='h-4 w-4' />
                {t('Send code')}
              </>
            )}
          </Button>
        </div>

        <DialogFooter>
          <Button
            variant='outline'
            onClick={() => handleClose(false)}
            disabled={loading}
          >
            {t('Cancel')}
          </Button>
          <Button
            variant='destructive'
            onClick={handleConfirm}
            disabled={loading || !code || !phone}
          >
            {loading ? <Loader2 className='h-4 w-4 animate-spin' /> : null}
            {t('Confirm Unbind')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
