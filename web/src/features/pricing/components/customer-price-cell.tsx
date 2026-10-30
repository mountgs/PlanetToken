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
import { useTranslation } from 'react-i18next'

import { useSystemConfigStore } from '@/stores/system-config-store'

import { useBillingTime } from '../hooks/use-billing-time'
import { CUSTOMER_PRICE_COLUMNS } from '../lib/customer-price-columns'
import {
  getDynamicDisplayGroupRatio,
  getDynamicPricingSummary,
  isUnconfiguredTaskUsageModel,
} from '../lib/dynamic-price'
import { isTokenBasedModel } from '../lib/model-helpers'
import { formatPrice, stripTrailingZeros } from '../lib/price'
import type { PricingModel, PriceType } from '../types'
import { ModelBillingModeBadge } from './model-billing-mode-badge'
import { ModelPriceCell, type ModelPriceCellOptions } from './model-price-cell'

export function CustomerBillingMode(props: {
  model: PricingModel
  options?: ModelPriceCellOptions
}) {
  const { t } = useTranslation()
  const billingTime = useBillingTime(props.model.billing_expr)
  const summary = getDynamicPricingSummary(props.model, {
    tokenUnit: 'M',
    now: billingTime === undefined ? undefined : new Date(billingTime),
  })
  const showSeparatePrice = summary
    ? summary.isSpecialExpression ||
      summary.isTaskUsage ||
      summary.isMixedBilling ||
      summary.entries.some(
        (entry) => entry.unit === 'request' || entry.unit === 'image'
      )
    : isUnconfiguredTaskUsageModel(props.model) ||
      !isTokenBasedModel(props.model)
  return (
    <div className='space-y-1'>
      <ModelBillingModeBadge model={props.model} className='-ml-1.5' />
      {showSeparatePrice && (
        <ModelPriceCell
          model={props.model}
          options={props.options}
          showExpression={false}
        />
      )}
      {summary && (
        <div className='text-muted-foreground space-y-1 text-xs'>
          {summary.tierCount > 1 && (
            <div>
              {t('Tiered pricing')} ·{' '}
              {t('{{count}} tiers', { count: summary.tierCount })}
            </div>
          )}
          {summary.isTimePricing && <div>{t('Time-based')}</div>}
          {summary.hasRequestRules && <div>{t('Request rules')}</div>}
        </div>
      )}
    </div>
  )
}

export function CustomerPriceCell(props: {
  model: PricingModel
  type: Extract<PriceType, 'input' | 'output' | 'cache' | 'create_cache'>
  options: ModelPriceCellOptions
}) {
  const { t } = useTranslation()
  // Subscribe because price formatters read the current display currency.
  useSystemConfigStore((state) => state.config.currency)
  const billingTime = useBillingTime(props.model.billing_expr)
  const summary = getDynamicPricingSummary(props.model, {
    ...props.options,
    tokenUnit: props.options.tokenUnit ?? 'M',
    groupRatioMultiplier: getDynamicDisplayGroupRatio(
      props.model,
      props.options.selectedGroup
    ),
    includeInputPriceCacheRead: true,
    now: billingTime === undefined ? undefined : new Date(billingTime),
  })
  const field = CUSTOMER_PRICE_COLUMNS.find(
    (column) => column.type === props.type
  )?.field
  if (summary) {
    if (
      summary.isSpecialExpression ||
      summary.isTaskUsage ||
      summary.isMixedBilling
    ) {
      return <span className='text-muted-foreground'>—</span>
    }
    const entries = summary.entries.filter((entry) => {
      if (props.type === 'create_cache') {
        return entry.field.startsWith('cacheCreate')
      }
      return entry.field === field
    })
    if (!entries.length) return <span className='text-muted-foreground'>—</span>
    return (
      <div className='space-y-1'>
        {entries.map((entry) => (
          <div key={entry.key}>
            {entries.length > 1 && (
              <div className='text-muted-foreground text-xs'>
                {t(entry.shortLabel)}
              </div>
            )}
            <span className='font-mono text-sm tabular-nums'>
              {stripTrailingZeros(entry.formattedRange ?? entry.formatted)}
            </span>
          </div>
        ))}
      </div>
    )
  }
  if (
    isUnconfiguredTaskUsageModel(props.model) ||
    !isTokenBasedModel(props.model)
  ) {
    return <span className='text-muted-foreground'>—</span>
  }
  const value =
    props.type === 'cache'
      ? props.model.cache_ratio
      : props.model.create_cache_ratio
  if (
    (props.type === 'cache' || props.type === 'create_cache') &&
    value == null
  ) {
    return <span className='text-muted-foreground'>—</span>
  }
  if (!Number.isFinite(props.model.model_ratio)) {
    return <span>{t('Unset price')}</span>
  }
  return (
    <span className='font-mono text-sm tabular-nums'>
      {stripTrailingZeros(
        formatPrice(
          props.model,
          props.type,
          props.options.tokenUnit ?? 'M',
          props.options.showRechargePrice,
          props.options.priceRate,
          props.options.usdExchangeRate,
          props.options.selectedGroup
        )
      )}
    </span>
  )
}
