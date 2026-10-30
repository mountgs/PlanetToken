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

import { CUSTOMER_PRICE_COLUMNS } from '../lib/customer-price-columns'
import {
  getDynamicDisplayGroupRatio,
  isDynamicPricingModel,
} from '../lib/dynamic-price'
import { withPluginPricing } from '../lib/plugin-pricing'
import type { PricingModel } from '../types'
import { CustomerPriceCell, CustomerBillingMode } from './customer-price-cell'
import { DynamicPricingBreakdown } from './dynamic-pricing-breakdown'
import type { ModelPriceCellOptions } from './model-price-cell'

export function CustomerPricingDetails(props: {
  model: PricingModel
  options: ModelPriceCellOptions
}) {
  const { t } = useTranslation()
  if (props.model.billing_plugin_variants?.length) {
    return (
      <div className='space-y-6'>
        {props.model.billing_plugin_variants.map((variant) => (
          <section key={variant.plugin_key} className='space-y-3'>
            <h3 className='text-sm font-medium'>{variant.plugin_name}</h3>
            <CustomerPricingDetails
              model={withPluginPricing(props.model, variant)}
              options={props.options}
            />
          </section>
        ))}
      </div>
    )
  }
  return (
    <div className='space-y-4'>
      <CustomerBillingMode model={props.model} options={props.options} />
      {isDynamicPricingModel(props.model) ? (
        <DynamicPricingBreakdown
          billingExpr={props.model.billing_expr}
          usageSchema={props.model.billing_usage_schema}
          showExpression={false}
          priceOptions={{
            ...props.options,
            tokenUnit: props.options.tokenUnit ?? 'M',
            groupRatioMultiplier: getDynamicDisplayGroupRatio(
              props.model,
              props.options.selectedGroup
            ),
          }}
        />
      ) : (
        <dl className='grid gap-4 sm:grid-cols-2'>
          {CUSTOMER_PRICE_COLUMNS.map((price) => (
            <div key={price.type} className='rounded-lg border p-3'>
              <dt className='text-muted-foreground mb-2 text-xs'>
                {t(price.label)}
              </dt>
              <dd>
                <CustomerPriceCell
                  model={props.model}
                  type={price.type}
                  options={props.options}
                />
              </dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  )
}
