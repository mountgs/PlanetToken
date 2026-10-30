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
import { getApiKeys, getTokenAutoGroups } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'
import { requireServerSuccess } from '@/lib/server-error-message'

import type { PricingModel } from '../types'

// Deliberately exclude even masked credentials from this feature's cache/storage.
export type PricingKey = Pick<
  ApiKey,
  | 'id'
  | 'name'
  | 'status'
  | 'group'
  | 'auto_groups'
  | 'model_limits_enabled'
  | 'model_limits'
>

export async function getPricingKeys(): Promise<{
  keys: PricingKey[]
  maxAutoGroups: number
}> {
  const keys: PricingKey[] = []
  let page = 1
  for (;;) {
    const response = requireServerSuccess(
      await getApiKeys({ p: page, size: 100 })
    )
    if (!response.data) throw new Error('Failed to load API keys')
    const items = response.data.items
    if (items.length === 0 && keys.length < response.data.total) {
      throw new Error('Failed to load API keys')
    }
    for (const item of items) {
      keys.push({
        id: item.id,
        name: item.name,
        status: item.status,
        group: item.group,
        auto_groups: item.auto_groups,
        model_limits_enabled: item.model_limits_enabled,
        model_limits: item.model_limits,
      })
    }
    if (keys.length >= response.data.total) break
    page += 1
  }
  let maxAutoGroups = 0
  if (keys.some((key) => key.group === 'auto')) {
    const response = requireServerSuccess(await getTokenAutoGroups())
    if (!response.data) throw new Error('Failed to load API keys')
    maxAutoGroups = response.data.max_count
  }
  return { keys, maxAutoGroups }
}

export function resolvePricingKey(
  keys: PricingKey[],
  selectedId: number | null
): PricingKey | null {
  return (
    keys.find((key) => key.id === selectedId) ??
    keys.find((key) => key.status === 1) ??
    keys[0] ??
    null
  )
}

export function getPricingGroups(options: {
  key: PricingKey | null
  userGroup?: string
  usableGroups: string[]
  ratios: Record<string, number>
  autoGroups: string[]
  maxAutoGroups: number
}): string[] {
  const group = options.key
    ? options.key.group || options.userGroup || 'default'
    : 'default'
  const permitted = (value: string) =>
    options.usableGroups.includes(value) &&
    Number.isFinite(options.ratios[value]) &&
    options.ratios[value] >= 0
  if (group !== 'auto') return permitted(group) ? [group] : []
  const explicit = options.key?.auto_groups
  if (explicit == null) return options.autoGroups.filter(permitted)
  return [...new Set(explicit)]
    .filter(permitted)
    .slice(0, options.maxAutoGroups)
}

export function getCustomerModels(
  models: PricingModel[],
  key: PricingKey | null,
  group: string,
  ratio: number
): PricingModel[] {
  const limits = new Set(
    (key?.model_limits ?? '')
      .split(',')
      .map((name) => name.trim())
      .filter(Boolean)
  )
  return models
    .filter(
      (model) =>
        model.enable_groups.includes(group) ||
        model.enable_groups.includes('all')
    )
    .filter(
      (model) => !key?.model_limits_enabled || limits.has(model.model_name)
    )
    .map((model) => ({
      ...model,
      enable_groups: [group],
      group_ratio: { [group]: ratio },
    }))
}

export function readPricingKeyId(userId: number): number | null {
  try {
    const value = localStorage.getItem(`customer-pricing-key:${userId}`)
    const id = Number(value)
    return value !== null && Number.isSafeInteger(id) && id > 0 ? id : null
  } catch {
    return null
  }
}

export function savePricingKeyId(userId: number, keyId: number): void {
  try {
    localStorage.setItem(`customer-pricing-key:${userId}`, String(keyId))
  } catch {
    /* Storage may be unavailable in private browsing. */
  }
}
