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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook } from '@testing-library/react'
import i18next from 'i18next'
import type { ReactNode } from 'react'
import { afterEach, expect, it } from 'vitest'

import { getDashboardSectionNavItems } from '@/features/dashboard/section-registry'
import { useAuthStore } from '@/stores/auth-store'

import { useSidebarConfig } from '../use-sidebar-config'
import { useSidebarData } from '../use-sidebar-data'

afterEach(() => useAuthStore.getState().auth.reset())

function navigation(role: number, status: Record<string, unknown> = {}) {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'test', role })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], status)
  function Wrapper(props: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        {props.children}
      </QueryClientProvider>
    )
  }
  return renderHook(() => useSidebarConfig(useSidebarData().navGroups), {
    wrapper: Wrapper,
  }).result.current.flatMap((group) => group.items)
}

it('shows separate recharge and billing links for customers', () => {
  expect(navigation(1).map((item) => item.url)).toEqual(
    expect.arrayContaining([
      '/wallet',
      '/billing',
      '/catalog',
      '/keys',
      '/usage-logs/common',
      '/usage-logs/task',
      '/contact',
    ])
  )
})
it('removes personal settings, audit and admin pages from customer navigation', () => {
  const urls = navigation(1).map((item) => item.url)
  for (const hidden of [
    '/profile',
    '/security',
    '/usage-logs/audit',
    '/channels',
  ]) {
    expect(urls).not.toContain(hidden)
  }
})
it('preserves administrator navigation', () => {
  expect(navigation(10).map((item) => item.url)).toEqual(
    expect.arrayContaining([
      '/profile',
      '/security',
      '/usage-logs/audit',
      '/playground',
      '/channels',
    ])
  )
})
it('honors the recharge toggle for both recharge and billing', () => {
  const urls = navigation(1, {
    SidebarModulesAdmin: JSON.stringify({
      personal: { enabled: true, topup: false },
    }),
  }).map((item) => item.url)
  expect(urls).not.toContain('/wallet')
  expect(urls).not.toContain('/billing')
})
it('honors the pricing module toggle', () => {
  expect(
    navigation(1, {
      HeaderNavModules: JSON.stringify({ pricing: { enabled: false } }),
    }).map((item) => item.url)
  ).not.toContain('/catalog')
})
it('links customer online chat directly to the existing playground without presets', () => {
  const items = navigation(1, { chats: '[]' })
  expect(items.find((item) => item.title === 'Online Chat')).toMatchObject({
    url: '/playground',
  })
  expect(items.some((item) => item.type === 'chat-presets')).toBe(false)
})
it('honors the playground toggle for the customer online chat entry', () => {
  const items = navigation(1, {
    SidebarModulesAdmin: JSON.stringify({
      chat: { enabled: true, playground: false, chat: true },
    }),
  })
  expect(items.some((item) => item.title === 'Online Chat')).toBe(false)
})
it('hides flow analysis from customer dashboard tabs while preserving admin tabs', () => {
  expect(
    getDashboardSectionNavItems(i18next.t).map((item) => item.url)
  ).not.toContain('/dashboard/flow')
  expect(
    getDashboardSectionNavItems(i18next.t, { isAdmin: true }).map(
      (item) => item.url
    )
  ).toContain('/dashboard/flow')
})
