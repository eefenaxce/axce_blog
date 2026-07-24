import { NavLink } from "react-router"
import {
  LayoutDashboard,
  Users,
  FileText,
  MessageSquare,
  Tags,
  Settings,
  Palette,
} from "lucide-react"
import { motion } from "framer-motion"

const menuItems = [
  { icon: LayoutDashboard, label: "仪表盘", path: "/admin" },
  { icon: Users, label: "用户管理", path: "/admin/users" },
  { icon: FileText, label: "文章管理", path: "/admin/articles" },
  { icon: MessageSquare, label: "评论管理", path: "/admin/comments" },
  { icon: Tags, label: "分类/标签", path: "/admin/categories" },
  { icon: Palette, label: "主题管理", path: "/admin/themes" },
  { icon: Settings, label: "系统设置", path: "/admin/settings" },
]

export function AdminSidebar() {
  return (
    <aside className="w-60 border-r bg-card h-full flex flex-col shrink-0">
      <nav className="space-y-1.5 px-3 pt-4 flex-1">
        {menuItems.map((item, i) => (
          <motion.div
            key={item.path}
            initial={{ opacity: 0, x: -12 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.2, delay: i * 0.04 }}
          >
            <NavLink
              to={item.path}
              end={item.path === "/admin"}
              className={({ isActive }) =>
                `relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                  isActive
                    ? "bg-primary/10 text-primary before:absolute before:left-0 before:top-1/2 before:-translate-y-1/2 before:w-[3px] before:h-5 before:rounded-full before:bg-primary"
                    : "text-muted-foreground hover:bg-accent hover:text-foreground"
                }`
              }
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {item.label}
            </NavLink>
          </motion.div>
        ))}
      </nav>
    </aside>
  )
}
