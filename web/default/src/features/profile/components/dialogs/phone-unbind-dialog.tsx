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
import { ConfirmDialog } from '@/components/confirm-dialog'
import { unbindPhone } from '@/features/auth/phone-auth/api'

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

  const handleConfirm = async () => {
    setLoading(true)
    try {
      const res = await unbindPhone()
      if (res.success) {
        toast.success(t('Phone number unbound successfully'))
        onOpenChange(false)
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

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Confirm Unbind Phone Number')}
      desc={t(
        'Are you sure you want to unbind phone number {{phone}}? You will no longer be able to log in via phone number.',
        { phone: maskedPhone || '****' }
      )}
      confirmText={t('Confirm Unbind')}
      destructive
      handleConfirm={handleConfirm}
      isLoading={loading}
    />
  )
}
