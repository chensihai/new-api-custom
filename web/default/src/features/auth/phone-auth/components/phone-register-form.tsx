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
import { useEffect, useState } from 'react'
import type { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Loader2, MessageSquare } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { useStatus } from '@/hooks/use-status'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { PasswordInput } from '@/components/password-input'
import { Turnstile } from '@/components/turnstile'
import { phoneRegister } from '../api'
import { LegalConsent } from '@/features/auth/components/legal-consent'
import { useAuthRedirect } from '@/features/auth/hooks/use-auth-redirect'
import { useTurnstile } from '@/features/auth/hooks/use-turnstile'
import {
  phoneRegisterFormSchema,
  filterDigits,
  PHONE_MAX_LENGTH,
  PHONE_CODE_LENGTH,
} from '../constants'
import { useSmsCountdown } from '../hooks/use-sms-countdown'

interface PhoneRegisterFormProps extends React.HTMLAttributes<HTMLFormElement> {
  onSwitchToPassword?: () => void
  agreedToLegal?: boolean
  requiresLegalConsent?: boolean
}

export function PhoneRegisterForm({
  className,
  onSwitchToPassword,
  agreedToLegal: externalAgreedToLegal,
  requiresLegalConsent: externalRequiresLegalConsent,
  ...props
}: PhoneRegisterFormProps) {
  const { t } = useTranslation()
  const [isLoading, setIsLoading] = useState(false)
  const [agreedToLegal, setAgreedToLegal] = useState(false)
  const legalConsentErrorMessage = t('Please agree to the legal terms first')

  const { status } = useStatus()
  const {
    isTurnstileEnabled,
    turnstileSiteKey,
    turnstileToken,
    setTurnstileToken,
    validateTurnstile,
  } = useTurnstile()
  const { redirectToLogin } = useAuthRedirect()
  const {
    isSending,
    secondsLeft,
    isCountdownActive,
    sendCode,
  } = useSmsCountdown({
    turnstileToken,
    validateTurnstile,
  })

  const hasUserAgreement = Boolean(status?.user_agreement_enabled)
  const hasPrivacyPolicy = Boolean(status?.privacy_policy_enabled)
  const requiresLegalConsent =
    externalRequiresLegalConsent ?? (hasUserAgreement || hasPrivacyPolicy)

  useEffect(() => {
    if (externalAgreedToLegal !== undefined) {
      setAgreedToLegal(externalAgreedToLegal)
    } else if (requiresLegalConsent) {
      setAgreedToLegal(false)
    } else {
      setAgreedToLegal(true)
    }
  }, [externalAgreedToLegal, requiresLegalConsent])

  const form = useForm<z.infer<typeof phoneRegisterFormSchema>>({
    resolver: zodResolver(phoneRegisterFormSchema),
    defaultValues: {
      phone: '',
      code: '',
      username: '',
      password: '',
    },
  })

  const phoneValue = form.watch('phone')

  async function onSubmit(data: z.infer<typeof phoneRegisterFormSchema>) {
    if (requiresLegalConsent && !agreedToLegal) {
      toast.error(legalConsentErrorMessage)
      return
    }

    if (!validateTurnstile()) return

    setIsLoading(true)
    try {
      const res = await phoneRegister({
        phone: data.phone,
        code: data.code,
        username: data.username,
        password: data.password || undefined,
      })

      if (res?.success) {
        toast.success(t('Account created! Please sign in'))
        redirectToLogin()
      }
    } catch (_error) {
      // Errors are handled by global interceptor
    } finally {
      setIsLoading(false)
    }
  }

  async function handleSendCode() {
    const phone = phoneValue
    if (!phone) {
      toast.error(t('Please enter your phone number first'))
      return
    }
    if (requiresLegalConsent && !agreedToLegal) {
      toast.error(legalConsentErrorMessage)
      return
    }
    await sendCode(phone, turnstileToken)
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className={cn('grid gap-4', className)}
        {...props}
      >
        <FormField
          control={form.control}
          name='phone'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Phone Number')}</FormLabel>
              <FormControl>
                <Input
                  placeholder={t('Enter your phone number')}
                  inputMode='numeric'
                  maxLength={PHONE_MAX_LENGTH}
                  {...field}
                  onChange={(e) => {
                    const filtered = filterDigits(e.target.value, PHONE_MAX_LENGTH)
                    field.onChange(filtered)
                  }}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <div className='flex items-end gap-2'>
          <div className='flex-1'>
            <FormField
              control={form.control}
              name='code'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Verification Code')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('Enter 6-digit code')}
                      inputMode='numeric'
                      maxLength={PHONE_CODE_LENGTH}
                      autoComplete='one-time-code'
                      {...field}
                      onChange={(e) => {
                        const filtered = filterDigits(e.target.value, PHONE_CODE_LENGTH)
                        field.onChange(filtered)
                      }}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <Button
            variant='outline'
            type='button'
            disabled={isLoading || isSending || isCountdownActive || !phoneValue}
            onClick={handleSendCode}
            className='gap-1'
          >
            {isCountdownActive ? (
              t('Resend ({{seconds}}s)', { seconds: secondsLeft })
            ) : isSending ? (
              <Loader2 className='h-4 w-4 animate-spin' />
            ) : (
              <>
                <MessageSquare className='h-4 w-4' />
                {t('Send code')}
              </>
            )}
          </Button>
        </div>

        <FormField
          control={form.control}
          name='username'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Username')}</FormLabel>
              <FormControl>
                <Input placeholder={t('Enter your username')} {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name='password'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Password (optional)')}</FormLabel>
              <FormControl>
                <PasswordInput
                  placeholder={t('Enter password (8-20 characters)')}
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        {isTurnstileEnabled && (
          <div className='mt-2'>
            <Turnstile
              siteKey={turnstileSiteKey}
              onVerify={setTurnstileToken}
            />
          </div>
        )}

        <LegalConsent
          status={status}
          checked={agreedToLegal}
          onCheckedChange={setAgreedToLegal}
          className='mt-1'
        />

        <Button
          type='submit'
          className='mt-2 w-full justify-center gap-2'
          disabled={isLoading || (requiresLegalConsent && !agreedToLegal)}
        >
          {isLoading ? <Loader2 className='h-4 w-4 animate-spin' /> : null}
          {t('Create account')}
        </Button>

        {onSwitchToPassword && (
          <Button
            type='button'
            variant='ghost'
            className='w-full justify-center'
            onClick={onSwitchToPassword}
          >
            {t('Sign up with password')}
          </Button>
        )}
      </form>
    </Form>
  )
}
