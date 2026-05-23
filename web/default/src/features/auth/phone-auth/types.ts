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
export interface PhoneAuthEnabledResponse {
  enabled: boolean
  sms_available: boolean
  one_click_available: boolean
  allow_phone_register: boolean
  force_real_name_auth: boolean
}

export interface PhoneAuthStatusResponse {
  phone_bound: boolean
  masked_phone?: string
}

export interface PhoneSmsSendPayload {
  phone: string
  turnstile_token?: string
}

export interface PhoneSmsLoginPayload {
  phone: string
  code: string
}

export interface PhoneRegisterPayload {
  phone: string
  code: string
  username: string
  password?: string
}

export interface PhoneOneClickLoginPayload {
  provider: string
  token: string
}

export interface PhoneBindPayload {
  phone: string
  code: string
}

export interface PhoneRebindPayload {
  new_phone: string
  code: string
}
