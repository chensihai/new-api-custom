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

interface AlipayQrDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  qrCode: string | null
}

export function AlipayQrDialog({
  open,
  onOpenChange,
  qrCode,
}: AlipayQrDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>{t('Alipay QR Code')}</DialogTitle>
          <DialogDescription>
            {t('Please scan the QR code with Alipay app to complete payment')}
          </DialogDescription>
        </DialogHeader>
        <div className='flex flex-col items-center gap-4 py-4'>
          {qrCode && (
            <div className='rounded-lg border bg-white p-4'>
              <QRCodeSVG value={qrCode} size={256} level='M' />
            </div>
          )}
          <p className='text-sm text-muted-foreground'>
            {t('QR code will expire in 15 minutes')}
          </p>
        </div>
        <DialogFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('Close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
