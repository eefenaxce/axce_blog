import { useState, useEffect, useCallback, useMemo } from "react"
import type { Category, Tag } from "../lib/api"
import { api } from "../lib/api"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table"
import { motion } from "framer-motion"
import { toast } from "sonner"
import { Folder, Tag as TagIcon, Search, Edit, Trash2, Plus, RefreshCw } from "lucide-react"
import { CreateCategoryTagModal } from "../components/admin/create-category-tag-modal"
import { EditCategoryTagModal } from "../components/admin/edit-category-tag-modal"
import { ConfirmDialog } from "../components/admin/confirm-dialog"

/* ─── Skeleton ─── */
function TableSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <div className="animate-pulse px-1">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 border-b border-border/40 px-4 py-3.5">
          <div className="h-4 w-8 rounded bg-muted" />
          <div className="h-4 flex-1 rounded bg-muted" />
          <div className="h-4 w-24 rounded bg-muted" />
          <div className="h-4 w-32 rounded bg-muted hidden lg:block" />
          <div className="h-4 w-28 rounded bg-muted hidden lg:block" />
          <div className="flex gap-2">
            <div className="h-8 w-16 rounded bg-muted" />
            <div className="h-8 w-8 rounded bg-muted" />
          </div>
        </div>
      ))}
    </div>
  )
}

/* ─── Empty State ─── */
function EmptyState({ search, type }: { search: string; type: "categories" | "tags" }) {
  const label = type === "categories" ? "分类" : "标签"
  const Icon = type === "categories" ? Folder : TagIcon
  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-col items-center justify-center py-20 text-muted-foreground"
    >
      <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted/50 mb-4">
        <Icon className="h-8 w-8" />
      </div>
      <p className="text-base font-medium">{search ? `没有匹配的${label}` : `暂无${label}`}</p>
      <p className="mt-1 text-sm">
        {search ? `未找到包含「${search}」的${label}` : `点击右上角按钮创建第一个${label}`}
      </p>
    </motion.div>
  )
}

/* ========== Page ========== */
const PAGE_SIZE = 15

