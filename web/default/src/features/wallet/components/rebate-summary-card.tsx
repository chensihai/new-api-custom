import { useTranslation } from 'react-i18next'
import { Coins } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { formatQuota } from '@/lib/format'
import type { RebateSummary } from '../api-rebate'

interface RebateSummaryCardProps {
  summary: RebateSummary | null
  onTransfer: () => void
  loading?: boolean
}

export function RebateSummaryCard({
  summary,
  onTransfer,
  loading,
}: RebateSummaryCardProps) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <Card className='bg-muted/20 py-0'>
        <CardContent className='grid gap-3 p-3 sm:gap-4 sm:p-4 lg:grid-cols-[minmax(200px,1fr)_minmax(180px,0.65fr)_minmax(280px,1fr)] lg:items-center'>
          <Skeleton className='h-20 w-full' />
        </CardContent>
      </Card>
    )
  }

  const canTransfer = (summary?.settled_quota ?? 0) > 0

  return (
    <Card className='bg-muted/20 py-0'>
      <CardContent className='grid gap-3 p-3 sm:gap-4 sm:p-4 lg:grid-cols-[minmax(200px,1fr)_minmax(180px,0.65fr)_minmax(280px,1fr)] lg:items-center'>
        <div className='space-y-1'>
          <div className='flex items-center gap-2'>
            <Coins className='h-5 w-5 text-amber-500' />
            <h3 className='text-base font-semibold'>{t('Rebate Rewards')}</h3>
          </div>
          <p className='text-sm text-muted-foreground'>
            {t('Earn rebate when your referrals top up')}
          </p>
        </div>

        <div className='grid grid-cols-3 gap-2 text-center'>
          <div>
            <p className='text-xs text-muted-foreground'>{t('Pending')}</p>
            <p className='text-sm font-medium'>
              {formatQuota(summary?.pending_quota ?? 0)}
            </p>
          </div>
          <div>
            <p className='text-xs text-muted-foreground'>{t('Available')}</p>
            <p className='text-sm font-medium text-green-600'>
              {formatQuota(summary?.settled_quota ?? 0)}
            </p>
          </div>
          <div>
            <p className='text-xs text-muted-foreground'>{t('Transferred')}</p>
            <p className='text-sm font-medium'>
              {formatQuota(summary?.total_transferred_quota ?? 0)}
            </p>
          </div>
        </div>

        <div className='flex flex-col gap-2'>
          {(summary?.frozen_quota ?? 0) > 0 && (
            <p className='text-xs text-amber-600'>
              {t('Frozen')}: {formatQuota(summary?.frozen_quota ?? 0)}
            </p>
          )}
          {(summary?.deficit_quota ?? 0) > 0 && (
            <p className='text-xs text-red-600'>
              {t('Deficit')}: {formatQuota(summary?.deficit_quota ?? 0)}
            </p>
          )}
          <Button
            variant='outline'
            size='sm'
            onClick={onTransfer}
            disabled={!canTransfer}
          >
            {t('Transfer Rebate')}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
