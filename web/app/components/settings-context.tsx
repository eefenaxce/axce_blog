import { createContext, useContext, useState, useEffect, type ReactNode } from "react"
import { api } from "../lib/api"

interface SiteSettings {
  site_title: string
  site_description: string
  site_keywords: string
  site_author: string
  site_copyright: string
  [key: string]: string
}

interface SettingsContextType {
  settings: SiteSettings
  loading: boolean
}

const SettingsContext = createContext<SettingsContextType>({
  settings: {} as SiteSettings,
  loading: true,
})

export function useSiteSettings() {
  return useContext(SettingsContext)
}

export function SettingsProvider({ children }: { children: ReactNode }) {
  const [settings, setSettings] = useState<SiteSettings>({} as SiteSettings)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const res = await api.getSettings()
        if (res.data?.settings) {
          const newSettings: SiteSettings = {} as SiteSettings
          res.data.settings.forEach((s) => {
            let value = s.value
            if ((value.startsWith('"') && value.endsWith('"')) ||
                (value.startsWith("'") && value.endsWith("'"))) {
              value = value.slice(1, -1)
            }
            newSettings[s.key] = value
          })
          setSettings(newSettings)

          if (newSettings.site_title) {
            document.title = newSettings.site_title
          }
        }
      } catch (error) {
        console.error("Failed to load settings:", error)
      } finally {
        setLoading(false)
      }
    }

    loadSettings()
  }, [])

  return (
    <SettingsContext.Provider value={{ settings, loading }}>
      {children}
    </SettingsContext.Provider>
  )
}
