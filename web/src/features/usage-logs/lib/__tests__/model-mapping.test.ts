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
import assert from 'node:assert/strict'
import { describe, test } from 'vitest'

import type { UsageLog } from '../../data/schema'
import type { LogOtherData } from '../../types'
import { formatModelName, shouldRevealLogModelMapping } from '../format'

function makeLog(): UsageLog {
  return {
    id: 1,
    user_id: 1,
    created_at: 1,
    type: 2,
    content: '',
    username: 'user',
    token_name: 'token',
    model_name: 'gpt-5.5',
    quota: 1,
    prompt_tokens: 1,
    completion_tokens: 1,
    use_time: 1,
    is_stream: false,
    channel: 1,
    channel_name: '',
    token_id: 1,
    group: 'default',
    ip: '',
    other: JSON.stringify({
      is_model_mapped: true,
      upstream_model_name: 'deepseek-flash',
    }),
    request_id: 'req-1',
    upstream_request_id: '',
  }
}

describe('formatModelName mapping visibility', () => {
  test('hides the upstream model unless mapping is explicitly revealed', () => {
    const hidden = formatModelName(makeLog())
    assert.equal(hidden.name, 'gpt-5.5')
    assert.equal(hidden.isMapped, false)
    assert.equal(hidden.actualModel, undefined)

    const revealed = formatModelName(makeLog(), { revealMapping: true })
    assert.equal(revealed.name, 'gpt-5.5')
    assert.equal(revealed.isMapped, true)
    assert.equal(revealed.actualModel, 'deepseek-flash')
  })
})

describe('shouldRevealLogModelMapping', () => {
  const mapped: LogOtherData = {
    is_model_mapped: true,
    upstream_model_name: 'deepseek-flash',
  }

  test('hides mapping from ordinary users even if fields leak', () => {
    assert.equal(shouldRevealLogModelMapping(false, mapped), false)
  })

  test('shows mapping in the admin global view', () => {
    assert.equal(shouldRevealLogModelMapping(true, mapped), true)
  })
})
