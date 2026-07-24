import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from "react"
import type { User } from "~/lib/api"

interface AdminUserContextType {
  user: User | null
  setUser: (user: User | null) => void
  clearUser: () => void
}

const AdminUserContext = createContext<AdminUserContextType>({
  user: null,
  setUser: () => {},
  clearUser: () => {},
})

const STORAGE_KEY = "user"

export function AdminUserProvider({ children }: { children: ReactNode }) {
  const [user, setUserState] = useState<User | null>(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  })

  const setUser = useCallback((u: User | null) => {
    setUserState(u)
    if (u) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(u))
    } else {
      localStorage.removeItem(STORAGE_KEY)
    }
  }, [])

  const clearUser = useCallback(() => setUser(null), [setUser])

  return (
    <AdminUserContext.Provider value={{ user, setUser, clearUser }}>
      {children}
    </AdminUserContext.Provider>
  )
}

export function useAdminUser() {
  return useContext(AdminUserContext)
}
