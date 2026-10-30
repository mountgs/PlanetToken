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
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
} from '@tanstack/react-router'
import { afterEach, expect, it } from 'vitest'

import { Route as PlaygroundRoute } from '@/routes/_authenticated/playground'
import { useAuthStore } from '@/stores/auth-store'

afterEach(() => {
  useAuthStore.getState().auth.reset()
  localStorage.clear()
})

async function openPlayground(enabled: boolean) {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'customer', role: 1 })
  localStorage.setItem(
    'status',
    JSON.stringify({
      SidebarModulesAdmin: JSON.stringify({
        chat: { enabled: true, playground: enabled },
      }),
    })
  )
  const root = createRootRoute()
  const playground = createRoute({
    getParentRoute: () => root,
    path: '/playground',
    beforeLoad: PlaygroundRoute.options.beforeLoad as () => void,
  })
  const dashboard = createRoute({
    getParentRoute: () => root,
    path: '/dashboard',
  })
  const router = createRouter({
    routeTree: root.addChildren([playground, dashboard]),
    history: createMemoryHistory({ initialEntries: ['/playground'] }),
  })
  await router.load()
  return router.state.location.pathname
}

it('allows an authenticated customer to open the enabled playground', async () => {
  expect(await openPlayground(true)).toBe('/playground')
})

it('redirects customers when the administrator disables the playground', async () => {
  expect(await openPlayground(false)).toBe('/dashboard')
})
