import { Outlet } from "react-router"
import { AdminSidebar } from "./sidebar"
import { AdminHeader } from "./header"
import { motion } from "framer-motion"
import { useSiteSettings } from "~/components/settings-context"
import { ScrollArea } from "~/components/ui/scroll-area"

export function AdminLayout() {
  const { settings } = useSiteSettings()

  return (
    <div className="flex flex-col h-screen overflow-hidden">
      <AdminHeader />
      <div className="flex flex-1 min-h-0">
        <AdminSidebar />
        <div className="flex-1 flex flex-col min-w-0">
          <motion.main
            className="flex-1 min-h-0"
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.1 }}
          >
            <ScrollArea className="h-full">
              <div className="p-6">
                <Outlet />
              </div>
            </ScrollArea>
          </motion.main>
          <footer className="border-t px-6 py-3 text-center text-xs text-muted-foreground shrink-0">
            <p>{settings.site_copyright}</p>
          </footer>
        </div>
      </div>
    </div>
  )
}
