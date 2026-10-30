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
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getApiKeys, getTokenAutoGroups } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'

import type { PricingModel } from '../../types'
import {
  getCustomerModels,
  getPricingGroups,
  getPricingKeys,
  readPricingKeyId,
  resolvePricingKey,
  savePricingKeyId,
  type PricingKey,
} from '../customer-pricing'
import { formatPrice } from '../price'

vi.mock('@/features/keys/api', () => ({
  getApiKeys: vi.fn(),
  getTokenAutoGroups: vi.fn(),
}))
const key: PricingKey = {
  id: 1,
  name: 'Production',
  status: 1,
  group: 'premium',
  auto_groups: null,
  model_limits_enabled: false,
  model_limits: '',
}
const model: PricingModel = {
  id: 1,
  model_name: 'gpt-test',
  quota_type: 0,
  model_ratio: 2,
  completion_ratio: 3,
  enable_groups: ['premium', 'default'],
  group_ratio: { default: 1, premium: 0.5 },
}
const groups = {
  key,
  usableGroups: ['premium', 'default'],
  ratios: { premium: 0.5, default: 1 },
  autoGroups: ['premium', 'default'],
  maxAutoGroups: 2,
}

beforeEach(() => {
  vi.resetAllMocks()
  localStorage.clear()
})

describe('customer pricing selection', () => {
  it('restores the remembered key even when it is inactive', () => {
    const inactive = { ...key, id: 2, status: 2 }
    expect(resolvePricingKey([key, inactive], 2)).toEqual(inactive)
  })
  it('falls back to an active key when the remembered key was deleted', () => {
    expect(resolvePricingKey([{ ...key, id: 3, status: 2 }, key], 99)?.id).toBe(
      1
    )
  })
  it('uses default pricing when there are no keys even for a premium account', () => {
    expect(
      getPricingGroups({ ...groups, key: null, userGroup: 'premium' })
    ).toEqual(['default'])
  })
  it('inherits the user group when the key has no explicit group', () => {
    expect(
      getPricingGroups({
        ...groups,
        key: { ...key, group: '' },
        userGroup: 'premium',
      })
    ).toEqual(['premium'])
  })
  it('preserves a valid free group multiplier', () => {
    expect(getPricingGroups({ ...groups, ratios: { premium: 0 } })).toEqual([
      'premium',
    ])
  })
  it('does not invent a price for a missing or no longer permitted group', () => {
    expect(getPricingGroups({ ...groups, usableGroups: ['default'] })).toEqual(
      []
    )
    expect(getPricingGroups({ ...groups, ratios: { default: 1 } })).toEqual([])
  })
  it('inherits global Auto groups only for an unspecified token selection', () => {
    expect(
      getPricingGroups({ ...groups, key: { ...key, group: 'auto' } })
    ).toEqual(['premium', 'default'])
  })
  it('keeps an explicitly empty Auto selection empty', () => {
    expect(
      getPricingGroups({
        ...groups,
        key: { ...key, group: 'auto', auto_groups: [] },
      })
    ).toEqual([])
  })
  it('filters permissions before applying the Auto selection limit', () => {
    expect(
      getPricingGroups({
        ...groups,
        maxAutoGroups: 1,
        key: {
          ...key,
          group: 'auto',
          auto_groups: ['removed', 'premium', 'default'],
        },
      })
    ).toEqual(['premium'])
  })
  it('applies key model limits and excludes unavailable group models', () => {
    const models = [
      model,
      { ...model, id: 2, model_name: 'other' },
      { ...model, id: 3, enable_groups: ['default'] },
    ]
    expect(
      getCustomerModels(
        models,
        { ...key, model_limits_enabled: true, model_limits: 'gpt-test' },
        'premium',
        0.5
      ).map((item) => item.id)
    ).toEqual([1])
  })
  it('does not treat an empty enabled model allowlist as unrestricted', () => {
    expect(
      getCustomerModels(
        [model],
        { ...key, model_limits_enabled: true, model_limits: '' },
        'premium',
        0.5
      )
    ).toEqual([])
  })
  it('supports all-group models and scopes their price to the selected group', () => {
    const [scoped] = getCustomerModels(
      [{ ...model, enable_groups: ['all'] }],
      key,
      'premium',
      0.5
    )
    expect(scoped.enable_groups).toEqual(['premium'])
    expect(formatPrice(scoped, 'input', 'M', false, 1, 1, 'premium')).toContain(
      '2'
    )
    expect(
      formatPrice(scoped, 'output', 'M', false, 1, 1, 'premium')
    ).toContain('6')
  })
  it('stores only the key ID separately for each account', () => {
    savePricingKeyId(10, 1)
    savePricingKeyId(20, 2)
    expect(readPricingKeyId(10)).toBe(1)
    expect(readPricingKeyId(20)).toBe(2)
    expect(localStorage.getItem('customer-pricing-key:10')).toBe('1')
  })
  it('ignores malformed storage and remains usable when storage is blocked', () => {
    localStorage.setItem('customer-pricing-key:10', 'NaN')
    expect(readPricingKeyId(10)).toBeNull()
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    expect(readPricingKeyId(10)).toBeNull()
  })
})

describe('pricing key metadata loading', () => {
  const apiKey: ApiKey = {
    ...key,
    key: 'masked-secret',
    remain_quota: 1,
    used_quota: 0,
    unlimited_quota: false,
    expired_time: -1,
    created_time: 0,
    accessed_time: 0,
    allow_ips: '',
    allow_user_quota_query: false,
    cross_group_retry: false,
  }
  it('loads every page and excludes credentials from cached metadata', async () => {
    vi.mocked(getApiKeys)
      .mockResolvedValueOnce({
        success: true,
        data: { items: [apiKey], total: 2, page: 1, page_size: 1 },
      })
      .mockResolvedValueOnce({
        success: true,
        data: {
          items: [{ ...apiKey, id: 2 }],
          total: 2,
          page: 2,
          page_size: 1,
        },
      })
    const data = await getPricingKeys()
    expect(data.keys.map((item) => item.id)).toEqual([1, 2])
    expect(JSON.stringify(data)).not.toContain('masked-secret')
    expect(data.keys[0]).not.toHaveProperty('key')
  })
  it('rejects partial results when a later page fails', async () => {
    vi.mocked(getApiKeys)
      .mockResolvedValueOnce({
        success: true,
        data: { items: [apiKey], total: 2, page: 1, page_size: 1 },
      })
      .mockResolvedValueOnce({ success: false, message: 'failed' })
    await expect(getPricingKeys()).rejects.toThrow()
  })
  it('does not loop when the server returns an empty unfinished page', async () => {
    vi.mocked(getApiKeys).mockResolvedValue({
      success: true,
      data: { items: [], total: 2, page: 1, page_size: 100 },
    })
    await expect(getPricingKeys()).rejects.toThrow()
  })
  it('loads current Auto limits for an Auto key', async () => {
    vi.mocked(getApiKeys).mockResolvedValue({
      success: true,
      data: {
        items: [{ ...apiKey, group: 'auto' }],
        total: 1,
        page: 1,
        page_size: 100,
      },
    })
    vi.mocked(getTokenAutoGroups).mockResolvedValue({
      success: true,
      data: { groups: ['premium'], max_count: 1 },
    })
    expect((await getPricingKeys()).maxAutoGroups).toBe(1)
  })
})
