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
import { useCallback, useEffect, useRef, useState } from 'react'

import { api } from '@/lib/api'

type CheckResult = {
  taken: boolean
}

/**
 * Debounced registration-email availability probe. Calls the guarded
 * /api/check_email endpoint after the user stops typing a plausible address
 * and reports whether the mailbox already holds an account, so the form can
 * redirect to sign-in instead of walking the user into a send-code rejection.
 */
export function useEmailTaken(email: string | undefined, enabled: boolean) {
  const [taken, setTaken] = useState(false)
  const [checking, setChecking] = useState(false)
  const lastCheckedRef = useRef<string>('')

  const check = useCallback(async (value: string) => {
    const trimmed = value.trim()
    if (!trimmed || !trimmed.includes('@')) {
      setTaken(false)
      return
    }
    setChecking(true)
    try {
      const res = await api.get<{ data?: CheckResult }>('/api/check_email', {
        params: { email: trimmed },
      })
      setTaken(Boolean(res.data?.data?.taken))
    } catch {
      // A failed probe must not block registration; the send-code route still
      // enforces the authoritative check server-side.
      setTaken(false)
    } finally {
      setChecking(false)
    }
  }, [])

  useEffect(() => {
    const trimmed = (email ?? '').trim()
    if (!enabled || !trimmed || !trimmed.includes('@')) {
      setTaken(false)
      return
    }
    if (lastCheckedRef.current === trimmed) return
    lastCheckedRef.current = trimmed
    const timer = window.setTimeout(() => {
      void check(trimmed)
    }, 600)
    return () => window.clearTimeout(timer)
  }, [email, enabled, check])

  const reset = useCallback(() => {
    lastCheckedRef.current = ''
    setTaken(false)
  }, [])

  return { emailTaken: taken, checkingEmail: checking, resetEmailCheck: reset }
}
