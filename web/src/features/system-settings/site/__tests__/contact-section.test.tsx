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
import {
  createRouter,
  createRootRoute,
  createMemoryHistory,
  RouterContextProvider,
} from '@tanstack/react-router'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { useState } from 'react'
import { afterEach, expect, it, vi } from 'vitest'

import { Contact } from '@/features/contact'
import zh from '@/i18n/locales/zh.json'
import { api } from '@/lib/api'

import { SettingsPageProvider } from '../../components/settings-page-context'
import { ContactSection } from '../contact-section'

function Workspace() {
  const [actions, setActions] = useState<HTMLDivElement | null>(null)
  return (
    <SettingsPageProvider actionsContainer={actions}>
      <div ref={setActions} />
      <ContactSection defaultValue='Contact our team.' />
    </SettingsPageProvider>
  )
}
let client: QueryClient
function mount() {
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const router = createRouter({
    routeTree: createRootRoute(),
    history: createMemoryHistory(),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterContextProvider router={router}>
        <Workspace />
        <Contact />
      </RouterContextProvider>
    </QueryClientProvider>
  )
  return userEvent.setup()
}
afterEach(async () => {
  client?.clear()
  vi.restoreAllMocks()
  await i18next.changeLanguage('en')
})

it('previews inserted QR images and saves them so the customer view refreshes', async () => {
  let stored = 'Contact our team.'
  vi.spyOn(api, 'get').mockImplementation(async () => ({
    data: { success: true, data: stored },
  }))
  const put = vi.spyOn(api, 'put').mockImplementation(async (_url, request) => {
    if (
      !request ||
      typeof request !== 'object' ||
      !('value' in request) ||
      typeof request.value !== 'string'
    ) {
      throw new Error('Unexpected settings request')
    }
    stored = request.value
    return { data: { success: true } }
  })
  const user = mount()
  await user.type(
    screen.getByRole('textbox', { name: 'Contact image URL' }),
    'https://example.com/qr.png'
  )
  await user.click(screen.getByRole('button', { name: 'Insert image' }))
  const preview = within(screen.getByRole('region', { name: 'Preview' }))
  expect(preview.getByRole('img', { name: 'Contact image' })).toHaveAttribute(
    'src',
    'https://example.com/qr.png'
  )
  await user.click(screen.getByRole('button', { name: 'Save Changes' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledWith('/api/option/', {
      key: 'Contact',
      value:
        'Contact our team.\n\n![Contact image](https://example.com/qr.png)\n',
    })
  )
  await waitFor(() =>
    expect(screen.getAllByRole('img', { name: 'Contact image' })).toHaveLength(
      2
    )
  )
  expect(screen.getByRole('button', { name: 'Reset' })).toBeDisabled()
})

it('rejects unsafe image addresses and resets an unsaved draft to stored content', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: '' } })
  const user = mount()
  const url = screen.getByRole('textbox', { name: 'Contact image URL' })
  await user.type(url, 'javascript:alert(1)')
  await user.click(screen.getByRole('button', { name: 'Insert image' }))
  expect(url).toHaveAttribute('aria-invalid', 'true')
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Enter a valid HTTP(S) image URL without credentials.'
  )
  expect(
    screen.getByRole('textbox', { name: 'Contact page content' })
  ).toHaveValue('Contact our team.')
  await user.type(
    screen.getByRole('textbox', { name: 'Contact page content' }),
    ' Draft'
  )
  await user.click(screen.getByRole('button', { name: 'Reset' }))
  expect(
    screen.getByRole('textbox', { name: 'Contact page content' })
  ).toHaveValue('Contact our team.')
})

it('keeps the contact draft after a failed save and allows retry', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: '' } })
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValueOnce({
      data: { success: false, message: 'Unable to save contact settings' },
    })
    .mockResolvedValue({ data: { success: true } })
  const user = mount()
  const content = screen.getByRole('textbox', { name: 'Contact page content' })
  await user.clear(content)
  await user.type(content, 'Weekday support')
  await user.click(screen.getByRole('button', { name: 'Save Changes' }))
  await waitFor(() =>
    expect(screen.getByRole('button', { name: 'Save Changes' })).toBeEnabled()
  )
  expect(content).toHaveValue('Weekday support')
  expect(screen.getByRole('button', { name: 'Reset' })).toBeEnabled()
  await user.click(screen.getByRole('button', { name: 'Save Changes' }))
  await waitFor(() => expect(put).toHaveBeenCalledTimes(2))
  await waitFor(() =>
    expect(screen.getByRole('button', { name: 'Reset' })).toBeDisabled()
  )
  expect(content).toHaveValue('Weekday support')
})

it('uses the Chinese contact editor labels from the shipped locale resource', async () => {
  i18next.addResourceBundle('zh', 'translation', zh.translation)
  await i18next.changeLanguage('zh')
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: '' } })
  mount()
  expect(
    screen.getByRole('textbox', { name: '联系我们页面内容' })
  ).toBeVisible()
  expect(screen.getByRole('textbox', { name: '联系图片链接' })).toBeVisible()
  expect(screen.getByRole('button', { name: '插入图片' })).toBeDisabled()
})
