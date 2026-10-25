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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'

// Every limiter shares the same shape; `enableKey` is omitted for the two that
// have no on/off switch. The option keys mirror the same-named environment
// variables, which only seed the defaults shown here.
type BoolLimitKey = {
  [K in keyof GlobalRateLimitDefaults]: GlobalRateLimitDefaults[K] extends boolean
    ? K
    : never
}[keyof GlobalRateLimitDefaults]

type NumLimitKey = {
  [K in keyof GlobalRateLimitDefaults]: GlobalRateLimitDefaults[K] extends number
    ? K
    : never
}[keyof GlobalRateLimitDefaults]

type LimiterConfig = {
  id: string
  numKey: NumLimitKey
  durationKey: NumLimitKey
  // Absent for the two limiters that have no on/off switch.
  enableKey?: BoolLimitKey
}

const LIMITERS: LimiterConfig[] = [
  {
    id: 'api',
    numKey: 'GlobalApiRateLimitNum',
    durationKey: 'GlobalApiRateLimitDuration',
    enableKey: 'GlobalApiRateLimitEnable',
  },
  {
    id: 'web',
    numKey: 'GlobalWebRateLimitNum',
    durationKey: 'GlobalWebRateLimitDuration',
    enableKey: 'GlobalWebRateLimitEnable',
  },
  {
    id: 'static',
    numKey: 'GlobalStaticRateLimitNum',
    durationKey: 'GlobalStaticRateLimitDuration',
    enableKey: 'GlobalStaticRateLimitEnable',
  },
  {
    id: 'critical',
    numKey: 'CriticalRateLimitNum',
    durationKey: 'CriticalRateLimitDuration',
    enableKey: 'CriticalRateLimitEnable',
  },
  {
    id: 'search',
    numKey: 'SearchRateLimitNum',
    durationKey: 'SearchRateLimitDuration',
    enableKey: 'SearchRateLimitEnable',
  },
  { id: 'upload', numKey: 'UploadRateLimitNum', durationKey: 'UploadRateLimitDuration' },
  {
    id: 'download',
    numKey: 'DownloadRateLimitNum',
    durationKey: 'DownloadRateLimitDuration',
  },
]

const countField = z.coerce.number().min(1).max(100000000)
const secondsField = z.coerce.number().min(1).max(2592000)

const globalRateLimitSchema = z.object({
  GlobalApiRateLimitEnable: z.boolean(),
  GlobalApiRateLimitNum: countField,
  GlobalApiRateLimitDuration: secondsField,
  GlobalWebRateLimitEnable: z.boolean(),
  GlobalWebRateLimitNum: countField,
  GlobalWebRateLimitDuration: secondsField,
  GlobalStaticRateLimitEnable: z.boolean(),
  GlobalStaticRateLimitNum: countField,
  GlobalStaticRateLimitDuration: secondsField,
  CriticalRateLimitEnable: z.boolean(),
  CriticalRateLimitNum: countField,
  CriticalRateLimitDuration: secondsField,
  SearchRateLimitEnable: z.boolean(),
  SearchRateLimitNum: countField,
  SearchRateLimitDuration: secondsField,
  UploadRateLimitNum: countField,
  UploadRateLimitDuration: secondsField,
  DownloadRateLimitNum: countField,
  DownloadRateLimitDuration: secondsField,
})

export type GlobalRateLimitDefaults = {
  GlobalApiRateLimitEnable: boolean
  GlobalApiRateLimitNum: number
  GlobalApiRateLimitDuration: number
  GlobalWebRateLimitEnable: boolean
  GlobalWebRateLimitNum: number
  GlobalWebRateLimitDuration: number
  GlobalStaticRateLimitEnable: boolean
  GlobalStaticRateLimitNum: number
  GlobalStaticRateLimitDuration: number
  CriticalRateLimitEnable: boolean
  CriticalRateLimitNum: number
  CriticalRateLimitDuration: number
  SearchRateLimitEnable: boolean
  SearchRateLimitNum: number
  SearchRateLimitDuration: number
  UploadRateLimitNum: number
  UploadRateLimitDuration: number
  DownloadRateLimitNum: number
  DownloadRateLimitDuration: number
}

