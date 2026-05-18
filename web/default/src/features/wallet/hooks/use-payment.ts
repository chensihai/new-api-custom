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
import { useState, useCallback } from 'react'
import i18next from 'i18next'
import { toast } from 'sonner'
import {
  calculateAmount,
  calculateStripeAmount,
  calculateWaffoPancakeAmount,
  calculateAlipayAmount,
  calculateWechatAmount,
  requestPayment,
  requestStripePayment,
  requestAlipayPayment,
  requestWechatPayment,
  isApiSuccess,
} from '../api'
import {
  isStripePayment,
  isWaffoPancakePayment,
  isAlipayPayment,
  isWechatPayment,
  getAlipayPaymentMethod,
  getWechatPaymentMethod,
  submitPaymentForm,
} from '../lib'

// ============================================================================
// Payment Hook
// ============================================================================

export function usePayment() {
  const [amount, setAmount] = useState<number>(0)
  const [qrCode, setQrCode] = useState<string | null>(null)
  const [calculating, setCalculating] = useState(false)
  const [processing, setProcessing] = useState(false)

  // Calculate payment amount
  const calculatePaymentAmount = useCallback(
    async (topupAmount: number, paymentType: string) => {
      try {
        setCalculating(true)

        const isStripe = isStripePayment(paymentType)
        const isPancake = isWaffoPancakePayment(paymentType)
        const isAlipay = isAlipayPayment(paymentType)
        const isWechat = isWechatPayment(paymentType)
        const response = isStripe
          ? await calculateStripeAmount({ amount: topupAmount })
          : isPancake
            ? await calculateWaffoPancakeAmount({ amount: topupAmount })
            : isAlipay
              ? await calculateAlipayAmount({ amount: topupAmount })
              : isWechat
                ? await calculateWechatAmount({ amount: topupAmount })
                : await calculateAmount({ amount: topupAmount })

        if (isApiSuccess(response) && response.data) {
          const calculatedAmount = parseFloat(response.data)
          setAmount(calculatedAmount)
          return calculatedAmount
        }

        // Don't show error for calculation, just set to 0
        setAmount(0)
        return 0
      } catch (_error) {
        setAmount(0)
        return 0
      } finally {
        setCalculating(false)
      }
    },
    []
  )

  // Process payment
  const processPayment = useCallback(
    async (topupAmount: number, paymentType: string) => {
      try {
        setProcessing(true)

        const isStripe = isStripePayment(paymentType)
        const isAlipay = isAlipayPayment(paymentType)
        const isWechat = isWechatPayment(paymentType)
        const amount = Math.floor(topupAmount)

        let response

        if (isStripe) {
          response = await requestStripePayment({
            amount,
            payment_method: 'stripe',
          })
        } else if (isAlipay) {
          response = await requestAlipayPayment({
            amount,
            payment_method: getAlipayPaymentMethod(),
          })
        } else if (isWechat) {
          response = await requestWechatPayment({
            amount,
            payment_method: getWechatPaymentMethod(),
          })
        } else {
          response = await requestPayment({
            amount,
            payment_method: paymentType,
          })
        }

        if (!isApiSuccess(response)) {
          toast.error(response.message || i18next.t('Payment request failed'))
          return false
        }

        // Handle Stripe payment
        if (isStripe && response.data?.pay_link) {
          window.open(response.data.pay_link as string, '_blank')
          toast.success(i18next.t('Redirecting to payment page...'))
          return true
        }

        // Handle Alipay payment
        if (isAlipay && response.data) {
          const data = response.data as unknown as {
            type?: string
            qr_code?: string
            url?: string
          }
          if (data.type === 'qr_code' && data.qr_code) {
            setQrCode(data.qr_code)
            toast.info(i18next.t('Please scan QR code with Alipay to pay'))
            return true
          }
          if (data.url) {
            window.open(data.url, '_blank')
            toast.success(i18next.t('Redirecting to payment page...'))
            return true
          }
          if (typeof response.data === 'string') {
            window.open(response.data, '_blank')
            toast.success(i18next.t('Redirecting to payment page...'))
            return true
          }
        }

        // Handle WeChat Pay payment
        if (isWechat && response.data) {
          const data = response.data as unknown as {
            type?: string
            code_url?: string
            h5_url?: string
          }
          if (data.type === 'native' && data.code_url) {
            toast.info(i18next.t('Please scan QR code with WeChat to pay'))
            window.open(data.code_url, '_blank')
            return true
          }
          if (data.type === 'h5' && data.h5_url) {
            window.location.href = data.h5_url
            return true
          }
          if (typeof response.data === 'string') {
            window.open(response.data, '_blank')
            toast.success(i18next.t('Redirecting to payment page...'))
            return true
          }
        }

        // Handle non-Stripe payment (epay form)
        if (!isStripe && !isAlipay && !isWechat && response.data) {
          const url = (response as unknown as { url?: string }).url
          if (url) {
            submitPaymentForm(url, response.data)
            toast.success(i18next.t('Redirecting to payment page...'))
            return true
          }
        }

        return false
      } catch (_error) {
        toast.error(i18next.t('Payment request failed'))
        return false
      } finally {
        setProcessing(false)
      }
    },
    []
  )

  const clearQrCode = useCallback(() => setQrCode(null), [])

  return {
    amount,
    qrCode,
    clearQrCode,
    calculating,
    processing,
    calculatePaymentAmount,
    processPayment,
    setAmount,
  }
}
