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
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useAuthStore } from '@/stores/auth-store'

import { CustomerPricing } from '../customer-pricing'
import { usePricingData } from '../hooks/use-pricing-data'
import { getPricingKeys, type PricingKey } from '../lib/customer-pricing'

vi.mock('../hooks/use-pricing-data', () => ({ usePricingData: vi.fn() }))
vi.mock('../lib/customer-pricing', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/customer-pricing')>()),
  getPricingKeys: vi.fn(),
}))
vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; children: ReactNode; className?: string }) => (
    <a href={props.to} className={props.className}>
      {props.children}
    </a>
  ),
}))

const key: PricingKey = {
  id: 1,
  name: 'Production key',
  status: 1,
  group: 'default',
  auto_groups: null,
  model_limits_enabled: false,
  model_limits: '',
}
const model = {
  id: 1,
  model_name: 'test-model',
  model_ratio: 1,
  completion_ratio: 2,
  quota_type: 0,
  enable_groups: ['default', 'premium'],
  cache_ratio: 0.1,
  create_cache_ratio: 1.25,
  key: 'test-model',
  vendor_name: undefined,
  vendor_icon: undefined,
  vendor_description: undefined,
  group_ratio: { default: 1, premium: 0.5 },
}
const pricing = {
  models: [model],
  vendors: [],
  groupRatio: { default: 1, premium: 0.5 },
  usableGroup: {
    default: { desc: '', ratio: 1 },
    premium: { desc: '', ratio: 0.5 },
  },
  endpointMap: {},
  autoGroups: ['default', 'premium'],
  isLoading: false,
  error: null,
  refetch: vi.fn(),
  priceRate: 1,
  usdExchangeRate: 1,
}

beforeEach(() => {
  localStorage.clear()
  useAuthStore
    .getState()
    .auth.setUser({ id: 5, username: 'test', role: 1, group: 'default' })
  vi.mocked(usePricingData).mockReturnValue(pricing)
  vi.mocked(getPricingKeys).mockResolvedValue({
    keys: [key, { ...key, id: 2, name: 'Premium key', group: 'premium' }],
    maxAutoGroups: 2,
  })
})
afterEach(() => useAuthStore.getState().auth.reset())

function mount() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <CustomerPricing />
    </QueryClientProvider>
  )
}

it('changes displayed multiplier and remembers only the selected key ID', async () => {
  const user = userEvent.setup()
  mount()
  await waitFor(() =>
    expect(screen.getByRole('combobox', { name: 'API Key' })).toBeEnabled()
  )
  await user.selectOptions(
    screen.getByRole('combobox', { name: 'API Key' }),
    '2'
  )
  expect(screen.getByText('0.5×')).toBeVisible()
  expect(localStorage.getItem('customer-pricing-key:5')).toBe('2')
  const row = screen.getByRole('row', { name: /test-model/ })
  const cells = within(row).getAllByRole('cell')
  expect(cells.slice(2, 6).map((cell) => cell.textContent)).toEqual([
    '$1',
    '$2',
    '$0.1',
    '$1.25',
  ])
  for (const label of [
    'Input price',
    'Output price',
    'Cache Read',
    'Cache Write',
  ]) {
    expect(screen.getByRole('columnheader', { name: label })).toBeVisible()
  }
})

it('shows every pricing tier and cache write price using the selected key group', async () => {
  vi.mocked(usePricingData).mockReturnValue({
    ...pricing,
    models: [
      {
        ...model,
        billing_mode: 'tiered_expr',
        billing_expr:
          'len <= 200000 ? tier("standard", p * 2 + c * 4 + cr * 0.2 + cc * 2.5) : tier("long", p * 4 + c * 8 + cr * 0 + cc * 5)',
      },
    ],
  })
  localStorage.setItem('customer-pricing-key:5', '2')
  mount()
  const user = userEvent.setup()
  await user.click(await screen.findByRole('button', { name: 'test-model' }))
  const dialog = within(await screen.findByRole('dialog'))
  const table = within(dialog.getByRole('table'))
  const standard = within(table.getByRole('row', { name: /standard/ }))
  const long = within(table.getByRole('row', { name: /long/ }))
  expect(standard.getByText('$1.25')).toBeVisible()
  expect(long.getByText('$2.5')).toBeVisible()
  expect(long.getByText('$0')).toBeVisible()
  expect(dialog.queryByText('Raw expression')).toBeNull()
  expect(dialog.getByText('0.5×', { exact: false })).toBeVisible()
})