export default function AdminCategories() {
  const [categories, setCategories] = useState<Category[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [page, setPage] = useState(1)
  const [activeTab, setActiveTab] = useState<"categories" | "tags">("categories")

  const [deleteTarget, setDeleteTarget] = useState<Category | Tag | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Category | Tag | null>(null)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [catRes, tagRes] = await Promise.all([api.getCategories(), api.getTags()])
      if (catRes.data?.categories) setCategories(catRes.data.categories)
      if (tagRes.data?.tags) setTags(tagRes.data.tags)
    } catch (e: any) {
      setError(e.message || "加载数据失败")
      toast.error(e.message || "加载数据失败")
    } finally { setLoading(false) }
  }, [])

  useEffect(() => { loadData() }, [loadData])

  /* ─── filter & page ─── */
  const tagList = useMemo(() => {
    if (!search.trim()) return tags
    const q = search.trim().toLowerCase()
    return tags.filter((t) => t.name.toLowerCase().includes(q) || t.slug.toLowerCase().includes(q))
  }, [tags, search])

  const catList = useMemo(() => {
    if (!search.trim()) return categories
    const q = search.trim().toLowerCase()
    return categories.filter((c) => c.name.toLowerCase().includes(q) || c.slug.toLowerCase().includes(q) || (c.description && c.description.toLowerCase().includes(q)))
  }, [categories, search])

  const currentList = activeTab === "categories" ? catList : tagList
  const totalPages = Math.max(1, Math.ceil(currentList.length / PAGE_SIZE))
  const paged = useMemo(() => currentList.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE), [currentList, page])
  useEffect(() => { setPage(1) }, [search, activeTab])

  /* ─── actions ─── */
  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      if (activeTab === "categories") {
        await api.deleteCategory((deleteTarget as Category).id)
        toast.success("分类已删除")
      } else {
        await api.deleteTag((deleteTarget as Tag).id)
        toast.success("标签已删除")
      }
      setDeleteTarget(null)
      loadData()
    } catch (e: any) { toast.error(e.message || "删除失败") }
  }

  /* ─── render ─── */
  return (
    <div className="flex flex-col gap-5">
      {/* Header */}
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-3"
      >
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">分类 / 标签</h1>
          <p className="text-sm text-muted-foreground">管理文章的分类与标签体系</p>
        </div>
        <div className="flex gap-2">
          <div className="flex rounded-lg border border-border bg-muted/40 p-0.5">
            <button
              onClick={() => setActiveTab("categories")}
              className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                activeTab === "categories" ? "bg-background shadow-sm text-foreground" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <Folder className="h-3.5 w-3.5" />
              分类
            </button>
            <button
              onClick={() => setActiveTab("tags")}
              className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                activeTab === "tags" ? "bg-background shadow-sm text-foreground" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <TagIcon className="h-3.5 w-3.5" />
              标签
            </button>
          </div>
          <Button size="lg" onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4 mr-1.5" />
            新建{activeTab === "categories" ? "分类" : "标签"}
          </Button>
        </div>
      </motion.div>

      {/* Stats */}
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.06 }}
        className="grid grid-cols-2 gap-3"
      >
        {([
          { label: "分类", value: categories.length, icon: Folder, color: "text-blue-500", bg: "bg-blue-500/10" },
          { label: "标签", value: tags.length, icon: TagIcon, color: "text-violet-500", bg: "bg-violet-500/10" },
        ]).map((s) => (
          <Card key={s.label} className="border-border/60 shadow-none">
            <CardContent className="flex items-center gap-3.5 p-4">
              <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${s.bg}`}>
                <s.icon className={`h-4 w-4 ${s.color}`} />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{s.label}</p>
                <p className="text-lg font-bold tabular-nums">{s.value}</p>
              </div>
            </CardContent>
          </Card>
        ))}
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
            placeholder={`搜索${activeTab === "categories" ? "分类" : "标签"}名称或别名...`}
            className="pl-9 h-9 text-sm"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <Button variant="outline" size="sm" onClick={loadData} className="h-9">
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
              {activeTab === "categories" ? "分类列表" : "标签列表"}
              {!loading && <span className="ml-2 text-xs font-normal text-muted-foreground">共 {currentList.length} 个</span>}
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
                <Button variant="outline" size="sm" className="mt-4" onClick={loadData}>重试</Button>
              </div>
            ) : paged.length === 0 ? (
              <EmptyState search={search} type={activeTab} />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="w-14 pl-5">ID</TableHead>
                    <TableHead className="w-16">图标</TableHead>
                    <TableHead>名称</TableHead>
                    <TableHead className="hidden md:table-cell">别名</TableHead>
                    {activeTab === "categories" && <TableHead className="hidden lg:table-cell">描述</TableHead>}
                    <TableHead className="hidden lg:table-cell">创建时间</TableHead>
                    <TableHead className="w-28 pr-5 text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paged.map((item) => (
                    <TableRow key={item.id} className="group">
                      <TableCell className="pl-5 text-muted-foreground text-sm tabular-nums">{item.id}</TableCell>
                      <TableCell>
                        {item.icon ? (
                          <img src={item.icon} alt="" className="h-8 w-8 rounded object-cover border" />
                        ) : (
                          <div className="h-8 w-8 rounded bg-muted/50" />
                        )}
                      </TableCell>
                      <TableCell>
                        <p className="font-medium text-sm">{item.name}</p>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <code className="text-xs bg-muted/60 px-1.5 py-0.5 rounded text-muted-foreground">{item.slug}</code>
                      </TableCell>
                      {activeTab === "categories" && (
                        <TableCell className="hidden lg:table-cell text-sm text-muted-foreground">
                          {(item as Category).description || "—"}
                        </TableCell>
                      )}
                      <TableCell className="hidden lg:table-cell text-sm text-muted-foreground">
                        {item.createdAt
                          ? new Date(item.createdAt).toLocaleDateString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })
                          : "—"}
                      </TableCell>
                      <TableCell className="pr-5 text-right">
                        <div className="flex items-center justify-end gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                          <Button variant="ghost" size="sm" className="h-8 text-xs"
                            onClick={() => { setEditTarget(item); setEditOpen(true) }}
                          >
                            <Edit className="h-3.5 w-3.5 mr-1" />编辑
                          </Button>
                          <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive"
                            onClick={() => setDeleteTarget(item)}
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
        title={`确认删除${activeTab === "categories" ? "分类" : "标签"}`}
        message={`确定要永久删除${activeTab === "categories" ? "分类" : "标签"}「${deleteTarget?.name}」吗？此操作不可撤销。`}
        onConfirm={confirmDelete}
        onCancel={() => setDeleteTarget(null)}
      />

      <CreateCategoryTagModal open={createOpen} onOpenChange={setCreateOpen} type={activeTab === "categories" ? "category" : "tag"} onCreated={loadData} />
      {editTarget && (
        <EditCategoryTagModal open={editOpen} onOpenChange={setEditOpen} type={activeTab === "categories" ? "category" : "tag"} item={editTarget} onUpdated={loadData} />
      )}
    </div>
  )
}
