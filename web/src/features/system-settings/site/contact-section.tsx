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
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
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
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const contactSchema = z.object({ Contact: z.string() })

export function ContactSection(props: { defaultValue: string }) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const defaultValues = useMemo(
    () => ({ Contact: props.defaultValue ?? '' }),
    [props.defaultValue]
  )
  const [imageUrl, setImageUrl] = useState('')
  const [imageError, setImageError] = useState(false)
  const { form, handleSubmit, handleReset, isDirty, isSubmitting } =
    useSettingsForm({
      resolver: zodResolver(contactSchema),
      defaultValues,
      onSubmit: async (data) => {
        await updateOption.mutateAsync({ key: 'Contact', value: data.Contact })
      },
    })
  const submit = async () => {
    try {
      await handleSubmit()
    } catch {
      // useUpdateOption reports the failure; retain the draft for retry.
    }
  }
  const content = form.watch('Contact')
  const insertImage = () => {
    let url: URL
    try {
      url = new URL(imageUrl.trim())
      if (
        !['https:', 'http:'].includes(url.protocol) ||
        url.username ||
        url.password
      ) {
        throw new Error('Invalid URL')
      }
    } catch {
      setImageError(true)
      return
    }
    form.setValue(
      'Contact',
      `${form.getValues('Contact')}\n\n![${t('Contact image')}](${url.href.replaceAll('(', '%28').replaceAll(')', '%29')})\n`,
      { shouldDirty: true }
    )
    setImageUrl('')
    setImageError(false)
  }
  return (
    <>
      <FormNavigationGuard when={isDirty} />
      <SettingsSection title={t('Contact Us')}>
        <Form {...form}>
          <SettingsForm
            onSubmit={(event) => {
              event.preventDefault()
              void submit()
            }}
          >
            <SettingsPageFormActions
              onSave={submit}
              onReset={handleReset}
              isSaving={isSubmitting || updateOption.isPending}
              isResetDisabled={!isDirty}
            />
            <FormDirtyIndicator isDirty={isDirty} />
            <FormField
              control={form.control}
              name='Contact'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Contact page content')}</FormLabel>
                  <FormControl>
                    <Textarea rows={10} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Add service hours, email, support links, and images using Markdown. Leave empty to show no contact details.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className='space-y-2'>
              <Label htmlFor='contact-image-url'>
                {t('Contact image URL')}
              </Label>
              <div className='flex flex-wrap gap-2'>
                <Input
                  id='contact-image-url'
                  type='url'
                  value={imageUrl}
                  onChange={(event) => {
                    setImageUrl(event.target.value)
                    setImageError(false)
                  }}
                  aria-invalid={imageError}
                  aria-describedby={
                    imageError ? 'contact-image-error' : 'contact-image-help'
                  }
                  className='min-w-0 flex-1'
                  placeholder='https://example.com/wechat-qr.png'
                />
                <Button
                  type='button'
                  variant='outline'
                  onClick={insertImage}
                  disabled={!imageUrl.trim()}
                >
                  {t('Insert image')}
                </Button>
              </div>
              <p
                id='contact-image-help'
                className='text-muted-foreground text-sm'
              >
                {t(
                  'Use a publicly accessible HTTP(S) image URL, such as a WeChat QR code. Insert multiple images and move their Markdown lines to change placement.'
                )}
              </p>
              {imageError && (
                <p
                  id='contact-image-error'
                  role='alert'
                  className='text-destructive text-sm'
                >
                  {t('Enter a valid HTTP(S) image URL without credentials.')}
                </p>
              )}
            </div>
            <section
              aria-label={t('Preview')}
              className='rounded-lg border p-5'
            >
              <h3 className='mb-4 text-sm font-medium'>{t('Preview')}</h3>
              <RichContent
                content={content}
                breaks
                className='[overflow-wrap:anywhere] break-words [&_img]:max-h-80 [&_img]:max-w-full [&_img]:object-contain'
              />
            </section>
          </SettingsForm>
        </Form>
      </SettingsSection>
    </>
  )
}
