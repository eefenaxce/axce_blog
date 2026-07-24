import { useState, useEffect, useCallback, useMemo } from "react"
import type { User } from "../lib/api"
import { api } from "../lib/api"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card"
import { Badge } from "../components/ui/badge"
import {
  Table,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableBody,
} from "../components/ui/table"
import { motion } from "framer-motion"
import { toast } from "sonner"
import { Users, UserCheck, UserX, Search, Shield, ShieldOff, Trash2, RefreshCw } from "lucide-react"
import { ConfirmDialog } from "../components/admin/confirm-dialog"

/* ─── Skeleton ─── */
function TableSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <div className="animate-pulse px-1">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 border-b border-border/40 px-4 py-3.5">
          <div className="h-4 w-8 rounded bg-muted" />
          <div className="flex items-center gap-3 flex-1">
            <div className="h-8 w-8 rounded-full bg-muted" />
            <div className="h-4 w-24 rounded bg-muted" />
          </div>
          <div className="h-4 w-40 rounded bg-muted" />
          <div className="h-5 w-14 rounded-full bg-muted" />
          <div className="h-5 w-12 rounded-full bg-muted" />
          <div className="h-4 w-28 rounded bg-muted hidden lg:block" />
          <div className="h-8 w-8 rounded bg-muted" />
        </div>
      ))}
    </div>
  )
}

/* ─── Empty State ─── */
function EmptyState({ search }: { search: string }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-col items-center justify-center py-20 text-muted-foreground"
    >
      <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted/50 mb-4">
        <Users className="h-8 w-8" />
      </div>
      <p className="text-base font-medium">{search ? "没有匹配的用户" : "暂无用户"}</p>
      <p className="mt-1 text-sm">{search ? `未找到包含「${search}」的用户` : "还没有任何注册用户"}</p>
    </motion.div>
  )
}

/* ─── Avatar ─── */
function Avatar({ avatarUrl, name, className }: { avatarUrl: string; name: string; className?: string }) {
  return (
    <img
      src={avatarUrl}
      alt={name}
      className={`h-9 w-9 shrink-0 rounded-full object-cover ${className ?? ""}`}
    />
  )
}

