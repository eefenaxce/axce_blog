import { useState, useEffect, useCallback, useMemo } from "react"
import type { Comment } from "../lib/api"
import { api } from "../lib/api"
import { Button } from "../components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table"
import { Badge } from "../components/ui/badge"
import { motion } from "framer-motion"
import { toast } from "sonner"
import { MessageSquare, CheckCircle2, XCircle, Trash2, RefreshCw } from "lucide-react"
import { ConfirmDialog } from "../components/admin/confirm-dialog"

const PAGE_SIZE = 15
const TABS = [
  { value: "", label: "全部" },
  { value: "pending", label: "待审核" },
  { value: "approved", label: "已通过" },
  { value: "rejected", label: "已拒绝" },
]

function TableSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <div className="animate-pulse px-1">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 border-b border-border/40 px-4 py-3.5">
          <div className="h-4 w-8 rounded bg-muted" />
          <div className="h-4 w-32 rounded bg-muted" />
          <div className="h-4 flex-1 rounded bg-muted" />
          <div className="h-5 w-16 rounded-full bg-muted" />
          <div className="h-4 w-28 rounded bg-muted hidden lg:block" />
          <div className="flex gap-2">
            <div className="h-8 w-8 rounded bg-muted" />
            <div className="h-8 w-8 rounded bg-muted" />
          </div>
        </div>
      ))}
    </div>
  )
}

function EmptyState({ tab }: { tab: string }) {
  const label = TABS.find((t) => t.value === tab)?.label || "全部"
  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-col items-center justify-center py-20 text-muted-foreground"
    >
      <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted/50 mb-4">
        <MessageSquare className="h-8 w-8" />
      </div>
      <p className="text-base font-medium">暂无{label}评论</p>
      <p className="mt-1 text-sm">读者提交的新评论将显示在这里</p>
    </motion.div>
  )
}

