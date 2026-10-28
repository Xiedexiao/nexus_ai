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
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { RefreshCw, MoveHorizontal } from 'lucide-react'

import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'

type SliderCaptchaChallenge = {
  id: string
  background: string
  piece: string
  piece_y: number
}

type SliderCaptchaProps = {
  onToken: (token: string) => void
  resetKey?: unknown
}

const PUZZLE_W = 320
const PUZZLE_H = 160
const PIECE = 44
// The slider handle is 36px wide; the piece rests at x=0 when untouched.
const HANDLE = 36

/**
 * Drag-the-puzzle captcha. The backend draws the background with a punched
 * hole plus the matching piece; dragging the handle moves the piece. Releasing
 * verifies server-side and hands a short-lived pass token to the parent form.
 */
export function SliderCaptcha({ onToken, resetKey }: SliderCaptchaProps) {
  const { t } = useTranslation()
  const [challenge, setChallenge] = useState<SliderCaptchaChallenge | null>(null)
  const [loading, setLoading] = useState(false)
  const [dragging, setDragging] = useState(false)
  const [offset, setOffset] = useState(0)
  const [verifying, setVerifying] = useState(false)
  const [solved, setSolved] = useState(false)
  const grabDxRef = useRef(0)
  const startTimeRef = useRef(0)
  const trackRef = useRef<HTMLDivElement | null>(null)

  const loadChallenge = useCallback(async () => {
    setLoading(true)
    setSolved(false)
    setOffset(0)
    try {
      const res = await api.get('/api/captcha/slider')
      setChallenge(res.data?.data ?? null)
    } catch {
      toast.error(t('Failed to load captcha, please refresh'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadChallenge()
  }, [loadChallenge, resetKey])

  const maxOffset = PUZZLE_W - PIECE

  const finishDrag = useCallback(
    async (finalOffset: number) => {
      if (!challenge) return
      setDragging(false)
      setVerifying(true)
      const durationMs = Date.now() - startTimeRef.current
      try {
        const res = await api.post('/api/captcha/slider/verify', {
          id: challenge.id,
          x: Math.round(finalOffset),
          duration_ms: durationMs,
        })
        const token = res.data?.data?.token
        if (token) {
          setSolved(true)
          onToken(token)
        } else {
          toast.error(t('Captcha verification failed, please try again'))
          void loadChallenge()
        }
      } catch {
        toast.error(t('Captcha verification failed, please try again'))
        void loadChallenge()
      } finally {
        setVerifying(false)
      }
    },
    [challenge, loadChallenge, onToken, t]
  )

  const onPointerDown = (e: React.PointerEvent) => {
    if (solved || verifying || !challenge) return
    e.preventDefault()
    setDragging(true)
    grabDxRef.current = e.clientX - offset
    startTimeRef.current = Date.now()
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }

  const onPointerMove = (e: React.PointerEvent) => {
    if (!dragging) return
    const next = Math.min(maxOffset, Math.max(0, e.clientX - grabDxRef.current))
    setOffset(next)
  }

  const onPointerUp = () => {
    if (!dragging) return
    void finishDrag(offset)
  }

  return (
    <div className='space-y-2'>
      <div
        className='border-border relative w-full select-none overflow-hidden rounded-md border'
        style={{ maxWidth: PUZZLE_W, height: PUZZLE_H, touchAction: 'none' }}
      >
        {challenge ? (
          <>
            <img
              src={challenge.background}
              alt=''
              className='h-full w-full'
              draggable={false}
            />
            {/* The piece floats exactly on top of its home slot when at rest
                (offset 0 would overlap the left edge), so it starts hidden off
                the image left edge and slides in with the handle. */}
            <img
              src={challenge.piece}
              alt=''
              width={PIECE}
              height={PIECE}
              draggable={false}
              className='absolute top-0 drop-shadow-lg transition-[left] duration-75'
              style={{
                top: challenge.piece_y,
                left: offset,
                pointerEvents: 'none',
              }}
            />
            {solved && (
              <div className='absolute inset-0 flex items-center justify-center bg-black/30'>
                <span className='text-sm font-medium text-white'>
                  {t('Verification passed')}
                </span>
              </div>
            )}
          </>
        ) : (
          <div className='bg-muted flex h-full w-full items-center justify-center'>
            <span className='text-muted-foreground text-xs'>
              {loading ? t('Loading captcha...') : t('No captcha available')}
            </span>
          </div>
        )}
      </div>

      {!solved && (
        <div
          ref={trackRef}
          className='bg-muted relative h-9 w-full rounded-md'
          style={{ maxWidth: PUZZLE_W }}
        >
          <div
            className='bg-primary/15 absolute inset-y-0 left-0 rounded-l-md'
            style={{ width: offset + HANDLE }}
          />
          <div
            role='slider'
            aria-label={t('Drag the slider to fit the puzzle')}
            aria-valuenow={Math.round(offset)}
            aria-valuemin={0}
            aria-valuemax={maxOffset}
            aria-valuetext={`${Math.round((offset / maxOffset) * 100)}%`}
            tabIndex={0}
            onPointerDown={onPointerDown}
            onPointerMove={onPointerMove}
            onPointerUp={onPointerUp}
            onPointerCancel={onPointerUp}
            onKeyDown={(e) => {
              if (solved || verifying) return
              if (e.key === 'ArrowRight') {
                e.preventDefault()
                setOffset((o) => Math.min(maxOffset, o + 4))
              } else if (e.key === 'ArrowLeft') {
                e.preventDefault()
                setOffset((o) => Math.max(0, o - 4))
              } else if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                void finishDrag(offset)
              }
            }}
            className='bg-primary text-primary-foreground absolute top-0 flex h-9 w-9 cursor-grab touch-none items-center justify-center rounded-md shadow-sm active:cursor-grabbing'
            style={{ left: offset }}
          >
            <MoveHorizontal className='size-4' />
          </div>
          {!dragging && (
            <span className='text-muted-foreground pointer-events-none absolute inset-0 flex items-center justify-center text-xs'>
              {t('Drag the slider to fit the puzzle')}
            </span>
          )}
        </div>
      )}

      <div className='flex items-center justify-between' style={{ maxWidth: PUZZLE_W }}>
        <Button
          type='button'
          variant='ghost'
          size='sm'
          onClick={() => void loadChallenge()}
          disabled={loading || verifying}
        >
          <RefreshCw data-icon='inline-start' />
          <span>{t('Change captcha')}</span>
        </Button>
      </div>
    </div>
  )
}
