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
import { render, screen } from '@testing-library/react'
import type { CSSProperties, ReactNode } from 'react'
import { afterEach, expect, it, vi } from 'vitest'

import { useAuthStore } from '@/stores/auth-store'

import { AuthenticatedLayout } from '../components/authenticated-layout'

vi.mock('@/components/command-menu', () => ({ CommandMenu: () => null }))
vi.mock('@/components/skip-to-main', () => ({ SkipToMain: () => null }))
vi.mock('@/components/page-transition', () => ({ AnimatedOutlet: () => null }))
vi.mock('@/components/ui/sidebar', () => ({
  SidebarProvider: (props: {
    children: ReactNode
    className: string
    style?: CSSProperties
  }) => (
    <div data-testid='shell' className={props.className} style={props.style}>
      {props.children}
    </div>
  ),
  SidebarInset: (props: { children: ReactNode; className: string }) => (
    <main className={props.className}>{props.children}</main>
  ),
}))
vi.mock('@/context/layout-provider', () => ({
  LayoutProvider: (props: { children: ReactNode }) => props.children,
}))
vi.mock('@/context/search-provider', () => ({
  SearchProvider: (props: { children: ReactNode }) => props.children,
}))
vi.mock('../components/app-sidebar', () => ({ AppSidebar: () => null }))
vi.mock('../components/app-header', () => ({
  AppHeader: (props: { customer: boolean; showConfigDrawer: boolean }) => (
    <header>
      {props.customer ? 'Customer header' : 'Admin header'}
      {props.showConfigDrawer && (
        <button type='button'>Theme configuration</button>
      )}
    </header>
  ),
}))

afterEach(() => useAuthStore.getState().auth.reset())

it('scopes the customer palette to the document so portals inherit it and removes it on unmount', () => {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'customer', role: 1 })
  const view = render(<AuthenticatedLayout>Content</AuthenticatedLayout>)
  expect(document.documentElement).toHaveAttribute('data-customer-ui', 'true')
  expect(screen.getByTestId('shell')).toHaveClass('customer-shell')
  expect(
    screen.getByTestId('shell').style.getPropertyValue('--sidebar-width')
  ).toBe('15rem')
  expect(
    screen.queryByRole('button', { name: 'Theme configuration' })
  ).toBeNull()
  view.unmount()
  expect(document.documentElement).not.toHaveAttribute('data-customer-ui')
})

it('leaves the administrator theme and header controls available', () => {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'admin', role: 10 })
  render(<AuthenticatedLayout>Content</AuthenticatedLayout>)
  expect(document.documentElement).not.toHaveAttribute('data-customer-ui')
  expect(screen.getByTestId('shell')).not.toHaveClass('customer-shell')
  expect(
    screen.getByRole('button', { name: 'Theme configuration' })
  ).toBeVisible()
})
