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

const runtimePolicySchema = z.object({
  TaskTimeoutMinutes: z.coerce.number().min(0).max(527040),
  TaskPollMaxFailures: z.coerce.number().min(0).max(100000),
  VerificationValidMinutes: z.coerce.number().min(1).max(1440),
})

export type RuntimePolicyDefaults = {
  TaskTimeoutMinutes: number
  TaskPollMaxFailures: number
  VerificationValidMinutes: number
}

type RuntimePolicyFormInput = z.input<typeof runtimePolicySchema>
type RuntimePolicyFormValues = z.output<typeof runtimePolicySchema>

type RuntimePolicySectionProps = {
  defaultValues: RuntimePolicyDefaults
}

export function RuntimePolicySection({
  defaultValues,
}: RuntimePolicySectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const baselineRef = useRef<RuntimePolicyDefaults>(defaultValues)
  const baselineSerializedRef = useRef<string>(JSON.stringify(defaultValues))

  const form = useForm<RuntimePolicyFormInput, unknown, RuntimePolicyFormValues>({
    resolver: zodResolver(runtimePolicySchema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  useEffect(() => {
    const serialized = JSON.stringify(defaultValues)
    if (serialized === baselineSerializedRef.current) return
    baselineRef.current = defaultValues
    baselineSerializedRef.current = serialized
  }, [defaultValues])

  const onSubmit = async (values: RuntimePolicyFormValues) => {
    const updates = (Object.keys(values) as Array<keyof RuntimePolicyDefaults>)
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
    <SettingsSection title={t('Background tasks & verification codes')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <p className='text-muted-foreground text-xs'>
            {t('Takes effect immediately')}
          </p>

          <div className='grid grid-cols-1 gap-4 md:grid-cols-3'>
            <FormField
              control={form.control}
              name='TaskTimeoutMinutes'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Async task hard timeout (minutes)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Tasks older than this are marked failed and refunded; 0 disables the cutoff.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='TaskPollMaxFailures'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Task poll failure cutoff')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Consecutive poll failures before a task is marked failed and refunded; 0 disables the cutoff.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='VerificationValidMinutes'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Verification code validity (minutes)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={1}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'How long email verification and password-reset codes stay usable.'
                    )}
                  </FormDescription>
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
