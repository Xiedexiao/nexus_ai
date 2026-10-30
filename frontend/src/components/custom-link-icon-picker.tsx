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
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import {
  CustomLinkIcon,
  CUSTOM_LINK_ICON_NAMES,
} from '@/components/custom-link-icon'

type CustomLinkIconPickerProps = {
  value?: string
  onChange: (name: string) => void
}

// Local search across every Lucide icon name; nothing leaves the page.
export function CustomLinkIconPicker({ value, onChange }: CustomLinkIconPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')

  const filtered = useMemo(() => {
    if (!open) return []
    const q = query.trim().toLowerCase()
    return CUSTOM_LINK_ICON_NAMES.filter((name) => !q || name.includes(q)).slice(
      0,
      120
    )
  }, [open, query])

  return (
    <>
      <Button
        type='button'
        variant='outline'
        className='w-full justify-start gap-2'
        onClick={() => setOpen(true)}
      >
        <CustomLinkIcon name={value} />
        <span className='font-mono text-xs'>{value || 'external-link'}</span>
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className='sm:max-w-xl'>
          <DialogHeader>
            <DialogTitle>{t('Choose an icon')}</DialogTitle>
          </DialogHeader>
          <div className='relative'>
            <Search className='text-muted-foreground absolute top-1/2 left-2.5 size-4 -translate-y-1/2' />
            <Input
              autoFocus
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t('Search icons by name...')}
              className='pl-8'
            />
          </div>
          <div className='grid max-h-80 grid-cols-8 gap-1 overflow-y-auto pt-1'>
            {filtered.map((name) => (
              <button
                key={name}
                type='button'
                title={name}
                onClick={() => {
                  onChange(name)
                  setOpen(false)
                }}
                className={cn(
                  'hover:bg-primary/10 rounded-md p-2',
                  value === name && 'bg-primary/15 border-primary border'
                )}
              >
                <CustomLinkIcon name={name} className='mx-auto' />
              </button>
            ))}
            {filtered.length === 0 && (
              <p className='text-muted-foreground col-span-8 py-6 text-center text-sm'>
                {t('No icons match your search')}
              </p>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
