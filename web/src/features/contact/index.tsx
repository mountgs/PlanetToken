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
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { RichContent } from '@/components/rich-content'
import { Skeleton } from '@/components/ui/skeleton'
import { requireServerSuccess } from '@/lib/server-error-message'

import { getContactContent } from './api'

export function Contact() {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['contact-content'],
    queryFn: async () => requireServerSuccess(await getContactContent()),
  })
  const content = query.data?.data?.trim() ?? ''
  let body = (
    <p
      role='status'
      className='bg-card text-muted-foreground rounded-xl border p-6 text-sm'
    >
      {t('Contact details have not been configured yet.')}
    </p>
  )
  if (query.isPending) {
    body = (
      <div role='status' aria-label={t('Loading...')} className='space-y-4'>
        <Skeleton className='h-6 w-48' />
        <Skeleton className='h-40 w-full max-w-3xl' />
      </div>
    )
  } else if (query.isError) {
    body = (
      <div role='alert'>
        <ErrorState
          title={t('Unable to load contact details')}
          onRetry={() => void query.refetch()}
        />
      </div>
    )
  } else if (content) {
    body = (
      <section className='bg-card max-w-4xl rounded-xl border p-6 sm:p-8'>
        <RichContent
          content={content}
          breaks
          className='[overflow-wrap:anywhere] break-words [&_img]:max-h-80 [&_img]:max-w-full [&_img]:object-contain'
        />
      </section>
    )
  }
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Contact Us')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>{body}</SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
