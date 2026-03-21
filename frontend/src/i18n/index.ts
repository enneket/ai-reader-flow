import i18n from 'i18next'
import {initReactI18next} from 'react-i18next'
import {en} from './en'
import {zh} from './zh'

const resources = {
  en: {translation: en},
  zh: {translation: zh},
}

// Get saved language or auto-detect
const getSavedLanguage = (): string => {
  const saved = localStorage.getItem('language')
  if (saved === 'en' || saved === 'zh') return saved

  const browserLang = navigator.language
  if (browserLang.startsWith('zh')) return 'zh'
  return 'en'
}

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: getSavedLanguage(),
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false,
    },
  })

export default i18n

export const changeLanguage = (lang: 'en' | 'zh') => {
  localStorage.setItem('language', lang)
  i18n.changeLanguage(lang)
}
