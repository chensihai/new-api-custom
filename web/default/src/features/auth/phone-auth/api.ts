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
import { api } from '@/lib/api'
import type {
  PhoneAuthEnabledResponse,
  PhoneAuthStatusResponse,
  PhoneSmsSendPayload,
  PhoneSmsLoginPayload,
  PhoneRegisterPayload,
  PhoneOneClickLoginPayload,
  PhoneBindPayload,
  PhoneRebindPayload,
} from './types'
import type { ApiResponse } from '../types'

// ============================================================================
// Phone Auth APIs
// ============================================================================

export async function getPhoneAuthEnabled(): Promise<PhoneAuthEnabledResponse> {
  const res = await api.get<{ success: boolean; data: PhoneAuthEnabledResponse }>('/api/phone-auth/enabled')
  return res.data.data
}

export async function getPhoneAuthStatus(): Promise<PhoneAuthStatusResponse> {
  const res = await api.get<{ success: boolean; data: PhoneAuthStatusResponse }>('/api/phone-auth/status')
  return res.data.data
}

export async function sendPhoneSmsCode(
  payload: PhoneSmsSendPayload
): Promise<ApiResponse> {
  const res = await api.post<ApiResponse>('/api/phone-auth/sms/send', payload)
  return res.data
}

export async function phoneSmsLogin(
  payload: PhoneSmsLoginPayload
): Promise<ApiResponse> {
  const res = await api.post<ApiResponse>('/api/phone-auth/sms/login', payload)
  return res.data
}

export async function phoneRegister(
  payload: PhoneRegisterPayload
): Promise<ApiResponse> {
  const res = await api.post<ApiResponse>('/api/phone-auth/register', payload)
  return res.data
}

export async function phoneOneClickLogin(
  payload: PhoneOneClickLoginPayload
): Promise<ApiResponse> {
  const res = await api.post<ApiResponse>('/api/phone-auth/one-click/login', payload)
  return res.data
}

export async function bindPhone(
  payload: PhoneBindPayload
): Promise<ApiResponse> {
  const res = await api.post<ApiResponse>('/api/phone-auth/bind', payload)
  return res.data
}

export async function rebindPhone(
  payload: PhoneRebindPayload
): Promise<ApiResponse> {
  const res = await api.post<ApiResponse>('/api/phone-auth/rebind', payload)
  return res.data
}

export async function unbindPhone(): Promise<ApiResponse> {
  const res = await api.post<ApiResponse>('/api/phone-auth/unbind')
  return res.data
}
