import { useState, useEffect, useCallback, useMemo } from "react"
import type { Article, Category, Tag } from "../lib/api"
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
import {
  FileText,
  Search,
  Edit,
  Trash2,
  Plus,
  Eye,
  EyeOff,
  RefreshCw,
} from "lucide-react"
import { CreateArticleModal } from "../components/admin/create-article-modal"
import { EditArticleModal } from "../components/admin/edit-article-modal"
import { ConfirmDialog } from "../components/admin/confirm-dialog"
import { Badge } from "~/components/ui/badge"

/* ─── Skeleton ─── */
function TableSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <div className="animate-pulse px-1">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 border-b border-border/40 px-4 py-3.5">
          <div className="h-4 w-8 rounded bg-muted" />
          <div className="h-4 flex-1 rounded bg-muted" />
          <div className="h-4 w-20 rounded bg-muted" />
          <div className="h-5 w-16 rounded-full bg-muted" />
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
function EmptyState({ search }: { search: string }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-col items-center justify-center py-20 text-muted-foreground"
    >
      <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted/50 mb-4">
        <FileText className="h-8 w-8" />
      </div>
      <p className="text-base font-medium">{search ? "没有匹配的文章" : "暂无文章"}</p>
      <p className="mt-1 text-sm">
        {search ? `未找到包含「${search}」的文章` : "点击右上角按钮创建第一篇文章"}
      </p>
    </motion.div>
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

export default function AdminArticles() {
  const [articles, setArticles] = useState<Article[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [page, setPage] = useState(1)

  const [deleteTarget, setDeleteTarget] = useState<Article | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Article | null>(null)

  const loadArticles = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [articleRes, catRes, tagRes] = await Promise.all([
        api.getArticles(),
        api.getCategories(),
        api.getTags(),
      ])
      if (articleRes.data?.articles) setArticles(articleRes.data.articles)
      if (catRes.data?.categories) setCategories(catRes.data.categories)
      if (tagRes.data?.tags) setTags(tagRes.data.tags)
    } catch (e: any) {
      setError(e.message || "加载文章失败")
      toast.error(e.message || "加载文章失败")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadArticles() }, [loadArticles])

  /* ─── filter & page ─── */
  const filtered = useMemo(() => {
    if (!search.trim()) return articles
    const q = search.trim().toLowerCase()
    return articles.filter((a) => a.title.toLowerCase().includes(q) || a.slug.toLowerCase().includes(q))
  }, [articles, search])

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const paged = useMemo(
    () => filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE), [filtered, page]
  )
  useEffect(() => { setPage(1) }, [search])

  /* ─── actions ─── */
  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.deleteArticle(deleteTarget.id)
      toast.success("文章已删除")
      setDeleteTarget(null)
      loadArticles()
    } catch (e: any) { toast.error(e.message || "删除失败") }
  }

  /* ─── stats ─── */
  const stats = useMemo(() => {
    const pub = articles.filter((a) => a.status === "published").length
    return { total: articles.length, published: pub, draft: articles.length - pub }
  }, [articles])

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
          <h1 className="text-2xl font-bold tracking-tight">文章管理</h1>
          <p className="text-sm text-muted-foreground">创建、编辑和管理所有文章</p>
        </div>
        <Button onClick={() => setCreateOpen(true)} size="lg">
          <Plus className="h-4 w-4 mr-1.5" />
          新建文章
        </Button>
      </motion.div>

      {/* Stats */}
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.06 }}
        className="grid grid-cols-3 gap-3"
      >
        <StatCard label="总文章" value={stats.total} icon={FileText} color="text-blue-500" bg="bg-blue-500/10" />
        <StatCard label="已发布" value={stats.published} icon={Eye} color="text-emerald-500" bg="bg-emerald-500/10" />
        <StatCard label="草稿" value={stats.draft} icon={EyeOff} color="text-amber-500" bg="bg-amber-500/10" />
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
            placeholder="搜索文章标题或别名..."
            className="pl-9 h-9 text-sm"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <Button variant="outline" size="sm" onClick={loadArticles} className="h-9">
          <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
          刷新
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
              文章列表
              {!loading && <span className="ml-2 text-xs font-normal text-muted-foreground">共 {filtered.length} 篇</span>}
            </CardTitle>
            {totalPages > 1 && (
              <div className="flex items-center gap-1 text-xs text-muted-foreground">
                <Button variant="ghost" size="icon" className="h-7 w-7" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
                  ‹
                </Button>
                <span className="tabular-nums px-1">{page}/{totalPages}</span>
                <Button variant="ghost" size="icon" className="h-7 w-7" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>
                  ›
                </Button>
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
                <Button variant="outline" size="sm" className="mt-4" onClick={loadArticles}>重试</Button>
              </div>
            ) : paged.length === 0 ? (
              <EmptyState search={search} />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="w-14 pl-5">ID</TableHead>
                    <TableHead>标题</TableHead>
                    <TableHead className="hidden md:table-cell">别名</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead className="hidden lg:table-cell">创建时间</TableHead>
                    <TableHead className="w-28 pr-5 text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paged.map((article) => (
                    <TableRow key={article.id} className="group">
                      <TableCell className="pl-5 text-muted-foreground text-sm tabular-nums">{article.id}</TableCell>
                      <TableCell>
                        <div className="flex flex-col gap-1.5">
                          <p className="truncate max-w-[200px] font-medium text-sm">{article.title}</p>
                          <div className="flex flex-wrap gap-1">
                            {article.categories && article.categories.length > 0 && article.categories.map((cat) => (
                              <Badge key={cat.id} variant="outline" className="text-[10px] px-1.5 py-0 h-5 border-blue-200 text-blue-600">
                                {cat.name}
                              </Badge>
                            ))}
                            {article.tags && article.tags.length > 0 && article.tags.map((tag) => (
                              <Badge key={tag.id} variant="secondary" className="text-[10px] px-1.5 py-0 h-5">
                                {tag.name}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <code className="text-xs bg-muted/60 px-1.5 py-0.5 rounded text-muted-foreground">{article.slug}</code>
                      </TableCell>
                      <TableCell>
                        <Badge variant={article.status === "published" ? "default" : "secondary"} className="gap-1 text-xs">
                          <span className={`h-1.5 w-1.5 rounded-full ${article.status === "published" ? "bg-emerald-400" : "bg-amber-400"}`} />
                          {article.status === "published" ? "已发布" : "草稿"}
                        </Badge>
                      </TableCell>
                      <TableCell className="hidden lg:table-cell text-sm text-muted-foreground">
                        {new Date(article.createdAt).toLocaleDateString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })}
                      </TableCell>
                      <TableCell className="pr-5 text-right">
                        <div className="flex items-center justify-end gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                          <Button
                            variant="ghost" size="sm" className="h-8 text-xs"
                            onClick={() => { setEditTarget(article); setEditOpen(true) }}
                          >
                            <Edit className="h-3.5 w-3.5 mr-1" />编辑
                          </Button>
                          <Button
                            variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive"
                            onClick={() => setDeleteTarget(article)}
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
        title="确认删除文章"
        message={`确定要永久删除文章「${deleteTarget?.title}」吗？此操作不可撤销。`}
        onConfirm={confirmDelete}
        onCancel={() => setDeleteTarget(null)}
      />

      <CreateArticleModal open={createOpen} onOpenChange={setCreateOpen} categories={categories} tags={tags} onCreated={loadArticles} />
      {editTarget && (
        <EditArticleModal open={editOpen} onOpenChange={setEditOpen} article={editTarget} categories={categories} tags={tags} onUpdated={loadArticles} />
      )}
    </div>
  )
}
