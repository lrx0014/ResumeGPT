import { createI18n } from 'vue-i18n'

const messages = {
  en: {
    navGroups: {
      workspace: 'Workspace',
      tracking: 'Job search',
      workflow: 'Creation workflow',
      system: 'System',
    },
    nav: {
      overview: 'Overview',
      profiles: 'Profiles',
      jobs: 'Job Opportunities',
      hunters: 'Job Hunter',
      templates: 'Templates',
      generate: 'Create CVs',
      settings: 'Settings',
      tasks: 'Task Monitor',
    },
  },
  de: {
    navGroups: {
      workspace: 'Arbeitsbereich',
      tracking: 'Jobsuche',
      workflow: 'Erstellungsablauf',
      system: 'System',
    },
    nav: {
      overview: 'Übersicht',
      profiles: 'Profile',
      jobs: 'Stellenangebote',
      hunters: 'Job Hunter',
      templates: 'Vorlagen',
      generate: 'Lebensläufe erstellen',
      settings: 'Einstellungen',
      tasks: 'Task-Monitor',
    },
  },
}

export const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'en',
  messages,
})
