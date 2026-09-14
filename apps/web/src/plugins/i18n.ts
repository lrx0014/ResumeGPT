import { createI18n } from 'vue-i18n'

const messages = {
  en: {
    nav: {
      overview: 'Overview',
      profiles: 'Profiles',
      jobs: 'Opportunities',
      generate: 'Generate',
      settings: 'Settings',
    },
  },
  de: {
    nav: {
      overview: 'Übersicht',
      profiles: 'Profile',
      jobs: 'Chancen',
      generate: 'Erstellen',
      settings: 'Einstellungen',
    },
  },
}

export const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'en',
  messages,
})
