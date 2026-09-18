import { createI18n } from 'vue-i18n'

import type { SettingsPreferences } from '../lib/types'
import en from './locales/en'

export type InterfaceLocale = SettingsPreferences['interfaceLanguage']

export const interfaceLanguages: { code: InterfaceLocale; nativeName: string; shortName: string }[] = [
  { code: 'en', nativeName: 'English', shortName: 'EN' },
  { code: 'de', nativeName: 'Deutsch', shortName: 'DE' },
  { code: 'fr', nativeName: 'Français', shortName: 'FR' },
  { code: 'es', nativeName: 'Español', shortName: 'ES' },
  { code: 'ja', nativeName: '日本語', shortName: '日本' },
  { code: 'zh-CN', nativeName: '简体中文', shortName: '简' },
  { code: 'zh-TW', nativeName: '繁體中文', shortName: '繁' },
]

export function normalizeInterfaceLocale(value: string): InterfaceLocale {
  return interfaceLanguages.some(language => language.code === value) ? value as InterfaceLocale : 'en'
}

export const i18n = createI18n({ legacy: false, locale: 'en', fallbackLocale: 'en', messages: { en } })

// Every locale other than the bundled default (`en`, needed synchronously as
// fallbackLocale) is code-split and only fetched the first time it is used.
const localeLoaders: Record<InterfaceLocale, () => Promise<{ default: typeof en }>> = {
  en: () => Promise.resolve({ default: en }),
  de: () => import('./locales/de'),
  fr: () => import('./locales/fr'),
  es: () => import('./locales/es'),
  ja: () => import('./locales/ja'),
  'zh-CN': () => import('./locales/zh-CN'),
  'zh-TW': () => import('./locales/zh-TW'),
}

const loadedLocales = new Set<InterfaceLocale>(['en'])

export async function loadLocaleMessages(locale: InterfaceLocale): Promise<void> {
  if (loadedLocales.has(locale)) return
  const module = await localeLoaders[locale]()
  i18n.global.setLocaleMessage(locale, module.default)
  loadedLocales.add(locale)
}