export default function AdminComments() {
  const [comments, setComments] = useState<Comment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState<string>("")
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)

  const [deleteTarget, setDeleteTarget] = useState<Comment | null>(null)

  const loadComments = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const offset = (page - 1) * PAGE_SIZE
      const res = await api.getAdminComments(status, offset, PAGE_SIZE)
      setComments(res.data?.items || [])
      setTotal(res.data?.total || 0)
    } catch (e: any) {
      setError(e.message || "加载评论失败")
      toast.error(e.message || "加载评论失败")
    } finally {
      setLoading(false)
    }
  }, [status, page])

  useEffect(() => { loadComments() }, [loadComments])

  const handleApprove = async (comment: Comment) => {
    try {
      await api.updateCommentStatus(comment.id, "approved")
      toast.success("评论已通过")
      loadComments()
    } catch (e: any) { toast.error(e.message || "操作失败") }
  }

  const handleReject = async (comment: Comment) => {
    try {
      await api.updateCommentStatus(comment.id, "rejected")
      toast.success("评论已拒绝")
      loadComments()
    } catch (e: any) { toast.error(e.message || "操作失败") }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.deleteComment(deleteTarget.id)
      toast.success("评论已删除")
      setDeleteTarget(null)
      loadComments()
    } catch (e: any) { toast.error(e.message || "删除失败") }
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const stats = useMemo(() => {
    // Approximate counts from current loaded set only; full stats would need a dedicated endpoint.
    return {
      total,
      pending: comments.filter((c) => c.status === "pending").length,
      approved: comments.filter((c) => c.status === "approved").length,
    }
  }, [comments, total])

  return (
    <div className="flex flex-col gap-5">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-3"
      >
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">评论管理</h1>
          <p className="text-sm text-muted-foreground">审核和管理读者评论</p>
        </div>
      </motion.div>

      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.06 }}
        className="grid grid-cols-3 gap-3"
      >
        <Card className="border-border/60 shadow-none">
          <CardContent className="flex items-center gap-3.5 p-4">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-500/10">
              <MessageSquare className="h-4 w-4 text-blue-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">总评论</p>
              <p className="text-lg font-bold tabular-nums">{stats.total}</p>
            </div>
          </CardContent>
        </Card>
        <Card className="border-border/60 shadow-none">
          <CardContent className="flex items-center gap-3.5 p-4">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-amber-500/10">
              <CheckCircle2 className="h-4 w-4 text-amber-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">待审核</p>
              <p className="text-lg font-bold tabular-nums">{stats.pending}</p>
            </div>
          </CardContent>
        </Card>
        <Card className="border-border/60 shadow-none">
          <CardContent className="flex items-center gap-3.5 p-4">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-emerald-500/10">
              <CheckCircle2 className="h-4 w-4 text-emerald-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">已通过</p>
              <p className="text-lg font-bold tabular-nums">{stats.approved}</p>
            </div>
          </CardContent>
        </Card>
      </motion.div>

      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.09 }}
        className="flex items-center justify-between"
      >
        <div className="flex items-center gap-1">
          {TABS.map((tab) => (
            <Button
              key={tab.value}
              variant={status === tab.value ? "secondary" : "ghost"}
              size="sm"
              onClick={() => { setStatus(tab.value); setPage(1) }}
            >
              {tab.label}
            </Button>
          ))}
        </div>
        <Button variant="outline" size="sm" onClick={loadComments}>
          <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
          刷新
        </Button>
      </motion.div>

      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.12 }}
      >
        <Card className="border-border/60 shadow-none">
          <CardHeader className="flex flex-row items-center justify-between pb-3 pt-5 px-5 space-y-0">
            <CardTitle className="text-sm font-semibold">
              评论列表
              {!loading && <span className="ml-2 text-xs font-normal text-muted-foreground">共 {total} 条</span>}
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
                <Button variant="outline" size="sm" className="mt-4" onClick={loadComments}>重试</Button>
              </div>
            ) : comments.length === 0 ? (
              <EmptyState tab={status} />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="w-14 pl-5">ID</TableHead>
                    <TableHead>作者</TableHead>
                    <TableHead>内容</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead className="hidden lg:table-cell">时间</TableHead>
                    <TableHead className="w-32 pr-5 text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {comments.map((comment) => (
                    <TableRow key={comment.id} className="group">
                      <TableCell className="pl-5 text-muted-foreground text-sm tabular-nums">{comment.id}</TableCell>
                      <TableCell>
                        <div className="flex flex-col gap-0.5">
                          <p className="font-medium text-sm">{comment.authorName || "匿名"}</p>
                          {comment.authorEmail && (
                            <p className="text-xs text-muted-foreground">{comment.authorEmail}</p>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <p className="text-sm line-clamp-2 max-w-md">{comment.content}</p>
                      </TableCell>
                      <TableCell>
                        {comment.status === "approved" ? (
                          <Badge variant="default" className="gap-1 text-xs bg-emerald-500/10 text-emerald-600 hover:bg-emerald-500/10">
                            <CheckCircle2 className="h-3 w-3" />
                            已通过
                          </Badge>
                        ) : comment.status === "pending" ? (
                          <Badge variant="secondary" className="gap-1 text-xs">
                            <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
                            待审核
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="gap-1 text-xs text-red-600">
                            <XCircle className="h-3 w-3" />
                            已拒绝
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="hidden lg:table-cell text-sm text-muted-foreground">
                        {new Date(comment.createdAt).toLocaleDateString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })}
                      </TableCell>
                      <TableCell className="pr-5 text-right">
                        <div className="flex items-center justify-end gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                          {comment.status !== "approved" && (
                            <Button variant="ghost" size="icon" className="h-8 w-8 text-emerald-600" onClick={() => handleApprove(comment)}>
                              <CheckCircle2 className="h-4 w-4" />
                            </Button>
                          )}
                          {comment.status !== "rejected" && (
                            <Button variant="ghost" size="icon" className="h-8 w-8 text-amber-600" onClick={() => handleReject(comment)}>
                              <XCircle className="h-4 w-4" />
                            </Button>
                          )}
                          <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive" onClick={() => setDeleteTarget(comment)}>
                            <Trash2 className="h-4 w-4" />
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
        title="确认删除评论"
        message={`确定要删除该评论吗？此操作不可撤销。`}
        onConfirm={confirmDelete}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  )
}
