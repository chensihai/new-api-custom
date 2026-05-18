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

export interface WechatSettingsValues {
  WechatPayEnabled: boolean
  WechatPayMchID: string
  WechatPaySerialNo: string
  WechatPayAPIv3Key: string
  WechatPayPrivateKey: string
  WechatPayMinTopUp: number
}

interface Props {
  defaultValues: WechatSettingsValues
  serverAddress?: string
}

export function WechatSettingsSection({ defaultValues, serverAddress }: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const initialRef = React.useRef(defaultValues)
  const defaultsSignature = React.useMemo(
    () => JSON.stringify(defaultValues),
    [defaultValues]
  )

  const form = useForm<WechatSettingsValues>({
    defaultValues,
  })

  React.useEffect(() => {
    const parsedDefaults = JSON.parse(defaultsSignature) as WechatSettingsValues
    initialRef.current = parsedDefaults
    form.reset(parsedDefaults)
  }, [defaultsSignature, form])

  const saveWechatSettings = async () => {
    const values = form.getValues()
    const sanitized = {
      WechatPayEnabled: values.WechatPayEnabled as boolean,
      WechatPayMchID: values.WechatPayMchID.trim(),
      WechatPaySerialNo: values.WechatPaySerialNo.trim(),
      WechatPayAPIv3Key: values.WechatPayAPIv3Key.trim(),
      WechatPayPrivateKey: values.WechatPayPrivateKey.trim(),
      WechatPayMinTopUp: values.WechatPayMinTopUp as number,
    }

    const initial = {
      WechatPayEnabled: initialRef.current.WechatPayEnabled,
      WechatPayMchID: initialRef.current.WechatPayMchID.trim(),
      WechatPaySerialNo: initialRef.current.WechatPaySerialNo.trim(),
      WechatPayAPIv3Key: initialRef.current.WechatPayAPIv3Key.trim(),
      WechatPayPrivateKey: initialRef.current.WechatPayPrivateKey.trim(),
      WechatPayMinTopUp: initialRef.current.WechatPayMinTopUp,
    }

    const updates: Array<{ key: string; value: string | number | boolean }> = []

    if (sanitized.WechatPayEnabled !== initial.WechatPayEnabled) {
      updates.push({ key: 'WechatPayEnabled', value: sanitized.WechatPayEnabled })
    }
    if (sanitized.WechatPayMchID !== initial.WechatPayMchID) {
      updates.push({ key: 'WechatPayMchID', value: sanitized.WechatPayMchID })
    }
    if (sanitized.WechatPaySerialNo !== initial.WechatPaySerialNo) {
      updates.push({ key: 'WechatPaySerialNo', value: sanitized.WechatPaySerialNo })
    }
    if (sanitized.WechatPayAPIv3Key && sanitized.WechatPayAPIv3Key !== initial.WechatPayAPIv3Key) {
      updates.push({ key: 'WechatPayAPIv3Key', value: sanitized.WechatPayAPIv3Key })
    }
    if (sanitized.WechatPayPrivateKey && sanitized.WechatPayPrivateKey !== initial.WechatPayPrivateKey) {
      updates.push({ key: 'WechatPayPrivateKey', value: sanitized.WechatPayPrivateKey })
    }
    if (sanitized.WechatPayMinTopUp !== initial.WechatPayMinTopUp) {
      updates.push({ key: 'WechatPayMinTopUp', value: sanitized.WechatPayMinTopUp })
    }

    if (updates.length === 0) {
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  const callbackUrl = serverAddress
    ? `${serverAddress.replace(/\/+$/, '')}/api/wechat/notify`
    : '<ServerAddress>/api/wechat/notify'

  return (
    <SettingsSection
      title={t('WeChat Pay Gateway')}
      description={t('Configuration for WeChat Pay integration')}
    >
      <div className='rounded-md bg-blue-50 p-4 text-sm text-blue-900 dark:bg-blue-950 dark:text-blue-100'>
        <ul className='list-inside list-disc space-y-1'>
          <li>
            {t('Configure at:')}{' '}
            <a
              href='https://pay.weixin.qq.com'
              target='_blank'
              rel='noreferrer'
              className='underline hover:no-underline'
            >
              {t('WeChat Pay Merchant Platform')}
            </a>
          </li>
          <li>
            {t('Supports Native QR code payment (wechat_native) and H5 payment (wechat_h5)')}
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
            saveWechatSettings()
          }}
          className='space-y-4'
          data-no-autosubmit='true'
        >
          <FormField
            control={form.control}
            name='WechatPayEnabled'
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>
                    {t('Enable WeChat Pay')}
                  </FormLabel>
                  <FormDescription>
                    {t('Allow users to pay with WeChat Pay')}
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
              name='WechatPayMchID'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Merchant ID')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('e.g., 16xxxxxx')}
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
              name='WechatPayMinTopUp'
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

          <div className='grid gap-6 md:grid-cols-3'>
            <FormField
              control={form.control}
              name='WechatPaySerialNo'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Certificate Serial Number')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('e.g., 5A2xxxxxx')}
                      {...field}
                      onChange={(event) => field.onChange(event.target.value)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Serial number of merchant API certificate')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='WechatPayAPIv3Key'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('APIv3 Key')}</FormLabel>
                  <FormControl>
                    <Input
                      type='password'
                      placeholder={t('Enter new key to update, leave blank to keep current')}
                      autoComplete='new-password'
                      {...field}
                      onChange={(event) => field.onChange(event.target.value)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('For callback verification and decryption, will not be echoed after save')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='WechatPayPrivateKey'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Merchant Private Key')}</FormLabel>
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
                    {t('PEM format private key, will not be echoed after save')}
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
              saveWechatSettings()
            }}
            disabled={updateOption.isPending}
          >
            {updateOption.isPending
              ? t('Saving...')
              : t('Save WeChat Pay settings')}
          </Button>
        </form>
      </Form>
    </SettingsSection>
  )
}
