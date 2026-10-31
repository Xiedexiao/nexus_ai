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
import { useQuery } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CompactDateTimeRangePicker } from './compact-date-time-range-picker'
import { Progress } from '@/components/ui/progress'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { api } from '@/lib/api'
import { formatCompactNumber } from '@/lib/format'
import { toIntlLocale } from '@/i18n/languages'

type ModelCacheHitStat = {
  model_name: string
  request_count: number
  cache_read_tokens: number
  cache_creation_tokens: number
  prompt_tokens: number
  cache_hit_rate: number
}

function formatTokens(value: number, locale: string): string {
  return formatCompactNumber(value, locale)
}

function rateColor(rate: number): string {
  if (rate >= 0.6) return 'text-success'
  if (rate >= 0.25) return 'text-warning'
  return 'text-destructive'
}

/**
 * Per-model cache-hit-rate panel for the usage-logs page, scoped to the
 * current user's own usage. Visibility is server-controlled: the admin can
 * disable the view, and the endpoint answers with an error when disabled, so
 * the panel collapses to that notice (queries fail silently, no toast).
 */
export function CacheHitStatsPanel() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language) ?? 'en'
  const now = Math.floor(Date.now() / 1000)
  const [range, setRange] = useState<{ start: number; end: number }>({
    start: now - 24 * 3600,
    end: now,
  })

  const query = useQuery({
    queryKey: ['cache-hit-stats', range.start, range.end],
    queryFn: async () => {
      const res = await api.get<{ data?: ModelCacheHitStat[] }>(
        '/api/data/cache_hit_by_model',
        { params: { start_timestamp: range.start, end_timestamp: range.end } }
      )
      return res.data?.data ?? []
    },
    retry: false,
    meta: { errorToast: false },
  })

  const rows = useMemo(
    () =>
      (query.data ?? []).map((row) => ({
        ...row,
        hitRatePct: row.cache_hit_rate * 100,
      })),
    [query.data]
  )

  if (query.isError) {
    return (
      <div className='text-muted-foreground rounded-md border p-6 text-center text-sm'>
        {t('Cache hit statistics are disabled by administrator')}
      </div>
    )
  }

  return (
    <div className='space-y-3'>
      <div className='flex items-center justify-between gap-2'>
        <CompactDateTimeRangePicker
          start={new Date(range.start * 1000)}
          end={new Date(range.end * 1000)}
          onChange={({ start, end }) => {
            if (start) setRange((r) => ({ ...r, start: Math.floor(start.getTime() / 1000) }))
            if (end) setRange((r) => ({ ...r, end: Math.floor(end.getTime() / 1000) }))
          }}
        />
      </div>

      <div className='rounded-md border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Model')}</TableHead>
              <TableHead className='text-right'>
                {t('Requests')}
              </TableHead>
              <TableHead className='text-right'>
                {t('Cache reads')}
              </TableHead>
              <TableHead className='text-right'>
                {t('Cache writes')}
              </TableHead>
              <TableHead className='text-right'>
                {t('Uncached prompt')}
              </TableHead>
              <TableHead className='w-[180px]'>{t('Cache hit rate')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {query.isLoading ? (
              <TableRow>
                <TableCell colSpan={6} className='text-muted-foreground h-24 text-center text-sm'>
                  {t('Loading...')}
                </TableCell>
              </TableRow>
            ) : rows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className='text-muted-foreground h-24 text-center text-sm'>
                  {t('No data for the selected period')}
                </TableCell>
              </TableRow>
            ) : (
              rows.map((row) => (
                <TableRow key={row.model_name}>
                  <TableCell className='font-medium'>
                    {row.model_name}
                  </TableCell>
                  <TableCell className='text-right'>
                    {formatTokens(row.request_count, locale)}
                  </TableCell>
                  <TableCell className='text-right'>
                    {formatTokens(row.cache_read_tokens, locale)}
                  </TableCell>
                  <TableCell className='text-right'>
                    {formatTokens(row.cache_creation_tokens, locale)}
                  </TableCell>
                  <TableCell className='text-right'>
                    {formatTokens(row.prompt_tokens, locale)}
                  </TableCell>
                  <TableCell>
                    <div className='flex items-center gap-2'>
                      <Progress
                        value={row.hitRatePct}
                        className='h-2 w-24'
                        aria-label={t('Cache hit rate')}
                      />
                      <span
                        className={`text-xs font-medium tabular-nums ${rateColor(
                          row.cache_hit_rate
                        )}`}
                      >
                        {row.hitRatePct.toFixed(1)}%
                      </span>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
      <p className='text-muted-foreground text-xs'>
        {t(
          'Hit rate = cached prompt tokens ÷ total prompt tokens (reads + writes + uncached). Data comes from usage aggregation and updates with the batch interval.'
        )}
      </p>
    </div>
  )
}
