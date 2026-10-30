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
import { Check, Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { useStatus } from '@/hooks/use-status'

export function ApiEndpointCard() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const configured =
    typeof status?.server_address === 'string'
      ? status.server_address.trim()
      : ''
  const base = (configured || window.location.origin).replace(/\/+$/, '')
  const url = base.endsWith('/v1') ? base : `${base}/v1`
  const { copyToClipboard, copiedText } = useCopyToClipboard({ notify: false })
  return (
    <section
      className='bg-card flex min-w-0 flex-wrap items-center justify-between gap-4 rounded-xl border p-5'
      aria-label={t('API Base URL')}
    >
      <div className='min-w-0 flex-1'>
        <h3 className='text-sm font-medium'>{t('API Base URL')}</h3>
        <p className='text-muted-foreground mt-1 text-xs'>
          {t('Use this endpoint in your SDK or API client.')}
        </p>
      </div>
      <div className='bg-muted flex max-w-full min-w-0 items-center gap-3 rounded-lg border px-3 py-2'>
        <code className='min-w-0 overflow-x-auto text-sm whitespace-nowrap'>
          {url}
        </code>
        <Button
          variant='ghost'
          size='icon'
          aria-label={t('Copy API Base URL')}
          onClick={() => void copyToClipboard(url)}
        >
          {copiedText === url ? (
            <Check className='size-4' />
          ) : (
            <Copy className='size-4' />
          )}
        </Button>
      </div>
    </section>
  )
}
