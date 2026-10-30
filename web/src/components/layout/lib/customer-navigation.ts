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
import type { TFunction } from 'i18next'
import {
  Activity,
  Box,
  CreditCard,
  FileText,
  Key,
  LayoutDashboard,
  ListTodo,
  MessageSquare,
  LifeBuoy,
  Wallet,
} from 'lucide-react'

import type { SidebarData } from '../types'

export function getCustomerNavigation(t: TFunction): SidebarData {
  return {
    navGroups: [
      {
        id: 'general',
        title: t('Workspace'),
        items: [
          { title: t('Overview'), url: '/dashboard/overview', icon: Activity },
          {
            title: t('Dashboard'),
            url: '/dashboard/models',
            icon: LayoutDashboard,
          },
          { title: t('Usage Logs'), url: '/usage-logs/common', icon: FileText },
          { title: t('Models & Pricing'), url: '/catalog', icon: Box },
          { title: t('API Keys'), url: '/keys', icon: Key },
          {
            title: t('Online Chat'),
            icon: MessageSquare,
            url: '/playground',
          },
          {
            title: t('Task Logs'),
            url: '/usage-logs/task',
            activeUrls: ['/usage-logs/drawing'],
            configUrls: ['/usage-logs/task', '/usage-logs/drawing'],
            icon: ListTodo,
          },
        ],
      },
      {
        id: 'personal',
        title: t('Billing'),
        items: [
          { title: t('Online Recharge'), url: '/wallet', icon: Wallet },
          { title: t('Recharge Bills'), url: '/billing', icon: CreditCard },
        ],
      },
      {
        id: 'support',
        title: t('Help'),
        items: [{ title: t('Contact Us'), url: '/contact', icon: LifeBuoy }],
      },
    ],
  }
}
