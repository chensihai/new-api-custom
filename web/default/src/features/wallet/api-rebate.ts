import { api } from '@/lib/api'

export async function getRebateVisibility(): Promise<{ visible: boolean }> {
  const res = await api.get('/api/user/rebate/visibility')
  return res.data?.data ?? { visible: false }
}

export async function getRebateSummary(): Promise<RebateSummary> {
  const res = await api.get('/api/user/rebate/summary')
  return res.data?.data ?? {
    pending_quota: 0,
    settled_quota: 0,
    frozen_quota: 0,
    deficit_quota: 0,
    total_settled_quota: 0,
    total_transferred_quota: 0,
    transferable_quota: 0,
  }
}

export async function getRebateRecords(
  status: string,
  page: number,
  pageSize: number,
): Promise<RebateRecordsResponse> {
  const res = await api.get(
    `/api/user/rebate/records?status=${status}&page=${page}&page_size=${pageSize}`,
  )
  return res.data ?? { data: [], total: 0, page, page_size: pageSize }
}

export async function getRebateDeficits(
  page: number,
  pageSize: number,
): Promise<RebateDeficitsResponse> {
  const res = await api.get(
    `/api/user/rebate/deficits?page=${page}&page_size=${pageSize}`,
  )
  return res.data ?? { data: [], total: 0, page, page_size: pageSize }
}

export async function getRebateInviteeProgress(
  page: number,
  pageSize: number,
): Promise<RebateInviteeProgressResponse> {
  const res = await api.get(
    `/api/user/rebate/invitee-progress?page=${page}&page_size=${pageSize}`,
  )
  return res.data ?? { data: [], total: 0, page, page_size: pageSize }
}

export async function postRebateTransfer(quota: number): Promise<{ success: boolean; message?: string }> {
  const res = await api.post('/api/user/rebate/transfer', { quota })
  return res.data
}

export async function getRebateSettings(): Promise<RebateSettings> {
  const res = await api.get('/api/user/rebate/settings')
  return res.data?.data ?? {
    rebate_enabled: false,
    settlement_period: 7,
    rebate_visible_groups: '',
  }
}

export async function putRebateSettings(
  data: Partial<RebateSettings>,
): Promise<{ success: boolean; message?: string }> {
  const res = await api.put('/api/user/rebate/settings', data)
  return res.data
}

export async function getRebateStatistics(): Promise<RebateStatistics> {
  const res = await api.get('/api/user/rebate/statistics')
  return res.data?.data ?? {
    TotalPendingQuota: 0,
    TotalSettledQuota: 0,
    TotalFrozenQuota: 0,
    TotalDeficitQuota: 0,
    TotalTransferredQuota: 0,
    TotalRecords: 0,
    ActiveDeficits: 0,
  }
}

export interface RebateSummary {
  pending_quota: number
  settled_quota: number
  frozen_quota: number
  deficit_quota: number
  total_settled_quota: number
  total_transferred_quota: number
  transferable_quota: number
}

export interface RebateRecord {
  id: number
  invitee_name: string
  recharge_quota: number
  rebate_quota: number
  rebate_rate: number
  status: string
  expected_settle_at: number
  capped_reason: string
  created_at: number
}

export interface RebateDeficit {
  id: number
  rebate_record_id: number
  refund_log_id: number
  total_deficit_quota: number
  remaining_deficit_quota: number
  status: string
  created_at: number
}

export interface RebateInviteeProgress {
  invitee_name: string
  total_rebate_quota: number
}

export interface RebateSettings {
  rebate_enabled: boolean
  settlement_period: number
  rebate_visible_groups: string
}

export interface RebateStatistics {
  TotalPendingQuota: number
  TotalSettledQuota: number
  TotalFrozenQuota: number
  TotalDeficitQuota: number
  TotalTransferredQuota: number
  TotalRecords: number
  ActiveDeficits: number
}

export interface RebateRecordsResponse {
  data: RebateRecord[]
  total: number
  page: number
  page_size: number
}

export interface RebateDeficitsResponse {
  data: RebateDeficit[]
  total: number
  page: number
  page_size: number
}

export interface RebateInviteeProgressResponse {
  data: RebateInviteeProgress[]
  total: number
  page: number
  page_size: number
}
