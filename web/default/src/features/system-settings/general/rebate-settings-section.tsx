import { z } from 'zod'
import { useForm, type Resolver } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
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
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const schema = z.object({
  enabled: z.boolean(),
  settlementPeriod: z.coerce.number().int().min(0),
  visibleGroups: z.string(),
})

type Values = z.infer<typeof schema>

export function RebateSettingsSection({
  defaultValues,
}: {
  defaultValues: {
    enabled: boolean
    settlementPeriod: number
    visibleGroups: string
  }
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<Values>({
    resolver: zodResolver(schema) as unknown as Resolver<Values>,
    defaultValues: {
      enabled: defaultValues.enabled,
      settlementPeriod: defaultValues.settlementPeriod,
      visibleGroups: defaultValues.visibleGroups,
    },
  })

  const { isDirty, isSubmitting } = form.formState
  const enabled = form.watch('enabled')

  async function onSubmit(values: Values) {
    const updates: Array<{ key: string; value: string }> = []

    if (values.enabled !== defaultValues.enabled) {
      updates.push({
        key: 'rebate_setting.enabled',
        value: String(values.enabled),
      })
    }

    if (values.settlementPeriod !== defaultValues.settlementPeriod) {
      updates.push({
        key: 'rebate_setting.settlement_period',
        value: String(values.settlementPeriod),
      })
    }

    if (values.visibleGroups !== defaultValues.visibleGroups) {
      updates.push({
        key: 'rebate_setting.visible_groups',
        value: values.visibleGroups,
      })
    }

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }

    form.reset(values)
  }

  return (
    <SettingsSection
      title={t('Rebate Settings')}
      description={t('Configure rebate settings for invitation rewards')}
    >
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          autoComplete='off'
          className='space-y-6'
        >
          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>
                    {t('Rebate Enabled')}
                  </FormLabel>
                  <FormDescription>
                    {t('Earn rebate when your referrals top up')}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={updateOption.isPending || isSubmitting}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          {enabled && (
            <div className='grid gap-6 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='settlementPeriod'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Settlement Period (days)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        placeholder='7'
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Days to wait before settling rebate')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='visibleGroups'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Rebate Visible Groups')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='default,vip'
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Comma-separated group names that can see rebate info')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          )}

          <Button
            type='submit'
            disabled={!isDirty || updateOption.isPending || isSubmitting}
          >
            {updateOption.isPending || isSubmitting
              ? t('Saving...')
              : t('Save rebate settings')}
          </Button>
        </form>
      </Form>
    </SettingsSection>
  )
}
