import { useState, useEffect } from "react"
import { api } from "../lib/api"
import { Card, CardContent } from "../components/ui/card"
import { Users, FileText, Layers, Tags, TrendingUp } from "lucide-react"
import { motion } from "framer-motion"

const STAT_ITEMS = [
  { label: "总用户", icon: Users, color: "text-blue-500", bg: "bg-blue-500/10" },
  { label: "文章总数", icon: FileText, color: "text-emerald-500", bg: "bg-emerald-500/10" },
  { label: "分类数", icon: Layers, color: "text-violet-500", bg: "bg-violet-500/10" },
  { label: "标签数", icon: Tags, color: "text-amber-500", bg: "bg-amber-500/10" },
]

function SkeletonCard() {
  return (
    <Card className="border-border/60 shadow-none">
      <CardContent className="flex items-center gap-4 p-5">
        <div className="h-12 w-12 shrink-0 rounded-xl bg-muted animate-pulse" />
        <div className="space-y-2 min-w-0 flex-1">
          <div className="h-3 w-12 rounded bg-muted animate-pulse" />
          <div className="h-6 w-16 rounded bg-muted animate-pulse" />
        </div>
      </CardContent>
    </Card>
  )
}

export default function AdminDashboard() {
  const [stats, setStats] = useState<{ label: string; value: string; icon: typeof Users; color: string; bg: string }[] | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadStats()
  }, [])

  const loadStats = async () => {
    setLoading(true)
    try {
      const [usersRes, articlesRes, catRes, tagRes] = await Promise.all([
        api.getUsers().catch(() => ({ data: { total: 0 } })),
        api.getArticles().catch(() => ({ data: { total: 0 } })),
        api.getCategories().catch(() => ({ data: { categories: [] } })),
        api.getTags().catch(() => ({ data: { tags: [] } })),
      ])

      setStats([
        { ...STAT_ITEMS[0], value: String(usersRes.data?.total ?? 0) },
        { ...STAT_ITEMS[1], value: String(articlesRes.data?.total ?? 0) },
        { ...STAT_ITEMS[2], value: String(catRes.data?.categories?.length ?? 0) },
        { ...STAT_ITEMS[3], value: String(tagRes.data?.tags?.length ?? 0) },
      ])
    } catch {
      // keep loading skeleton on error
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Welcome Header */}
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="relative overflow-hidden rounded-2xl border border-border/60 bg-gradient-to-br from-primary/5 via-primary/3 to-transparent px-6 py-8"
      >
        <div className="relative z-10 space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">仪表盘</h1>
          <p className="text-sm text-muted-foreground">欢迎回来，这里是站点数据概览</p>
        </div>
        <div className="absolute right-6 top-1/2 -translate-y-1/2 opacity-[0.07]">
          <TrendingUp className="h-32 w-32" />
        </div>
      </motion.div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {(loading || !stats ? STAT_ITEMS.map((s) => ({ ...s, value: "" })) : stats).map((stat, i) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.06 + i * 0.05 }}
          >
            {loading || !stats ? (
              <SkeletonCard />
            ) : (
              <Card className="border-border/60 shadow-none hover:shadow-sm hover:border-border transition-all duration-200">
                <CardContent className="flex items-center gap-4 p-5">
                  <div className={`flex h-12 w-12 shrink-0 items-center justify-center rounded-xl ${stat.bg}`}>
                    <stat.icon className={`h-5 w-5 ${stat.color}`} />
                  </div>
                  <div className="min-w-0">
                    <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{stat.label}</p>
                    <motion.p
                      key={stat.value}
                      initial={{ scale: 1.1, opacity: 0 }}
                      animate={{ scale: 1, opacity: 1 }}
                      className="text-2xl font-bold tabular-nums truncate"
                    >
                      {stat.value}
                    </motion.p>
                  </div>
                </CardContent>
              </Card>
            )}
          </motion.div>
        ))}
      </div>
    </div>
  )
}
