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
import i18next from 'i18next'
import { useState } from 'react'
import { toast } from 'sonner'

import { useCountdown } from '@/hooks/use-countdown'
import { handleServerError } from '@/lib/handle-server-error'
import { AuthOperationError } from '@/lib/secure-verification'
import { createServerError } from '@/lib/server-error-message'

import { sendEmailVerification } from '../api'
import { EMAIL_VERIFICATION_COUNTDOWN } from '../constants'

interface UseEmailVerificationOptions {
  turnstileToken?: string
  validateTurnstile?: () => boolean
}

/**
 * Hook for managing email verification code sending
 */
export function useEmailVerification(options?: UseEmailVerificationOptions) {
  const [isSending, setIsSending] = useState(false)
  const {
    secondsLeft,
    isActive,
    start: startCountdown,
  } = useCountdown({ initialSeconds: EMAIL_VERIFICATION_COUNTDOWN })

  /**
   * Send verification code to email
   */
  const sendCode = async (email: string, sliderToken?: string) => {
    if (!email) {
      toast.error(i18next.t('Please enter your email first'))
      return false
    }

    // Validate turnstile if validation function is provided
    if (options?.validateTurnstile && !options.validateTurnstile()) {
      return false
    }

    setIsSending(true)
    try {
      const res = await sendEmailVerification(
        email,
        options?.turnstileToken,
        sliderToken
      )
      if (res?.success) {
        startCountdown()
        toast.success(i18next.t('Verification email sent'))
        return true
      }
      handleServerError(
        createServerError(res, i18next.t('Failed to send verification email'))
      )
      return false
    } catch (_error) {
      // The server enforces a fixed-window send limit per IP. When it answers
      // 429 it also sends Retry-After; adopt it as the local cooldown so the
      // button stays disabled until the server would accept again, instead of
      // letting the user click into the same rejection.
      const status = (
        _error as { response?: { status?: number; headers?: Record<string, unknown> } }
      )?.response?.status
      const retryAfter = Number(
        (
          _error as { response?: { headers?: Record<string, unknown> } }
        )?.response?.headers?.['retry-after']
      )
      if (status === 429 && Number.isFinite(retryAfter) && retryAfter > 0) {
        startCountdown(Math.min(Math.ceil(retryAfter), 120))
      }
      handleServerError(
        AuthOperationError.from(
          _error,
          i18next.t('Failed to send verification email')
        )
      )
      return false
    } finally {
      setIsSending(false)
    }
  }

  return {
    isSending,
    secondsLeft,
    isActive,
    sendCode,
  }
}
