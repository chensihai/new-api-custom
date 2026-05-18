import { useTranslation } from 'react-i18next'
import { QRCodeSVG } from 'qrcode.react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Loader2 } from 'lucide-react'
import { useOrderPolling } from '../hooks/use-order-polling'

interface PaymentPollingDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  tradeNo: string | null
  paymentType?: string
  qrCodeUrl?: string
  onSuccess: () => void
}

export function PaymentPollingDialog({
  open,
  onOpenChange,
  tradeNo,
  paymentType,
  qrCodeUrl,
  onSuccess,
}: PaymentPollingDialogProps) {
  const { t } = useTranslation()
  const { status, polling } = useOrderPolling(tradeNo, 2000)

  const handleClose = () => {
    onOpenChange(false)
  }

  if (status === 'success') {
    onSuccess()
    onOpenChange(false)
    return null
  }

  const isWechatNative = paymentType === 'wechat_native'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>{t('Payment Confirmation')}</DialogTitle>
          <DialogDescription>
            {polling
              ? t('Waiting for payment result...')
              : status === 'expired'
                ? t('Order has expired')
                : status === 'failed'
                  ? t('Payment failed')
                  : status === 'timeout'
                    ? t('Payment result pending, please refresh later')
                    : ''}
          </DialogDescription>
        </DialogHeader>
        <div className='flex flex-col items-center gap-4 py-4'>
          {isWechatNative && qrCodeUrl && (
            <div className='rounded-lg border bg-white p-4'>
              <QRCodeSVG value={qrCodeUrl} size={256} level='M' />
            </div>
          )}
          {isWechatNative && (
            <p className='text-sm text-muted-foreground'>
              {t('Please scan QR code with WeChat to pay')}
            </p>
          )}
          {!isWechatNative && (
            <p className='text-sm text-muted-foreground'>
              {t('Payment page has been opened, please complete payment')}
            </p>
          )}
          {polling && (
            <div className='flex items-center gap-2 text-sm text-muted-foreground'>
              <Loader2 className='h-4 w-4 animate-spin' />
              {t('Waiting for payment result...')}
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant='outline' onClick={handleClose}>
            {t('Close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
