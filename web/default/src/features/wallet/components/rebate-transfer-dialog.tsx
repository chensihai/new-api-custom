import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { formatQuota } from '@/lib/format'
import { postRebateTransfer } from '../api-rebate'

interface RebateTransferDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  maxQuota: number
  onSuccess: () => void
}

export function RebateTransferDialog({
  open,
  onOpenChange,
  maxQuota,
  onSuccess,
}: RebateTransferDialogProps) {
  const { t } = useTranslation()
  const [quota, setQuota] = useState<number>(0)
  const [loading, setLoading] = useState(false)

  const handleTransfer = async () => {
    if (quota <= 0 || quota > maxQuota) return
    setLoading(true)
    try {
      const res = await postRebateTransfer(quota)
      if (res.success) {
        onSuccess()
        onOpenChange(false)
        setQuota(0)
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-[400px]'>
        <DialogHeader>
          <DialogTitle>{t('Transfer Rebate')}</DialogTitle>
        </DialogHeader>
        <div className='space-y-4 py-4'>
          <p className='text-sm text-muted-foreground'>
            {t('Transfer rebate rewards to your main balance.')}
          </p>
          <p className='text-sm'>
            {t('Available')}: <span className='font-medium text-green-600'>{formatQuota(maxQuota)}</span>
          </p>
          <Input
            type='number'
            min={1}
            max={maxQuota}
            value={quota || ''}
            onChange={(e) => setQuota(parseInt(e.target.value) || 0)}
            placeholder={t('Enter amount to transfer')}
          />
        </div>
        <DialogFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button
            onClick={handleTransfer}
            disabled={loading || quota <= 0 || quota > maxQuota}
          >
            {t('Transfer')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
