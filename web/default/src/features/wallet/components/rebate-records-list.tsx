import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { List } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { formatQuota, formatTimestampToDate } from '@/lib/format'
import { getRebateRecords } from '../api-rebate'
import type { RebateRecord } from '../api-rebate'

const PAGE_SIZE = 10

const STATUS_TABS = ['all', 'pending', 'settled', 'frozen'] as const
type StatusTab = (typeof STATUS_TABS)[number]

interface RebateRecordsListProps {
  visible?: boolean
}

export function RebateRecordsList({ visible = true }: RebateRecordsListProps) {
  const { t } = useTranslation()
  const [status, setStatus] = useState<StatusTab>('all')
  const [page, setPage] = useState(1)
  const [records, setRecords] = useState<RebateRecord[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)

  const fetchRecords = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getRebateRecords(
        status === 'all' ? '' : status,
        page,
        PAGE_SIZE,
      )
      setRecords(res.data)
      setTotal(res.total)
    } catch {
      setRecords([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [status, page])

  useEffect(() => {
    if (visible) {
      fetchRecords()
    }
  }, [fetchRecords, visible])

  const handleStatusChange = (newStatus: StatusTab) => {
    setStatus(newStatus)
    setPage(1)
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const statusBadgeVariant = (s: string) => {
    switch (s) {
      case 'pending':
        return 'secondary'
      case 'settled':
        return 'default'
      case 'frozen':
        return 'destructive'
      default:
        return 'outline'
    }
  }

  const statusLabel = (s: string) => {
    switch (s) {
      case 'pending':
        return t('Pending')
      case 'settled':
        return t('Settled')
      case 'frozen':
        return t('Frozen')
      default:
        return s
    }
  }

  if (!visible) return null

  return (
    <Card className='bg-muted/20 py-0'>
      <CardContent className='space-y-3 p-3 sm:space-y-4 sm:p-4'>
        <div className='flex items-center gap-2'>
          <List className='h-5 w-5 text-blue-500' />
          <h3 className='text-base font-semibold'>{t('Rebate Records')}</h3>
        </div>

        <Tabs
          value={status}
          onValueChange={(v) => handleStatusChange(v as StatusTab)}
        >
          <TabsList>
            {STATUS_TABS.map((tab) => (
              <TabsTrigger key={tab} value={tab}>
                {tab === 'all'
                  ? t('All')
                  : tab === 'pending'
                    ? t('Pending')
                    : tab === 'settled'
                      ? t('Settled')
                      : t('Frozen')}
              </TabsTrigger>
            ))}
          </TabsList>

          {STATUS_TABS.map((tab) => (
            <TabsContent key={tab} value={tab}>
              {loading ? (
                <div className='space-y-2'>
                  {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className='h-10 w-full' />
                  ))}
                </div>
              ) : records.length === 0 ? (
                <div className='py-8 text-center text-sm text-muted-foreground'>
                  {t('No rebate records found')}
                </div>
              ) : (
                <div className='space-y-3'>
                  <div className='overflow-hidden rounded-lg border'>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Invitee')}</TableHead>
                          <TableHead>{t('Recharge Amount')}</TableHead>
                          <TableHead>{t('Rebate Amount')}</TableHead>
                          <TableHead>{t('Rebate Rate')}</TableHead>
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead className='hidden md:table-cell'>
                            {t('Expected Settlement')}
                          </TableHead>
                          <TableHead className='hidden lg:table-cell'>
                            {t('Capped Reason')}
                          </TableHead>
                          <TableHead className='hidden sm:table-cell'>
                            {t('Created At')}
                          </TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {records.map((record) => (
                          <TableRow key={record.id}>
                            <TableCell className='font-medium'>
                              {record.invitee_name}
                            </TableCell>
                            <TableCell className='font-mono tabular-nums'>
                              {formatQuota(record.recharge_quota)}
                            </TableCell>
                            <TableCell className='font-mono tabular-nums'>
                              {formatQuota(record.rebate_quota)}
                            </TableCell>
                            <TableCell className='font-mono tabular-nums'>
                              {(record.rebate_rate * 100).toFixed(1)}%
                            </TableCell>
                            <TableCell>
                              <Badge
                                variant={statusBadgeVariant(record.status)}
                              >
                                {statusLabel(record.status)}
                              </Badge>
                            </TableCell>
                            <TableCell className='hidden md:table-cell'>
                              {formatTimestampToDate(
                                record.expected_settle_at,
                              )}
                            </TableCell>
                            <TableCell className='hidden max-w-[200px] truncate lg:table-cell'>
                              {record.capped_reason || '-'}
                            </TableCell>
                            <TableCell className='hidden sm:table-cell'>
                              {formatTimestampToDate(record.created_at)}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>

                  {totalPages > 1 && (
                    <div className='flex items-center justify-between'>
                      <p className='text-muted-foreground text-xs'>
                        {t('Total {{count}} records', { count: total })}
                      </p>
                      <div className='flex items-center gap-2'>
                        <Button
                          variant='outline'
                          size='sm'
                          disabled={page <= 1}
                          onClick={() => setPage((p) => Math.max(1, p - 1))}
                        >
                          {t('Previous')}
                        </Button>
                        <span className='text-muted-foreground text-xs tabular-nums'>
                          {page} / {totalPages}
                        </span>
                        <Button
                          variant='outline'
                          size='sm'
                          disabled={page >= totalPages}
                          onClick={() => setPage((p) => p + 1)}
                        >
                          {t('Next')}
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </TabsContent>
          ))}
        </Tabs>
      </CardContent>
    </Card>
  )
}