type GlobalRateLimitFormInput = z.input<typeof globalRateLimitSchema>
type GlobalRateLimitFormValues = z.output<typeof globalRateLimitSchema>

const LIMITER_TITLES: Record<string, string> = {
  api: 'API requests',
  web: 'Console pages',
  static: 'Static assets',
  critical: 'Critical operations',
  search: 'Search',
  upload: 'Uploads',
  download: 'Downloads',
}

const LIMITER_DESCRIPTIONS: Record<string, string> = {
  api: 'Per IP across the relay API (/v1 and friends).',
  web: 'Per IP across console pages.',
  static: 'Per IP across built console assets; disabled by default.',
  critical: 'Per IP for login, password reset, and other sensitive endpoints.',
  search: 'Per user for search endpoints.',
  upload: 'Per IP for file uploads.',
  download: 'Per IP for file downloads.',
}

type GlobalRateLimitSectionProps = {
  defaultValues: GlobalRateLimitDefaults
}

export function GlobalRateLimitSection({
  defaultValues,
}: GlobalRateLimitSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const baselineRef = useRef<GlobalRateLimitDefaults>(defaultValues)
  const baselineSerializedRef = useRef<string>(JSON.stringify(defaultValues))

  const form = useForm<GlobalRateLimitFormInput, unknown, GlobalRateLimitFormValues>({
    resolver: zodResolver(globalRateLimitSchema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  useEffect(() => {
    const serialized = JSON.stringify(defaultValues)
    if (serialized === baselineSerializedRef.current) return
    baselineRef.current = defaultValues
    baselineSerializedRef.current = serialized
  }, [defaultValues])

  const onSubmit = async (values: GlobalRateLimitFormValues) => {
    const updates = (Object.keys(values) as Array<keyof GlobalRateLimitDefaults>)
      .filter((key) => values[key] !== baselineRef.current[key])
      .map((key) => ({
        key,
        value: String(values[key]),
      }))

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }

    baselineRef.current = values
    baselineSerializedRef.current = JSON.stringify(values)
  }

  return (
    <SettingsSection title={t('Global rate limits')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'Requests allowed per window; window length is in seconds. The matching environment variables only seed these defaults.'
            )}
          </p>
          <p className='text-muted-foreground text-xs'>
            {t('Takes effect after restart')}
          </p>

          <div className='grid grid-cols-1 gap-4 lg:grid-cols-2'>
            {LIMITERS.map((limiter) => (
              <div
                key={limiter.id}
                className='space-y-3 rounded-lg border p-4'
              >
                <div className='flex items-center justify-between gap-2'>
                  <div>
                    <p className='text-sm font-medium'>
                      {t(LIMITER_TITLES[limiter.id])}
                    </p>
                    <p className='text-muted-foreground text-xs'>
                      {t(LIMITER_DESCRIPTIONS[limiter.id])}
                    </p>
                  </div>
                  {limiter.enableKey && (
                    <FormField
                      control={form.control}
                      name={limiter.enableKey}
                      render={({ field }) => (
                        <SettingsSwitchItem>
                          <SettingsSwitchContent>
                            <span className='text-xs'>
                              {t('Enabled')}
                            </span>
                          </SettingsSwitchContent>
                          <FormControl>
                            <Switch
                              checked={field.value}
                              onCheckedChange={field.onChange}
                            />
                          </FormControl>
                        </SettingsSwitchItem>
                      )}
                    />
                  )}
                </div>
                <div className='grid grid-cols-2 gap-3'>
                  <FormField
                    control={form.control}
                    name={limiter.numKey}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Requests per window')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            min={1}
                            step={1}
                            {...safeNumberFieldProps(field)}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name={limiter.durationKey}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Window length (seconds)')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            min={1}
                            step={1}
                            {...safeNumberFieldProps(field)}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </div>
            ))}
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
