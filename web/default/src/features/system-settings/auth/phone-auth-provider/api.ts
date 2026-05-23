import { api } from '@/lib/api'
import type {
  PhoneAuthProvidersResponse,
  PhoneAuthProvidersUpdatePayload,
} from './types'

export async function getPhoneAuthProviders(): Promise<PhoneAuthProvidersResponse> {
  const res = await api.get<PhoneAuthProvidersResponse>(
    '/api/phone-auth/admin/providers'
  )
  return res.data
}

export async function updatePhoneAuthProviders(
  payload: PhoneAuthProvidersUpdatePayload
): Promise<PhoneAuthProvidersResponse> {
  const res = await api.put<PhoneAuthProvidersResponse>(
    '/api/phone-auth/admin/providers',
    payload
  )
  return res.data
}
