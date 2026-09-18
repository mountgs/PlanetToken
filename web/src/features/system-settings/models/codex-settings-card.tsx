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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const DEFAULT_IMAGES_MAIN_MODEL = 'gpt-5.6-luna'

/**
 * Nested object schema so dotted FormField names match react-hook-form paths.
 */
const codexSchema = z.object({
  codex: z.object({
    images_main_model: z.string().trim().min(1),
  }),
})

type CodexFormInput = z.input<typeof codexSchema>
type CodexFormValues = z.output<typeof codexSchema>

type FlatCodexDefaults = {
  'codex.images_main_model': string
}

const buildFormDefaults = (defaults: FlatCodexDefaults): CodexFormInput => ({
  codex: {
    images_main_model:
      defaults['codex.images_main_model'] || DEFAULT_IMAGES_MAIN_MODEL,
  },
})

const normalizeFormValues = (values: CodexFormValues): FlatCodexDefaults => ({
  'codex.images_main_model': values.codex.images_main_model.trim(),
})

interface Props {
  defaultValues: FlatCodexDefaults
}

export function CodexSettingsCard(props: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const formDefaults = useMemo(
    () => buildFormDefaults(props.defaultValues),
    [props.defaultValues]
  )

  const form = useForm<CodexFormInput, unknown, CodexFormValues>({
    resolver: zodResolver(codexSchema),
    defaultValues: formDefaults,
  })

  const baselineRef = useRef<FlatCodexDefaults>(
    normalizeFormValues(buildFormDefaults(props.defaultValues) as CodexFormValues)
  )
  const baselineSerializedRef = useRef<string>(
    JSON.stringify(baselineRef.current)
  )

  useEffect(() => {
    const next = buildFormDefaults(props.defaultValues)
    const normalized = normalizeFormValues(next as CodexFormValues)
    const serialized = JSON.stringify(normalized)
    if (serialized === baselineSerializedRef.current) return
    baselineRef.current = normalized
    baselineSerializedRef.current = serialized
    form.reset(next)
  }, [props.defaultValues, form])

  const onSubmit = async (values: CodexFormValues) => {
    const normalized = normalizeFormValues(values)
    const changedKeys = (
      Object.keys(normalized) as Array<keyof FlatCodexDefaults>
    ).filter((key) => normalized[key] !== baselineRef.current[key])

    if (changedKeys.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const key of changedKeys) {
      await updateOption.mutateAsync({
        key,
        value: normalized[key],
      })
    }

    baselineRef.current = normalized
    baselineSerializedRef.current = JSON.stringify(normalized)
    form.reset(buildFormDefaults(normalized))
  }

  return (
    <SettingsSection title={t('Codex Settings')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='codex.images_main_model'
            render={({ field }) => (
              <FormItem className='max-w-md'>
                <FormLabel>{t('Images main model')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={DEFAULT_IMAGES_MAIN_MODEL}
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Responses main model for Codex Image 2. ChatGPT-signed Codex no longer supports gpt-5.4-mini; default is gpt-5.6-luna. gpt-image-2 remains the tool model.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
