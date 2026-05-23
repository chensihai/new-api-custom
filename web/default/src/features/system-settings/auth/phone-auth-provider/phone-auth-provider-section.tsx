import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PasswordInput } from '@/components/password-input'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { SettingsSection } from '../../components/settings-section'
import { usePhoneAdminProviders } from './use-phone-admin-providers'

const PROVIDERS = [
  { key: 'aliyun', labelKey: 'Aliyun' },
  { key: 'tencent', labelKey: 'Tencent Cloud' },
  { key: 'huawei', labelKey: 'Huawei Cloud' },
] as const

const PROVIDER_FIELDS = {
  aliyun: [
    { key: 'aliyun_access_key_id', labelKey: 'AccessKey ID', secret: false },
    { key: 'aliyun_access_key_secret', labelKey: 'AccessKey Secret', secret: true },
    { key: 'aliyun_sign_name', labelKey: 'SMS Signature', secret: false },
    { key: 'aliyun_template_code', labelKey: 'Template Code', secret: false },
  ],
  tencent: [
    { key: 'tencent_secret_id', labelKey: 'SecretId', secret: false },
    { key: 'tencent_secret_key', labelKey: 'SecretKey', secret: true },
    { key: 'tencent_sms_sdk_app_id', labelKey: 'SdkAppId', secret: false },
    { key: 'tencent_sign_name', labelKey: 'SMS Signature', secret: false },
    { key: 'tencent_template_id', labelKey: 'Template ID', secret: false },
  ],
  huawei: [
    { key: 'huawei_app_id', labelKey: 'APP Key', secret: false },
    { key: 'huawei_app_secret', labelKey: 'APP Secret', secret: true },
    { key: 'huawei_sign_name', labelKey: 'SMS Signature', secret: false },
    { key: 'huawei_template_id', labelKey: 'Template ID', secret: false },
    { key: 'huawei_sender', labelKey: 'Sender ID', secret: false },
  ],
} as const

const ENABLED_KEYS = {
  aliyun: 'aliyun_enabled',
  tencent: 'tencent_enabled',
  huawei: 'huawei_enabled',
} as const

export function PhoneAuthProviderSection() {
  const { t } = useTranslation()
  const { settings, setSettings, loading, saving, saveProviders } =
    usePhoneAdminProviders()
  const [activeTab, setActiveTab] = useState<string>('aliyun')

  const getEnabledProvider = (): string | null => {
    return (
      PROVIDERS.find((p) => settings[ENABLED_KEYS[p.key]])?.key || null
    )
  }

  const handleToggleProvider = (key: string, checked: boolean) => {
    setSettings((prev) => {
      const updated = { ...prev }
      if (checked) {
        for (const p of PROVIDERS) {
          updated[ENABLED_KEYS[p.key]] = p.key === key
        }
      } else {
        updated[ENABLED_KEYS[key as keyof typeof ENABLED_KEYS]] = false
      }
      return updated
    })
  }

  const handleFieldChange = (fieldKey: string, value: string) => {
    setSettings((prev) => ({ ...prev, [fieldKey]: value }))
  }

  const checkProviderComplete = (key: string): boolean => {
    const fields =
      PROVIDER_FIELDS[key as keyof typeof PROVIDER_FIELDS] || []
    return fields.every(
      (f) => settings[f.key] && String(settings[f.key]).trim() !== ''
    )
  }

  const handleSave = async () => {
    const enabledKey = getEnabledProvider()
    if (enabledKey && !checkProviderComplete(enabledKey)) {
      const provider = PROVIDERS.find((p) => p.key === enabledKey)
      toast.error(
        t('{{provider}} credentials are incomplete, please fill in all fields', {
          provider: provider ? t(provider.labelKey) : enabledKey,
        })
      )
      setActiveTab(enabledKey)
      return
    }
    await saveProviders({ ...settings })
  }

  if (loading) {
    return (
      <div className='flex items-center justify-center py-8'>
        <Loader2 className='h-6 w-6 animate-spin text-muted-foreground' />
      </div>
    )
  }

  const currentEnabled = getEnabledProvider()

  return (
    <SettingsSection
      title={t('Phone Auth Provider Settings')}
      description={t(
        'Configure SMS service providers for phone number authentication'
      )}
    >
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className='w-full'>
          {PROVIDERS.map((provider) => {
            const isEnabled = Boolean(settings[ENABLED_KEYS[provider.key]])
            const isComplete = checkProviderComplete(provider.key)
            return (
              <TabsTrigger key={provider.key} value={provider.key} className='flex-1'>
                <span className='flex items-center gap-1'>
                  {t(provider.labelKey)}
                  {isEnabled && !isComplete && (
                    <span className='text-xs text-warning'>(!)</span>
                  )}
                  {isEnabled && isComplete && (
                    <span className='text-xs text-green-600'>✓</span>
                  )}
                </span>
              </TabsTrigger>
            )
          })}
        </TabsList>

        {PROVIDERS.map((provider) => {
          const isEnabled = Boolean(settings[ENABLED_KEYS[provider.key]])
          const isOtherEnabled =
            currentEnabled !== null && currentEnabled !== provider.key
          const fields =
            PROVIDER_FIELDS[provider.key as keyof typeof PROVIDER_FIELDS] || []

          return (
            <TabsContent key={provider.key} value={provider.key} className='space-y-4'>
              <div className='flex flex-row items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <span className='text-base font-semibold'>
                    {t('Enable {{provider}}', {
                      provider: t(provider.labelKey),
                    })}
                  </span>
                </div>
                <Switch
                  checked={isEnabled}
                  onCheckedChange={(checked) =>
                    handleToggleProvider(provider.key, checked)
                  }
                  disabled={isOtherEnabled}
                />
              </div>

              {isOtherEnabled && (
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Another provider is already enabled. Only one SMS provider can be active at a time.'
                  )}
                </p>
              )}

              {isEnabled && (
                <div className='space-y-3'>
                  {fields.map((field) => (
                    <div key={field.key} className='space-y-1'>
                      <label className='text-sm font-medium'>
                        {t(field.labelKey)}
                      </label>
                      {field.secret ? (
                        <PasswordInput
                          placeholder={t(field.labelKey)}
                          value={String(settings[field.key] || '')}
                          onChange={(e) =>
                            handleFieldChange(field.key, e.target.value)
                          }
                        />
                      ) : (
                        <Input
                          placeholder={t(field.labelKey)}
                          value={String(settings[field.key] || '')}
                          onChange={(e) =>
                            handleFieldChange(field.key, e.target.value)
                          }
                        />
                      )}
                    </div>
                  ))}
                </div>
              )}
            </TabsContent>
          )
        })}
      </Tabs>

      <div className='flex justify-end pt-4'>
        <Button type='button' onClick={handleSave} disabled={saving}>
          {saving ? (
            <>
              <Loader2 className='h-4 w-4 animate-spin' />
              {t('Saving...')}
            </>
          ) : (
            t('Save Configuration')
          )}
        </Button>
      </div>
    </SettingsSection>
  )
}
