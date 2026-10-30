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

import { StatusBadge } from '@/components/status-badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatCurrencyFromUSD } from '@/lib/currency'
import { formatNumber } from '@/lib/format'

import {
  formatTimestamp,
  getPaymentMethodName,
  getStatusConfig,
} from '../lib/billing'
import type { TopupRecord } from '../types'

export function BillingRecordsTable(props: { records: TopupRecord[] }) {
  const { t } = useTranslation()
  return (
    <div className='hidden min-w-0 sm:block'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Order Number')}</TableHead>
            <TableHead>{t('Created Time')}</TableHead>
            <TableHead>{t('Payment Method')}</TableHead>
            <TableHead>{t('Credited Amount')}</TableHead>
            <TableHead>{t('Paid Amount')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.records.map((record) => {
            const status = getStatusConfig(record.status)
            return (
              <TableRow key={record.id}>
                <TableCell>
                  <code className='text-xs'>{record.trade_no}</code>
                </TableCell>
                <TableCell className='text-muted-foreground'>
                  {formatTimestamp(record.create_time)}
                </TableCell>
                <TableCell>
                  {getPaymentMethodName(record.payment_method, t)}
                </TableCell>
                <TableCell className='tabular-nums'>
                  {formatCurrencyFromUSD(record.amount, {
                    digitsLarge: 2,
                    digitsSmall: 2,
                    abbreviate: false,
                  })}
                </TableCell>
                <TableCell className='tabular-nums'>
                  {formatNumber(record.money)}
                </TableCell>
                <TableCell>
                  <StatusBadge
                    label={t(status.label)}
                    variant={status.variant}
                    showDot
                    copyable={false}
                  />
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}
