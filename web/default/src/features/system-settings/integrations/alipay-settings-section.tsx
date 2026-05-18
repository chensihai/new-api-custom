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
import * as React from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

export interface AlipaySettingsValues {
  AlipayEnabled: boolean
  AlipaySandbox: boolean
  AlipayAppId: string
  AlipayPrivateKey: string
  AlipayPublicKey: string
  AlipayMinTopUp: number
}

interface Props {
  defaultValues: AlipaySettingsValues
  serverAddress?: string
}

export function AlipaySettingsSection({ defaultValues, serverAddress }: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const initialRef = React.useRef(defaultValues)
  const defaultsSignature = React.useMemo(
    () => JSON.stringify(defaultValues),
    [defaultValues]
  )

  const form = useForm<AlipaySettingsValues>({
    defaultValues,
  })

  React.useEffect(() => {
    const parsedDefaults = JSON.parse(defaultsSignature) as AlipaySettingsValues
    initialRef.current = parsedDefaults
    form.reset(parsedDefaults)
  }, [defaultsSignature, form])

  const saveAlipaySettings = async () => {
    const values = form.getValues()
    const sanitized = {
      AlipayEnabled: values.AlipayEnabled as boolean,
      AlipaySandbox: values.AlipaySandbox as boolean,
      AlipayAppId: values.AlipayAppId.trim(),
      AlipayPrivateKey: values.AlipayPrivateKey.trim(),
      AlipayPublicKey: values.AlipayPublicKey.trim(),
      AlipayMinTopUp: values.AlipayMinTopUp as number,
    }

    const initial = {
      AlipayEnabled: initialRef.current.AlipayEnabled,
      AlipaySandbox: initialRef.current.AlipaySandbox,
      AlipayAppId: initialRef.current.AlipayAppId.trim(),
      AlipayPrivateKey: initialRef.current.AlipayPrivateKey.trim(),
      AlipayPublicKey: initialRef.current.AlipayPublicKey.trim(),
      AlipayMinTopUp: initialRef.current.AlipayMinTopUp,
    }

    const updates: Array<{ key: string; value: string | number | boolean }> = []

    if (sanitized.AlipayEnabled !== initial.AlipayEnabled) {
      updates.push({ key: 'AlipayEnabled', value: sanitized.AlipayEnabled })
    }
    if (sanitized.AlipaySandbox !== initial.AlipaySandbox) {
      updates.push({ key: 'AlipaySandbox', value: sanitized.AlipaySandbox })
    }
    if (sanitized.AlipayAppId !== initial.AlipayAppId) {
      updates.push({ key: 'AlipayAppId', value: sanitized.AlipayAppId })
    }
    if (sanitized.AlipayPrivateKey && sanitized.AlipayPrivateKey !== initial.AlipayPrivateKey) {
      updates.push({ key: 'AlipayPrivateKey', value: sanitized.AlipayPrivateKey })
    }
    if (sanitized.AlipayPublicKey && sanitized.AlipayPublicKey !== initial.AlipayPublicKey) {
      updates.push({ key: 'AlipayPublicKey', value: sanitized.AlipayPublicKey })
    }
    if (sanitized.AlipayMinTopUp !== initial.AlipayMinTopUp) {
      updates.push({ key: 'AlipayMinTopUp', value: sanitized.AlipayMinTopUp })
    }

    if (updates.length === 0) {
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  const callbackUrl = serverAddress
    ? `${serverAddress.replace(/\/+$/, '')}/api/alipay/notify`
    : '<ServerAddress>/api/alipay/notify'

  return (
    <SettingsSection
      title={t('Alipay Gateway')}
      description={t('Configuration for Alipay payment integration')}
    >
      <div className='rounded-md bg-blue-50 p-4 text-sm text-blue-900 dark:bg-blue-950 dark:text-blue-100'>
        <ul className='list-inside list-disc space-y-1'>
          <li>
            {t('Configure at:')}{' '}
            <a
              href='https://open.alipay.com'
              target='_blank'
              rel='noreferrer'
              className='underline hover:no-underline'
            >
              {t('Alipay Open Platform')}
            </a>
          </li>
          <li>
            {t('Supports PC web payment (alipay_page) and mobile web payment (alipay_wap)')}
          </li>
          <li>
            {t('Callback URL:')}{' '}
            <code className='rounded bg-blue-100 px-1 py-0.5 text-xs dark:bg-blue-900'>
              {callbackUrl}
            </code>
          </li>
        </ul>
      </div>

      <Form {...form}>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            e.stopPropagation()
            saveAlipaySettings()
          }}
          className='space-y-4'
          data-no-autosubmit='true'
        >
          <FormField
            control={form.control}
            name='AlipayEnabled'
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>
                    {t('Enable Alipay')}
                  </FormLabel>
                  <FormDescription>
                    {t('Allow users to pay with Alipay')}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='AlipaySandbox'
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>
                    {t('Sandbox Mode')}
                  </FormLabel>
                  <FormDescription>
                    {t('Use Alipay sandbox for development testing')}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          <div className='grid gap-6 md:grid-cols-2'>
            <FormField
              control={form.control}
              name='AlipayAppId'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('App ID')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('e.g., 2021xxxxxx')}
                      {...field}
                      onChange={(event) => field.onChange(event.target.value)}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='AlipayMinTopUp'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Minimum top-up (USD)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      step='0.01'
                      min={0}
                      value={(field.value ?? 1) as number}
                      onChange={(event) =>
                        field.onChange(event.target.valueAsNumber)
                      }
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className='grid gap-6 md:grid-cols-2'>
            <FormField
              control={form.control}
              name='AlipayPrivateKey'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('App Private Key')}</FormLabel>
                  <FormControl>
                    <Textarea
                      rows={3}
                      type='password'
                      placeholder={t('Enter new key to update, leave blank to keep current')}
                      autoComplete='new-password'
                      {...field}
                      onChange={(event) => field.onChange(event.target.value)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('RSA2 private key, will not be echoed after save')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='AlipayPublicKey'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Alipay Public Key')}</FormLabel>
                  <FormControl>
                    <Textarea
                      rows={3}
                      type='password'
                      placeholder={t('Enter Alipay public key')}
                      autoComplete='new-password'
                      {...field}
                      onChange={(event) => field.onChange(event.target.value)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Used for callback notification verification')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <Button
            type='button'
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              saveAlipaySettings()
            }}
            disabled={updateOption.isPending}
          >
            {updateOption.isPending
              ? t('Saving...')
              : t('Save Alipay settings')}
          </Button>
        </form>
      </Form>
    </SettingsSection>
  )
}
