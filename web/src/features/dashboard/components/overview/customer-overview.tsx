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
import { Link } from '@tanstack/react-router'
import { ArrowRight, Box, FileText, KeyRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { ApiEndpointCard } from '@/components/api-endpoint-card'
import { SectionPageLayout } from '@/components/layout'
import { useIsSidebarModuleVisible } from '@/hooks/use-sidebar-config'
import { useStatus } from '@/hooks/use-status'
import { getModuleAccessFromStatus } from '@/lib/nav-modules'

import { useDashboardContentVisibility } from '../../hooks/use-status-data'
import { AnnouncementsPanel } from './announcements-panel'
import { FAQPanel } from './faq-panel'
import { SummaryCards } from './summary-cards'

export function CustomerOverview() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const pricingVisible = getModuleAccessFromStatus(
    status as Record<string, unknown> | null,
    'pricing'
  ).enabled
  const visibility = useDashboardContentVisibility()
  const keysVisible = useIsSidebarModuleVisible('/keys')
  const logsVisible = useIsSidebarModuleVisible('/usage-logs/common')
  const actions = [
    {
      to: '/keys',
      title: 'API Keys',
      description: 'Create a key for your app or service',
      icon: KeyRound,
      visible: keysVisible,
    },
    {
      to: '/catalog',
      title: 'Models & Pricing',
      description: 'Compare available models and prices for your API key.',
      icon: Box,
      visible: pricingVisible,
    },
    {
      to: '/usage-logs',
      title: 'Usage Logs',
      description: 'Review requests, usage, and charges.',
      icon: FileText,
      visible: logsVisible,
    },
  ] as const
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Overview')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='space-y-6'>
          <p className='text-muted-foreground text-sm'>
            {t('Your API workspace at a glance.')}
          </p>
          <SummaryCards customer />
          <div className='grid gap-4 md:grid-cols-3'>
            {actions
              .filter((action) => action.visible)
              .map((action) => (
                <Link
                  key={action.to}
                  to={action.to}
                  className='bg-card hover:border-primary/50 focus-visible:ring-ring flex min-w-0 items-center gap-3 rounded-xl border p-5 outline-none focus-visible:ring-2'
                >
                  <span className='bg-accent text-primary flex size-10 shrink-0 items-center justify-center rounded-lg'>
                    <action.icon className='size-5' />
                  </span>
                  <span className='min-w-0 flex-1'>
                    <span className='block font-medium'>{t(action.title)}</span>
                    <span className='text-muted-foreground mt-1 block text-xs leading-relaxed'>
                      {t(action.description)}
                    </span>
                  </span>
                  <ArrowRight className='text-muted-foreground size-4 shrink-0' />
                </Link>
              ))}
          </div>
          <ApiEndpointCard />
          {(visibility.announcements || visibility.faq) && (
            <div className='grid gap-4 xl:grid-cols-2'>
              {visibility.announcements && <AnnouncementsPanel />}
              {visibility.faq && <FAQPanel />}
            </div>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
