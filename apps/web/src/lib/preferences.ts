import type { SettingsPreferences } from './types'

export function applyTheme(theme: SettingsPreferences['theme']) {
  document.documentElement.dataset.theme = theme
}
