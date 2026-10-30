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
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect } from '@/components/ui/native-select'
import { toIntlLocale } from '@/i18n/languages'
import { useAuthStore } from '@/stores/auth-store'

import { CustomerPricingDetails } from './components/customer-pricing-details'
import { PricingTable } from './components/pricing-table'
import { FILTER_ALL } from './constants'
import { usePricingData } from './hooks/use-pricing-data'
import { getBillingModeLabelKey } from './lib/billing-mode'
import {
  getCustomerModels,
  getPricingGroups,
  getPricingKeys,
  readPricingKeyId,
  resolvePricingKey,
  savePricingKeyId,
} from './lib/customer-pricing'
import { filterBySearch, filterByVendor } from './lib/filters'

export function CustomerPricing() {
  const user = useAuthStore((state) => state.auth.user)
  // Remount account-specific selection state when the authenticated account changes.
  return (
    <CustomerPricingContent
      key={user?.id}
      userId={user?.id ?? 0}
      userGroup={user?.group}
    />
  )
}

function CustomerPricingContent(props: { userId: number; userGroup?: string }) {
  const { t, i18n } = useTranslation()
  const multiplier = new Intl.NumberFormat(toIntlLocale(i18n.language), {
    maximumSignificantDigits: 12,
  })
  const pricing = usePricingData()
  const keys = useQuery({
    queryKey: ['customer-pricing-keys', props.userId],
    queryFn: getPricingKeys,
    staleTime: 0,
  })
  const [selectedId, setSelectedId] = useState<number | null>(() =>
    readPricingKeyId(props.userId)
  )
  const [autoGroup, setAutoGroup] = useState('')
  const [search, setSearch] = useState('')
  const [vendor, setVendor] = useState(FILTER_ALL)
  const [billingMode, setBillingMode] = useState(FILTER_ALL)
  const [selectedModel, setSelectedModel] = useState<string | null>(null)
  const key = resolvePricingKey(keys.data?.keys ?? [], selectedId)
  const groups = getPricingGroups({
    key,
    userGroup: props.userGroup,
    usableGroups: Object.keys(pricing.usableGroup),
    ratios: pricing.groupRatio,
    autoGroups: pricing.autoGroups,
    maxAutoGroups: keys.data?.maxAutoGroups ?? 0,
  })
  const group = groups.includes(autoGroup) ? autoGroup : groups[0]
  const ratio = group === undefined ? undefined : pricing.groupRatio[group]
  const models =
    group === undefined || ratio === undefined
      ? []
      : getCustomerModels(pricing.models, key, group, ratio)
  const filteredModels = filterByVendor(
    filterBySearch(models, search),
    vendor
  ).filter(
    (item) =>
      billingMode === FILTER_ALL || getBillingModeLabelKey(item) === billingMode
  )
  const model = models.find((item) => item.model_name === selectedModel)
  const priceOptions = {
    selectedGroup: group,
    priceRate: pricing.priceRate,
    usdExchangeRate: pricing.usdExchangeRate,
    tokenUnit: 'M' as const,
  }
  const loading = pricing.isLoading || keys.isPending || keys.isFetching
  const failed = Boolean(pricing.error || keys.error)

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Models & Pricing')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <p className='text-muted-foreground mb-6 text-sm'>
          {t('Compare available models and prices for your API key.')}
        </p>
        <div className='bg-card mb-5 flex flex-wrap items-end gap-4 rounded-xl border p-4 sm:p-5'>
          <label className='grid w-full min-w-0 gap-2 text-sm font-medium sm:w-auto sm:flex-1'>
            {t('API Key')}
            <NativeSelect
              className='w-full max-w-md'
              aria-label={t('API Key')}
              disabled={loading || failed || !key}
              value={key?.id ?? ''}
              onChange={(event) => {
                const id = Number(event.target.value)
                setSelectedId(id)
                savePricingKeyId(props.userId, id)
                setAutoGroup('')
                setVendor(FILTER_ALL)
              }}
            >
              {!key && (
                <option value=''>
                  {t(loading ? 'Loading...' : 'Default group pricing')}
                </option>
              )}
              {(keys.data?.keys ?? []).map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name || `#${item.id}`}
                  {item.status !== 1 ? ` · ${t('Inactive')}` : ''}
                </option>
              ))}
            </NativeSelect>
          </label>
          {groups.length > 1 && (
            <label className='grid gap-2 text-sm font-medium'>
              {t('Pricing group')}
              <NativeSelect
                aria-label={t('Pricing group')}
                value={group}
                onChange={(event) => setAutoGroup(event.target.value)}
              >
                {groups.map((value) => (
                  <option key={value} value={value}>
                    {value}
                  </option>
                ))}
              </NativeSelect>
            </label>
          )}
          {!loading && !failed && (
            <div
              className='bg-muted max-w-full rounded-lg px-4 py-2 text-sm wrap-anywhere'
              aria-live='polite'
            >
              {t('Group')}: <strong>{group ?? '—'}</strong>
              <span className='text-muted-foreground mx-3'>/</span>
              {t('Multiplier')}:{' '}
              <strong>
                {ratio === undefined ? '—' : `${multiplier.format(ratio)}×`}
              </strong>
            </div>
          )}
        </div>
        {!loading && !failed && !key && (
          <p className='text-muted-foreground mb-4 text-sm'>
            {t('No API keys yet. Showing default group prices.')}{' '}
            <Link to='/keys' className='text-primary underline'>
              {t('Create API Key')}
            </Link>
          </p>
        )}
        {!loading && !failed && key?.group === 'auto' && (
          <p className='text-muted-foreground mb-4 text-sm'>
            {t(
              'This key uses automatic routing. Prices below apply to the selected group; actual charges depend on the routed group.'
            )}
          </p>
        )}
        {!loading && !failed && key && key.status !== 1 && (
          <p role='status' className='text-muted-foreground mb-4 text-sm'>
            {t(
              'This API key is inactive. Prices are for reference; enable the key before use.'
            )}
          </p>
        )}
        {failed ? (
          <div role='alert' className='bg-card rounded-xl border p-6'>
            <p>{t('Unable to load model pricing. Please try again.')}</p>
            <Button
              className='mt-4'
              variant='outline'
              onClick={() => {
                void pricing.refetch()
                void keys.refetch()
              }}
            >
              {t('Retry')}
            </Button>
          </div>
        ) : (
          <div>
            {!loading && !group ? (
              <p role='status' className='bg-card rounded-xl border p-6'>
                {t(
                  'Pricing is unavailable for this group. Please contact support.'
                )}
              </p>
            ) : (
              <div className='bg-card min-w-0 rounded-xl border p-4 sm:p-5'>
                <div className='mb-5 flex flex-wrap items-center gap-3'>
                  <Input
                    aria-label={t('Search models')}
                    placeholder={t('Search models')}
                    className='min-w-0 flex-1 sm:max-w-sm'
                    value={search}
                    onChange={(event) => setSearch(event.target.value)}
                  />
                  <NativeSelect
                    aria-label={t('Vendor')}
                    value={vendor}
                    onChange={(event) => setVendor(event.target.value)}
                  >
                    <option value={FILTER_ALL}>{t('All vendors')}</option>
                    {[
                      ...new Set(
                        models
                          .map((item) => item.vendor_name)
                          .filter((name): name is string => Boolean(name))
                      ),
                    ].map((name) => (
                      <option key={name} value={name}>
                        {name}
                      </option>
                    ))}
                  </NativeSelect>
                  <NativeSelect
                    aria-label={t('Billing method')}
                    value={billingMode}
                    onChange={(event) => setBillingMode(event.target.value)}
                  >
                    <option value={FILTER_ALL}>
                      {t('All billing methods')}
                    </option>
                    {(
                      [
                        'Token-based',
                        'Per Request',
                        'Dynamic Pricing',
                        'Task billing',
                      ] as const
                    ).map((mode) => (
                      <option key={mode} value={mode}>
                        {t(mode)}
                      </option>
                    ))}
                  </NativeSelect>
                  <span className='text-muted-foreground text-xs'>
                    {t('Prices per 1M tokens unless otherwise stated.')}
                  </span>
                </div>
                <p className='text-muted-foreground mb-4 text-xs'>
                  {t(
                    'Tiered models show starting-tier prices; open a model to see every tier and condition. Time-based prices reflect the current period.'
                  )}
                </p>
                <PricingTable
                  key={`${key?.id ?? 'default'}:${group}:${search}:${vendor}:${billingMode}`}
                  customer
                  models={filteredModels}
                  isLoading={loading}
                  {...priceOptions}
                  onModelClick={setSelectedModel}
                />
              </div>
            )}
          </div>
        )}
        {model && !failed && (
          <Dialog
            open
            onOpenChange={(open) => {
              if (!open) setSelectedModel(null)
            }}
            title={model.model_name}
            description={model.description || t('Model pricing details')}
          >
            <div className='space-y-5'>
              <p className='text-muted-foreground text-sm'>
                {t('Group')}: {group} · {t('Multiplier')}:{' '}
                {multiplier.format(ratio ?? 0)}×
              </p>
              <CustomerPricingDetails model={model} options={priceOptions} />
              <p className='text-muted-foreground text-xs'>
                {t('Prices per 1M tokens unless otherwise stated.')}
              </p>
              <div className='flex flex-wrap gap-2'>
                {model.supported_endpoint_types?.map((endpoint) => (
                  <span
                    key={endpoint}
                    className='bg-muted rounded-md px-2 py-1 text-xs'
                  >
                    {endpoint}
                  </span>
                ))}
              </div>
            </div>
          </Dialog>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