it('filters by billing method while retaining per-request price units', async () => {
  vi.mocked(usePricingData).mockReturnValue({
    ...pricing,
    models: [
      model,
      {
        ...model,
        id: 2,
        model_name: 'request-model',
        quota_type: 1,
        model_price: 0.25,
      },
    ],
  })
  mount()
  await waitFor(() =>
    expect(screen.getByRole('combobox', { name: 'API Key' })).toBeEnabled()
  )
  await userEvent
    .setup()
    .selectOptions(
      screen.getByRole('combobox', { name: 'Billing method' }),
      'Per Request'
    )
  expect(screen.queryByRole('button', { name: 'test-model' })).toBeNull()
  const row = screen.getByRole('row', { name: /request-model/ })
  expect(row).toHaveTextContent('0.25')
  expect(row).toHaveTextContent('request')
  const cells = within(row).getAllByRole('cell')
  expect(cells[1]).toHaveTextContent('0.25')
  expect(cells.slice(2, 6).map((cell) => cell.textContent)).toEqual([
    '—',
    '—',
    '—',
    '—',
  ])
})
it('restores a previously selected key on a new visit', async () => {
  localStorage.setItem('customer-pricing-key:5', '2')
  mount()
  await waitFor(() =>
    expect(screen.getByRole('combobox', { name: 'API Key' })).toHaveValue('2')
  )
  expect(screen.getByText('0.5×')).toBeVisible()
})
it('shows default prices and a creation link when the account has no keys', async () => {
  vi.mocked(getPricingKeys).mockResolvedValue({ keys: [], maxAutoGroups: 0 })
  mount()
  expect(
    await screen.findByText('No API keys yet. Showing default group prices.')
  ).toBeVisible()
  expect(screen.getByRole('link', { name: 'Create API Key' })).toHaveAttribute(
    'href',
    '/keys'
  )
  expect(screen.getByText('1×')).toBeVisible()
})
it('shows a retryable error instead of default prices when key loading fails', async () => {
  vi.mocked(getPricingKeys).mockRejectedValue(new Error('offline'))
  mount()
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Unable to load model pricing'
  )
  expect(
    screen.queryByText('No API keys yet. Showing default group prices.')
  ).toBeNull()
  expect(screen.getByRole('button', { name: 'Retry' })).toBeEnabled()
})
it('allows selecting an Auto pricing group and explains variable charges', async () => {
  const user = userEvent.setup()
  vi.mocked(getPricingKeys).mockResolvedValue({
    keys: [{ ...key, group: 'auto' }],
    maxAutoGroups: 2,
  })
  mount()
  const group = await screen.findByRole('combobox', { name: 'Pricing group' })
  await user.selectOptions(group, 'premium')
  expect(screen.getByText('0.5×')).toBeVisible()
  expect(screen.getByText(/This key uses automatic routing/)).toBeVisible()
})
it('does not show a fabricated price for an unavailable default group', async () => {
  vi.mocked(getPricingKeys).mockResolvedValue({ keys: [], maxAutoGroups: 0 })
  vi.mocked(usePricingData).mockReturnValue({
    ...pricing,
    groupRatio: { premium: 0.5 },
  })
  mount()
  expect(
    await screen.findByText(
      'Pricing is unavailable for this group. Please contact support.'
    )
  ).toBeVisible()
  expect(screen.queryByRole('table')).toBeNull()
})
it('opens a model using the keyboard and retains customer group pricing', async () => {
  const user = userEvent.setup()
  mount()
  const modelButton = await screen.findByRole('button', { name: 'test-model' })
  modelButton.focus()
  await user.keyboard('{Enter}')
  expect(await screen.findByRole('dialog')).toHaveTextContent('test-model')
  expect(screen.getByRole('dialog')).not.toHaveTextContent('Base Price')
})
it('keeps long key names in a width-constrained native selector on narrow screens', async () => {
  vi.mocked(getPricingKeys).mockResolvedValue({
    keys: [{ ...key, name: 'Customer production key '.repeat(30) }],
    maxAutoGroups: 0,
  })
  mount()
  const selector = screen.getByRole('combobox', { name: 'API Key' })
  await waitFor(() => expect(selector).toBeEnabled())
  expect(selector).toHaveClass('w-full', 'min-w-0')
  expect(selector.parentElement).toHaveClass('w-full', 'max-w-md')
})

it('shows an input-priced cache read and leaves unconfigured cache writes empty', async () => {
  vi.mocked(usePricingData).mockReturnValue({
    ...pricing,
    models: [
      {
        ...model,
        billing_mode: 'tiered_expr',
        billing_expr: 'tier("base", p * 2 + c * 4 + cr * 2)',
      },
    ],
  })
  mount()
  await waitFor(() =>
    expect(screen.getByRole('combobox', { name: 'API Key' })).toBeEnabled()
  )
  const row = screen.getByRole('row', { name: /test-model/ })
  const cells = within(row).getAllByRole('cell')
  expect(cells.slice(2, 6).map((cell) => cell.textContent)).toEqual([
    '$2',
    '$4',
    '$2',
    '—',
  ])
})
