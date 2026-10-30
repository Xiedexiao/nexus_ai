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
import { useEffect } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'
import { CustomLinkIcon } from '@/components/custom-link-icon'
import { CustomLinkIconPicker } from '@/components/custom-link-icon-picker'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'

import {
  SettingsForm,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const customLinkSchema = z.object({
  links: z.array(
    z.object({
      id: z.string().min(1),
      title: z.string().min(1, { message: 'Name is required' }),
      target: z
        .string()
        .url({ message: 'Must be a valid http(s) URL' })
        .startsWith('http', { message: 'Must start with http(s)://' }),
      icon: z.string().optional(),
      proxy: z.boolean(),
      enabled: z.boolean(),
    })
  ),
})

export type CustomLinksFormValues = z.infer<typeof customLinkSchema>

type CustomLinksSectionProps = {
  defaultValues: { links: CustomLinksFormValues['links'] }
}

const QUICK_ICONS = [
  'globe',
  'book',
  'graduation-cap',
  'file-text',
  'message-square',
  'circle-help',
  'headphones',
  'newspaper',
  'megaphone',
  'shopping-cart',
  'star',
  'rocket',
  'bot',
  'terminal',
  'wrench',
  'package',
] as const

/**
 * Admin editor for the sidebar custom links. Stored as one CustomLinks option
 * (JSON array); the console receives ids only for proxy links, so the real
 * target never reaches user-side markup.
 */
export function CustomLinksSection(props: CustomLinksSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<CustomLinksFormValues>({
    defaultValues: props.defaultValues,
  })
  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: 'links',
  })

  useEffect(() => {
    form.reset(props.defaultValues)
  }, [props.defaultValues, form])

  async function onSubmit(values: CustomLinksFormValues) {
    const seen = new Set<string>()
    for (const [index, link] of values.links.entries()) {
      if (seen.has(link.id)) {
        toast.error(
          t('Link id must be unique', { index: index + 1 })
        )
        return
      }
      seen.add(link.id)
    }
    await updateOption.mutateAsync({
      key: 'CustomLinks',
      value: JSON.stringify(values.links),
    })
  }

  function addLink() {
    append({
      id: 'link-' + Math.random().toString(36).slice(2, 8),
      title: '',
      target: 'https://',
      icon: 'external',
      proxy: true,
      enabled: true,
    })
  }

  return (
    <SettingsSection title={t('Custom sidebar links')}>
      <SettingsPageFormActions
        onSave={form.handleSubmit(onSubmit)}
        isSaving={updateOption.isPending}
      />
      <p className='text-muted-foreground text-xs'>
        {t(
          'Configure extra sidebar entries. Proxy mode opens them under this site’s own /ext/ path so users never see the real address; direct mode opens the target in a new tab.'
        )}
      </p>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          {fields.map((field, index) => (
            <div key={field.id} className='rounded-lg border p-4'>
              <div className='grid grid-cols-1 gap-3 md:grid-cols-2'>
                <FormField
                  control={form.control}
                  name={`links.${index}.title`}
                  render={({ field: f }) => (
                    <FormItem>
                      <FormLabel>{t('Sidebar name')}</FormLabel>
                      <FormControl>
                        <Input {...f} value={f.value ?? ''} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`links.${index}.target`}
                  render={({ field: f }) => (
                    <FormItem>
                      <FormLabel>{t('Target URL')}</FormLabel>
                      <FormControl>
                        <Input {...f} value={f.value ?? ''} />
                      </FormControl>
                      {form.watch(`links.${index}.proxy`) && (
                        <FormDescription>
                          {t(
                            'Hidden from users: they open /ext/<id>/ on this site.'
                          )}
                        </FormDescription>
                      )}
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`links.${index}.icon`}
                  render={({ field: f }) => (
                    <FormItem>
                      <FormLabel>{t('Icon')}</FormLabel>
                      <div className='flex items-center gap-2'>
                        <FormControl>
                          <CustomLinkIconPicker
                            value={f.value}
                            onChange={f.onChange}
                          />
                        </FormControl>
                      </div>
                      <div className='flex flex-wrap gap-1'>
                        {QUICK_ICONS.map((name) => (
                          <button
                            key={name}
                            type='button'
                            onClick={() => f.onChange(name)}
                            className={cn(
                              'border-border hover:border-primary rounded-md border p-1.5',
                              f.value === name && 'border-primary bg-primary/10'
                            )}
                            aria-label={name}
                          >
                            <CustomLinkIcon name={name} />
                          </button>
                        ))}
                      </div>
                      <FormDescription>
                        {t(
                          'Pick from all Lucide icons, or type any icon name. Unknown names fall back to a link icon.'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name={`links.${index}.id`}
                  render={({ field: f }) => (
                    <FormItem>
                      <FormLabel>{t('Link id')}</FormLabel>
                      <FormControl>
                        <Input {...f} value={f.value ?? ''} />
                      </FormControl>
                      <FormDescription>
                        {t('Unique; used in the proxy path')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <div className='mt-3 flex flex-wrap items-center gap-4'>
                <FormField
                  control={form.control}
                  name={`links.${index}.proxy`}
                  render={({ field: f }) => (
                    <FormItem className='flex items-center gap-2'>
                      <Switch
                        checked={f.value}
                        onCheckedChange={f.onChange}
                      />
                      <FormLabel className='!mt-0'>
                        {t('Hide real address (proxy)')}
                      </FormLabel>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`links.${index}.enabled`}
                  render={({ field: f }) => (
                    <FormItem className='flex items-center gap-2'>
                      <Switch
                        checked={f.value}
                        onCheckedChange={f.onChange}
                      />
                      <FormLabel className='!mt-0'>{t('Enabled')}</FormLabel>
                    </FormItem>
                  )}
                />
                <Button
                  type='button'
                  variant='destructive'
                  size='sm'
                  className='ml-auto'
                  onClick={() => remove(index)}
                >
                  {t('Remove')}
                </Button>
              </div>
            </div>
          ))}
          <div className='flex items-center gap-2'>
            <Button type='button' variant='outline' size='sm' onClick={addLink}>
              {t('Add link')}
            </Button>
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
