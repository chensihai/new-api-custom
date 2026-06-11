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
import { z } from 'zod'

export const PHONE_REGEX = /^1[3-9]\d{9}$/
export const PHONE_MAX_LENGTH = 11
export const PHONE_CODE_LENGTH = 6
export const PHONE_CODE_REGEX = /^\d{6}$/
export const SMS_COUNTDOWN_SECONDS = 60

export const phoneFieldSchema = z
  .string()
  .min(1, 'Please enter your phone number')
  .regex(PHONE_REGEX, 'Please enter a valid phone number')

export const phoneCodeSchema = z
  .string()
  .min(1, 'Please enter the verification code')
  .regex(PHONE_CODE_REGEX, 'Verification code must be 6 digits')

export const phoneSmsLoginFormSchema = z.object({
  phone: phoneFieldSchema,
  code: phoneCodeSchema,
})

export const phoneRegisterFormSchema = z.object({
  phone: phoneFieldSchema,
  code: phoneCodeSchema,
  username: z.string().min(1, 'Please enter your username'),
  password: z.string().optional(),
})

export const phoneBindFormSchema = z.object({
  phone: phoneFieldSchema,
  code: phoneCodeSchema,
})

export const phoneRebindFormSchema = z.object({
  new_phone: phoneFieldSchema,
  code: phoneCodeSchema,
})

export function maskPhone(phone: string): string {
  if (phone.length < 7) return phone
  if (phone.length === 11) return phone.slice(0, 3) + '****' + phone.slice(7)
  return phone.slice(0, 3) + '****' + phone.slice(-2)
}

export function filterDigits(value: string, maxLength: number): string {
  return value.replace(/\D/g, '').slice(0, maxLength)
}
