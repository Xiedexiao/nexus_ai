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
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'

// These keys mirror the same-named environment variables; the environment
// value only seeds the default shown here, and anything saved from this page
// is stored in the options table and wins from then on.
const relayTimeoutSchema = z.object({
  RelayTimeout: z.coerce.number().min(0),
  RelayIdleConnTimeout: z.coerce.number().min(0),
  RelayResponseHeaderTimeout: z.coerce.number().min(0),
  RelayMaxIdleConns: z.coerce.number().min(0),
  RelayMaxIdleConnsPerHost: z.coerce.number().min(0),
  StreamingTimeout: z.coerce.number().min(0),
})

type RelayTimeoutFormInput = z.input<typeof relayTimeoutSchema>
type RelayTimeoutFormValues = z.output<typeof relayTimeoutSchema>

export type RelayTimeoutDefaults = {
  RelayTimeout: number
  RelayIdleConnTimeout: number
  RelayResponseHeaderTimeout: number
  RelayMaxIdleConns: number
  RelayMaxIdleConnsPerHost: number
  StreamingTimeout: number
}

type RelayTimeoutSettingsSectionProps = {
  defaultValues: RelayTimeoutDefaults
}

export function RelayTimeoutSettingsSection({
  defaultValues,
}: RelayTimeoutSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const baselineRef = useRef<RelayTimeoutDefaults>(defaultValues)
  const baselineSerializedRef = useRef<string>(JSON.stringify(defaultValues))

  const form = useForm<RelayTimeoutFormInput, unknown, RelayTimeoutFormValues>({
    resolver: zodResolver(relayTimeoutSchema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  useEffect(() => {
    const serialized = JSON.stringify(defaultValues)
    if (serialized === baselineSerializedRef.current) return
    baselineRef.current = defaultValues
    baselineSerializedRef.current = serialized
  }, [defaultValues])

  const onSubmit = async (values: RelayTimeoutFormValues) => {
    const updates = (Object.keys(values) as Array<keyof RelayTimeoutDefaults>)
      .filter((key) => values[key] !== baselineRef.current[key])
      .map((key) => ({ key, value: String(values[key]) }))

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
    <SettingsSection title={t('Relay & streaming timeouts')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'Seconds the gateway waits on upstream providers; 0 disables a timeout. The matching environment variables only seed these defaults.'
            )}
          </p>

          <div className='grid grid-cols-1 gap-4 md:grid-cols-3'>
            <FormField
              control={form.control}
              name='StreamingTimeout'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Streaming idle timeout')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <FormDescription>{t('Takes effect immediately')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='RelayTimeout'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Overall request timeout')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <FormDescription>{t('Takes effect after restart')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='RelayResponseHeaderTimeout'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Response header wait timeout')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <FormDescription>{t('Takes effect after restart')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='RelayIdleConnTimeout'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Idle connection timeout')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <FormDescription>{t('Takes effect after restart')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='RelayMaxIdleConns'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Max idle connections')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <FormDescription>{t('Takes effect after restart')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='RelayMaxIdleConnsPerHost'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Max idle connections per host')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <FormDescription>{t('Takes effect after restart')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
