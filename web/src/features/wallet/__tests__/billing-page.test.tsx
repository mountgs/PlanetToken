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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'

import { Billing } from '../billing'
import { useBillingHistory } from '../hooks/use-billing-history'
import type { TopupRecord } from '../types'

vi.mock('../hooks/use-billing-history', () => ({ useBillingHistory: vi.fn() }))
const history = {
  records: [],
  error: false,
  total: 0,
  page: 1,
  pageSize: 10,
  keyword: '',
  loading: false,
  completing: false,
  isAdmin: false,
  handlePageChange: vi.fn(),
  handlePageSizeChange: vi.fn(),
  handleSearch: vi.fn(),
  handleCompleteOrder: vi.fn(),
  refresh: vi.fn(),
}
beforeEach(() => vi.mocked(useBillingHistory).mockReturnValue(history))

const record: TopupRecord = {
  id: 1,
  user_id: 5,
  amount: 10,
  money: 72,
  trade_no: 'ORDER-001',
  payment_method: 'alipay',
  create_time: 1700000000,
  status: 'success',
}

it('shows customer orders with credited and paid amounts in separate columns', () => {
  vi.mocked(useBillingHistory).mockReturnValue({
    ...history,
    records: [record],
    total: 1,
  })
  render(<Billing />)
  const table = within(screen.getByRole('table'))
  expect(
    table.getByRole('columnheader', { name: 'Credited Amount' })
  ).toBeVisible()
  expect(table.getByRole('columnheader', { name: 'Paid Amount' })).toBeVisible()
  expect(table.getByRole('cell', { name: 'ORDER-001' })).toBeVisible()
  expect(table.getByRole('cell', { name: '72' })).toBeVisible()
  expect(table.getByRole('cell', { name: 'Success' })).toBeVisible()
  expect(screen.queryByText('User ID: 5')).toBeNull()
})

it('hides stale orders after a failed refresh', () => {
  vi.mocked(useBillingHistory).mockReturnValue({
    ...history,
    records: [record],
    error: true,
  })
  render(<Billing />)
  expect(screen.getByRole('alert')).toBeVisible()
  expect(screen.queryByText('ORDER-001')).toBeNull()
})

it('renders billing as a full page without opening a modal', () => {
  render(<Billing />)
  expect(screen.getByRole('heading', { name: 'Recharge Bills' })).toBeVisible()
  expect(screen.getByText('No billing records found')).toBeVisible()
  expect(screen.queryByRole('dialog')).toBeNull()
})
it('shows failed loads as retryable errors instead of an empty history', async () => {
  vi.mocked(useBillingHistory).mockReturnValue({ ...history, error: true })
  render(<Billing />)
  expect(screen.getByRole('alert')).toHaveTextContent(
    'Failed to load billing history'
  )
  expect(screen.queryByText('No billing records found')).toBeNull()
  await userEvent.setup().click(screen.getByRole('button', { name: 'Retry' }))
  expect(history.refresh).toHaveBeenCalled()
})
it('searches orders using the existing billing hook', async () => {
  render(<Billing />)
  await userEvent
    .setup()
    .type(
      screen.getByRole('textbox', { name: 'Search by order number...' }),
      'A'
    )
  expect(history.handleSearch).toHaveBeenCalledWith('A')
})
