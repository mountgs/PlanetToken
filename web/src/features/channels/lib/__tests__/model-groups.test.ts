import { describe, expect, it } from 'vitest'

import { channelSchema } from '../../types'
import {
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
  transformFormDataToUpdatePayload,
} from '../channel-form'

function savedChannel() {
  return channelSchema.parse({
    id: 7,
    type: 57,
    key: '',
    status: 1,
    name: 'Codex',
    created_time: 0,
    test_time: 0,
    response_time: 0,
    balance_updated_time: 0,
    group: 'default,premium',
    models: 'gpt-6.1-mini,gpt-6.1-sol',
  })
}

describe('Codex channel model groups', () => {
  it('expands legacy channels with the original models in every group', () => {
    const draft = transformChannelToFormDefaults(savedChannel())
    expect(draft.model_groups).toEqual([
      { group: 'default', models: ['gpt-6.1-mini', 'gpt-6.1-sol'] },
      { group: 'premium', models: ['gpt-6.1-mini', 'gpt-6.1-sol'] },
    ])
  })

  it('round trips explicit rows through both add and edit payloads', () => {
    const original = savedChannel()
    const draft = transformChannelToFormDefaults(original)
    const rows = [
      { group: 'default', models: ['gpt-6.1-mini'] },
      { group: 'premium', models: ['gpt-6.1-sol'] },
    ]
    draft.model_groups = rows
    const created = transformFormDataToCreatePayload(draft).channel
    const edited = transformFormDataToUpdatePayload(draft, original.id)
    for (const payload of [created, edited]) {
      expect(
        transformChannelToFormDefaults({ ...original, ...payload }).model_groups
      ).toEqual(rows)
    }
  })
})