/* ─── Stat Card ─── */
function StatCard({ label, value, icon: Icon, color, bg }: {
  label: string; value: number; icon: React.ComponentType<{ className?: string }>; color: string; bg: string
}) {
  return (
    <Card className="border-border/60 shadow-none">
      <CardContent className="flex items-center gap-3.5 p-4">
        <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${bg}`}>
          <Icon className={`h-4 w-4 ${color}`} />
        </div>
        <div>
          <p className="text-xs text-muted-foreground">{label}</p>
          <p className="text-lg font-bold tabular-nums">{value}</p>
        </div>
      </CardContent>
    </Card>
  )
}

/* ========== Page ========== */
const PAGE_SIZE = 15

export default function AdminUsers() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [page, setPage] = useState(1)
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null)

  const loadUsers = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.getUsers()
      if (res.data?.users) setUsers(res.data.users)
    } catch (e: any) {
      setError(e.message || "加载用户失败")
      toast.error(e.message || "加载用户失败")
    } finally { setLoading(false) }
  }, [])

  useEffect(() => { loadUsers() }, [loadUsers])

  /* ─── filter & page ─── */
  const filtered = useMemo(() => {
    if (!search.trim()) return users
    const q = search.trim().toLowerCase()
    return users.filter((u) => u.username.toLowerCase().includes(q) || u.email.toLowerCase().includes(q) || (u.nickname && u.nickname.toLowerCase().includes(q)))
  }, [users, search])

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const paged = useMemo(() => filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE), [filtered, page])
  useEffect(() => { setPage(1) }, [search])

  /* ─── actions ─── */
  const updateStatus = async (user: User) => {
    const newStatus = user.status === 0 ? 1 : 0
    try {
      await api.updateUserStatus(user.id, newStatus)
      toast.success(newStatus === 0 ? "用户已启用" : "用户已封禁")
      loadUsers()
    } catch (e: any) { toast.error(e.message || "操作失败") }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.deleteUser(deleteTarget.id)
      toast.success("用户已删除")
      setDeleteTarget(null)
      loadUsers()
    } catch (e: any) { toast.error(e.message || "删除失败") }
  }

  /* ─── stats ─── */
  const stats = useMemo(() => {
    const active = users.filter((u) => u.status === 0).length
    return { total: users.length, active, banned: users.length - active }
  }, [users])

  /* ─── render ─── */
  return (
    <div className="flex flex-col gap-5">
      {/* Header */}
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="space-y-1"
      >
        <h1 className="text-2xl font-bold tracking-tight">用户管理</h1>
        <p className="text-sm text-muted-foreground">管理所有注册用户及其权限状态</p>
      </motion.div>

      {/* Stats */}
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.06 }}
        className="grid grid-cols-3 gap-3"
      >
        <StatCard label="总用户" value={stats.total} icon={Users} color="text-blue-500" bg="bg-blue-500/10" />
        <StatCard label="正常" value={stats.active} icon={UserCheck} color="text-emerald-500" bg="bg-emerald-500/10" />
        <StatCard label="封禁" value={stats.banned} icon={UserX} color="text-red-500" bg="bg-red-500/10" />
      </motion.div>

      {/* Toolbar */}
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.09 }}
        className="flex items-center gap-2"
      >
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索用户名、昵称或邮箱..."
            className="pl-9 h-9 text-sm"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <Button variant="outline" size="sm" onClick={loadUsers} className="h-9">
          <RefreshCw className="h-3.5 w-3.5 mr-1.5" />刷新
        </Button>
      </motion.div>

      {/* Table */}
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.12 }}
      >
        <Card className="border-border/60 shadow-none">
          <CardHeader className="flex flex-row items-center justify-between pb-3 pt-5 px-5 space-y-0">
            <CardTitle className="text-sm font-semibold">
              用户列表
              {!loading && <span className="ml-2 text-xs font-normal text-muted-foreground">共 {filtered.length} 人</span>}
            </CardTitle>
            {totalPages > 1 && (
              <div className="flex items-center gap-1 text-xs text-muted-foreground">
                <Button variant="ghost" size="icon" className="h-7 w-7" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>‹</Button>
                <span className="tabular-nums px-1">{page}/{totalPages}</span>
                <Button variant="ghost" size="icon" className="h-7 w-7" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>›</Button>
              </div>
            )}
          </CardHeader>
          <CardContent className="p-0">
            {loading ? (
              <TableSkeleton />
            ) : error ? (
              <div className="flex flex-col items-center py-20 text-muted-foreground">
                <p className="text-base font-medium">加载失败</p>
                <p className="mt-1 text-sm">{error}</p>
                <Button variant="outline" size="sm" className="mt-4" onClick={loadUsers}>重试</Button>
              </div>
            ) : paged.length === 0 ? (
              <EmptyState search={search} />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="w-14 pl-5">ID</TableHead>
                    <TableHead>用户</TableHead>
                    <TableHead className="hidden md:table-cell">邮箱</TableHead>
                    <TableHead>角色</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead className="hidden lg:table-cell">注册时间</TableHead>
                    <TableHead className="w-20 pr-5 text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paged.map((user) => (
                    <TableRow key={user.id} className="group">
                      <TableCell className="pl-5 text-muted-foreground text-sm tabular-nums">{user.id}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <Avatar avatarUrl={user.avatar || ""} name={user.nickname || user.username} />
                          <div className="min-w-0">
                            <p className="truncate text-sm font-medium">{user.nickname || user.username}</p>
                            <p className="truncate text-xs text-muted-foreground">@{user.username}</p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="hidden md:table-cell text-sm text-muted-foreground">{user.email}</TableCell>
                      <TableCell>
                        <Badge variant={user.group === "admin" ? "default" : "secondary"} className="text-xs">
                          {user.group === "admin" ? "管理员" : "用户"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={user.status === 0 ? "default" : "destructive"} className="gap-1 text-xs">
                          <span className={`h-1.5 w-1.5 rounded-full ${user.status === 0 ? "bg-emerald-400" : "bg-red-400"}`} />
                          {user.status === 0 ? "正常" : "封禁"}
                        </Badge>
                      </TableCell>
                      <TableCell className="hidden lg:table-cell text-sm text-muted-foreground">
                        {new Date(user.createdAt).toLocaleDateString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })}
                      </TableCell>
                      <TableCell className="pr-5 text-right">
                        <div className="flex items-center justify-end gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                          <Button
                            variant="ghost" size="icon" className="h-8 w-8"
                            title={user.status === 0 ? "封禁用户" : "启用用户"}
                            onClick={() => updateStatus(user)}
                          >
                            {user.status === 0 ? (
                              <ShieldOff className="h-3.5 w-3.5 text-muted-foreground" />
                            ) : (
                              <Shield className="h-3.5 w-3.5 text-emerald-500" />
                            )}
                          </Button>
                          <Button
                            variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive"
                            title="删除用户"
                            onClick={() => setDeleteTarget(user)}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </motion.div>

      <ConfirmDialog
        open={!!deleteTarget}
        title="确认删除用户"
        message={`确定要永久删除用户「${deleteTarget?.nickname || deleteTarget?.username}」吗？此操作不可撤销。`}
        onConfirm={confirmDelete}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  )
}
