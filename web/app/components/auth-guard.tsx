import { useEffect, useState } from "react"
import { Outlet, useNavigate } from "react-router"
import { api } from "../lib/api"

interface JwtPayload {
  user_id: number
  username: string
  group: string
  exp: number
}

function decodeToken(token: string): JwtPayload | null {
  try {
    const parts = token.split(".")
    if (parts.length !== 3) return null
    const payload = atob(parts[1].replace(/-/g, "+").replace(/_/g, "/"))
    return JSON.parse(payload)
  } catch {
    return null
  }
}

const ADMIN_GROUPS = ["admin"]

export default function AuthGuard() {
  const [checking, setChecking] = useState(true)
  const navigate = useNavigate()

  useEffect(() => {
    const token = api.getToken()
    if (!token) {
      navigate("/login", { replace: true })
      return
    }

    const payload = decodeToken(token)
    if (!payload || !payload.group || !ADMIN_GROUPS.includes(payload.group)) {
      // 如果 Token 无效或已过期，清除它
      if (!payload || payload.exp * 1000 < Date.now()) {
        api.clearToken()
      }
      // 跳转到首页或无权访问页面，而不是强制登录（除非 Token 彻底无效）
      navigate("/", { replace: true })
      return
    }

    setChecking(false)
  }, [navigate])

  if (checking) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <svg className="h-6 w-6 animate-spin text-primary" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          <div className="text-muted-foreground text-sm">验证身份中...</div>
        </div>
      </div>
    )
  }

  return <Outlet />
}
