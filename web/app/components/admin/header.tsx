import { useState, useRef, useEffect } from "react"
import { useNavigate } from "react-router"
import { Bell, LogOut, User, ChevronDown } from "lucide-react"
import { Button } from "../ui/button"
import { ThemeToggle } from "~/components/theme-toggle"
import { api } from "~/lib/api"
import { useAdminUser } from "./user-context"
import { useSiteSettings } from "~/components/settings-context"
import { motion, AnimatePresence } from "framer-motion"

function UserAvatar({ avatarUrl, name, size }: { avatarUrl: string; name: string; size: "sm" | "md" }) {
  const sizeClass = size === "sm" ? "h-7 w-7" : "h-8 w-8"

  return (
    <img
      src={avatarUrl}
      alt={name}
      className={`rounded-full object-cover shrink-0 ${sizeClass}`}
    />
  )
}

export function AdminHeader() {
  const { user, clearUser } = useAdminUser()
  const { settings } = useSiteSettings()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener("mousedown", handleClick)
    return () => document.removeEventListener("mousedown", handleClick)
  }, [])

  const handleLogout = () => {
    api.clearToken()
    clearUser()
    window.location.href = "/login"
  }

  const displayName = user?.nickname || user?.username || ""
  const avatarUrl = user?.avatar || ""

  return (
    <header className="relative z-50 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80 h-14 flex items-center justify-between px-6 shrink-0">
      {/* Left: Brand */}
      <div className="flex items-center gap-2.5">
        {settings.site_icon && (
          <img src={settings.site_icon} alt="logo" className="h-9 w-9 rounded object-contain" />
        )}
        <span className="font-semibold text-base">{settings.site_title}</span>
      </div>

      {/* Right */}
      <div className="flex items-center gap-2">
        <ThemeToggle className="h-8 w-8 text-muted-foreground hover:text-foreground" />
        <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-foreground">
          <Bell className="h-4 w-4" />
        </Button>

        {/* User dropdown */}
        <div className="relative" ref={ref}>
          <button
            onClick={() => setOpen(!open)}
            className="flex items-center gap-2 rounded-lg px-1.5 py-1 transition-colors hover:bg-accent"
          >
            <UserAvatar avatarUrl={avatarUrl} name={displayName} size="sm" />
            <span className="hidden sm:inline text-sm font-medium">{displayName}</span>
            <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform ${open ? "rotate-180" : ""}`} />
          </button>

          <AnimatePresence>
            {open && (
              <motion.div
                initial={{ opacity: 0, y: -4, scale: 0.96 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -4, scale: 0.96 }}
                transition={{ duration: 0.12 }}
                className="absolute right-0 top-full z-10 mt-1.5 w-52 rounded-xl border bg-popover p-1.5 shadow-lg"
              >
                <div className="flex items-center gap-2.5 px-3 py-2.5 border-b mb-1">
                  <UserAvatar avatarUrl={avatarUrl} name={displayName} size="md" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{displayName}</p>
                    <p className="text-xs text-muted-foreground truncate">{user?.email || ""}</p>
                  </div>
                </div>
                <button
                  onClick={() => { setOpen(false); navigate("/admin/settings") }}
                  className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
                >
                  <User className="h-4 w-4" />
                  个人设置
                </button>
                <button
                  onClick={handleLogout}
                  className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-950 transition-colors"
                >
                  <LogOut className="h-4 w-4" />
                  退出登录
                </button>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    </header>
  )
}
