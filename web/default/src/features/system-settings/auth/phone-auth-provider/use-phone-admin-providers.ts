import { useState, useEffect, useCallback } from 'react'
import { toast } from 'sonner'
import i18next from 'i18next'
import {
  getPhoneAuthProviders,
  updatePhoneAuthProviders,
} from './api'
import type { PhoneAuthProvidersResponse } from './types'

export function usePhoneAdminProviders() {
  const [settings, setSettings] = useState<Record<string, string | boolean>>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const fetchProviders = useCallback(async () => {
    setLoading(true)
    try {
      const res: PhoneAuthProvidersResponse = await getPhoneAuthProviders()
      if (res.success && res.data?.settings) {
        setSettings(res.data.settings)
      } else if (!res.success) {
        toast.error(res.message || i18next.t('Failed to load phone auth provider settings'))
      }
    } catch {
      toast.error(i18next.t('Failed to load phone auth provider settings'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchProviders()
  }, [fetchProviders])

  const saveProviders = async (payload: Record<string, string | boolean>) => {
    setSaving(true)
    try {
      const res = await updatePhoneAuthProviders(payload)
      if (res.success) {
        toast.success(i18next.t('Settings saved successfully'))
        await fetchProviders()
        return true
      }
      toast.error(res.message || i18next.t('Failed to save settings'))
      return false
    } catch {
      toast.error(i18next.t('Failed to save settings'))
      return false
    } finally {
      setSaving(false)
    }
  }

  return { settings, setSettings, loading, saving, saveProviders }
}
