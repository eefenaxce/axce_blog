import { useMemo, useState } from "react"
import { Modal } from "../ui/modal"
import { Button } from "../ui/button"
import { Input } from "../ui/input"
import { Textarea } from "../ui/textarea"
import { Label } from "../ui/label"
import { Select } from "../ui/select"
import { api } from "../../lib/api"
import { toast } from "sonner"
import type { Category, Article, Tag } from "../../lib/api"
import { Badge } from "../ui/badge"
import { Switch } from "../ui/switch"
import { Upload, ImageIcon, X, ChevronLeft, ChevronRight, Eye, FileEdit } from "lucide-react"
import { marked } from "marked"

interface CreateArticleModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  categories: Category[]
  tags: Tag[]
  onCreated: () => void
}

const initialState = {
  title: "",
  slug: "",
  summary: "",
  content: "",
  coverUrl: "",
  status: "draft" as Article["status"],
  commentEnabled: true,
  categoryIds: [] as number[],
  tagIds: [] as number[],
}

export function CreateArticleModal({
  open,
  onOpenChange,
  categories,
  tags,
  onCreated,
}: CreateArticleModalProps) {
  const [step, setStep] = useState(1)
  const [form, setForm] = useState(initialState)
  const [loading, setLoading] = useState(false)
  const [uploadingCover, setUploadingCover] = useState(false)

  const update = <K extends keyof typeof form>(key: K, value: (typeof form)[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const renderedContent = useMemo(() => {
    return marked.parse(form.content || "", { async: false }) as string
  }, [form.content])

  const handleSubmit = async (): Promise<void> => {
    if (!form.title.trim()) {
      toast.error("请输入文章标题")
      setStep(1)
      return
    }

    setLoading(true)
    try {
      await api.createArticle({
        title: form.title,
        slug: form.slug || undefined,
        summary: form.summary || undefined,
        content: form.content || undefined,
        coverUrl: form.coverUrl || undefined,
        status: form.status,
        commentEnabled: form.commentEnabled,
        categoryIds: form.categoryIds.length > 0 ? form.categoryIds : undefined,
        tagIds: form.tagIds.length > 0 ? form.tagIds : undefined,
      })
      toast.success("文章创建成功")
      handleClose()
      onCreated()
    } catch (e: any) {
      toast.error(e.message || "创建失败")
    } finally {
      setLoading(false)
    }
  }

  const handleClose = () => {
    onOpenChange(false)
    setStep(1)
    setForm(initialState)
  }

  const toggleCategory = (id: number) => {
    update(
      "categoryIds",
      form.categoryIds.includes(id)
        ? form.categoryIds.filter((c) => c !== id)
        : [...form.categoryIds, id]
    )
  }

  const toggleTag = (id: number) => {
    update(
      "tagIds",
      form.tagIds.includes(id) ? form.tagIds.filter((t) => t !== id) : [...form.tagIds, id]
    )
  }

  const handleCoverUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const allowed = ["image/png", "image/jpeg", "image/jpg", "image/svg+xml", "image/webp", "image/gif"]
    if (!allowed.includes(file.type)) {
      toast.error("只支持图片文件")
      return
    }
    if (file.size > 2 * 1024 * 1024) {
      toast.error("图片大小不能超过 2MB")
      return
    }
    setUploadingCover(true)
    try {
      const res = await api.uploadImage(file)
      if (res.data?.url) {
        update("coverUrl", res.data.url)
        toast.success("封面上传成功")
      }
    } catch (e: any) {
      toast.error(e.message || "封面上传失败")
    } finally {
      setUploadingCover(false)
    }
  }

  const goNext = () => {
    if (!form.title.trim()) {
      toast.error("请输入文章标题")
      return
    }
    setStep(2)
  }

  const goBack = () => setStep(1)

  const renderStep1 = () => (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="title">标题 <span className="text-red-500">*</span></Label>
        <Input
          id="title"
          placeholder="请输入文章标题"
          value={form.title}
          onChange={(e) => update("title", e.target.value)}
          disabled={loading}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="slug">别名</Label>
        <Input
          id="slug"
          placeholder="URL 别名（留空自动生成）"
          value={form.slug}
          onChange={(e) => update("slug", e.target.value)}
          disabled={loading}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="summary">简述</Label>
        <Textarea
          id="summary"
          placeholder="文章简述（留空将自动从正文生成）"
          value={form.summary}
          onChange={(e) => update("summary", e.target.value)}
          className="min-h-[80px] resize-none"
          disabled={loading}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="cover">封面</Label>
        <div className="flex items-center gap-3">
          <div className="relative flex h-16 w-16 shrink-0 items-center justify-center rounded-lg border border-dashed border-border bg-muted/30 overflow-hidden">
            {form.coverUrl ? (
              <img src={form.coverUrl} alt="cover" className="h-full w-full object-cover" />
            ) : (
              <ImageIcon className="h-5 w-5 text-muted-foreground/60" />
            )}
          </div>
          <div className="flex flex-1 flex-col gap-2">
            <div className="flex items-center gap-2">
              <label className="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium hover:bg-accent transition-colors">
                <Upload className="h-3 w-3" />
                {uploadingCover ? "上传中..." : "上传封面"}
                <input
                  type="file"
                  accept="image/*"
                  onChange={handleCoverUpload}
                  disabled={loading || uploadingCover}
                  className="hidden"
                />
              </label>
              {form.coverUrl && (
                <button
                  type="button"
                  onClick={() => update("coverUrl", "")}
                  disabled={loading}
                  className="inline-flex items-center gap-1 rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 dark:hover:bg-red-950 transition-colors"
                >
                  <X className="h-3 w-3" />
                  移除
                </button>
              )}
            </div>
            <Input
              id="cover"
              placeholder="或输入封面图片 URL"
              value={form.coverUrl}
              onChange={(e) => update("coverUrl", e.target.value)}
              disabled={loading}
              className="h-8 text-xs"
            />
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label>分类</Label>
          <div className="flex flex-wrap gap-2 min-h-[34px] rounded-md border border-input bg-transparent px-3 py-2">
            {categories.length === 0 ? (
              <span className="text-sm text-muted-foreground">暂无分类，请先到分类管理创建</span>
            ) : (
              categories.map((cat) => (
                <Badge
                  key={cat.id}
                  variant={form.categoryIds.includes(cat.id) ? "default" : "outline"}
                  className="cursor-pointer select-none"
                  onClick={() => toggleCategory(cat.id)}
                >
                  {cat.name}
                </Badge>
              ))
            )}
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="status">状态</Label>
          <Select
            value={form.status}
            onValueChange={(v) => update("status", v as Article["status"])}
            placeholder="选择状态"
            options={[
              { value: "draft", label: "草稿" },
              { value: "published", label: "发布" }
            ]}
            disabled={loading}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="comment-enabled">允许评论</Label>
          <div className="flex h-9 items-center gap-2">
            <Switch
              id="comment-enabled"
              checked={form.commentEnabled}
              onCheckedChange={(v) => update("commentEnabled", v)}
              disabled={loading}
            />
            <span className="text-sm text-muted-foreground">
              {form.commentEnabled ? "开启" : "关闭"}
            </span>
          </div>
        </div>
      </div>

      <div className="space-y-2">
        <Label>标签</Label>
        <div className="flex flex-wrap gap-2 min-h-[34px] rounded-md border border-input bg-transparent px-3 py-2">
          {tags.length === 0 ? (
            <span className="text-sm text-muted-foreground">暂无标签，请先到标签管理创建</span>
          ) : (
            tags.map((tag) => (
              <Badge
                key={tag.id}
                variant={form.tagIds.includes(tag.id) ? "default" : "outline"}
                className="cursor-pointer select-none"
                onClick={() => toggleTag(tag.id)}
              >
                {tag.name}
              </Badge>
            ))
          )}
        </div>
      </div>
    </div>
  )

  const renderStep2 = () => (
    <div className="space-y-3">
      <div className="flex items-center gap-4 text-sm text-muted-foreground">
        <div className="flex items-center gap-1.5">
          <FileEdit className="h-4 w-4" />
          <span>编辑</span>
        </div>
        <div className="flex items-center gap-1.5">
          <Eye className="h-4 w-4" />
          <span>实时预览</span>
        </div>
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="flex flex-col gap-2">
          <Label htmlFor="content">内容</Label>
          <Textarea
            id="content"
            placeholder="请输入文章内容（支持 Markdown）"
            value={form.content}
            onChange={(e) => update("content", e.target.value)}
            className="min-h-[60vh] resize-none font-mono text-sm leading-relaxed"
            disabled={loading}
          />
        </div>
        <div className="flex flex-col gap-2 min-w-0">
          <Label>预览</Label>
          <div
            className="min-h-[60vh] overflow-y-auto rounded-md border border-input bg-background px-4 py-3 prose prose-sm dark:prose-invert max-w-none"
            dangerouslySetInnerHTML={{ __html: renderedContent }}
          />
        </div>
      </div>
    </div>
  )

  return (
    <Modal
      open={open}
      onOpenChange={handleClose}
      title={`新建文章 - 步骤 ${step}/2`}
      size={step === 2 ? "2xl" : "xl"}
    >
      {step === 1 ? renderStep1() : renderStep2()}

      <div className="flex justify-between gap-3 pt-6 border-t mt-6">
        {step === 1 ? (
          <>
            <Button variant="outline" onClick={handleClose} disabled={loading}>
              取消
            </Button>
            <Button onClick={goNext} disabled={loading}>
              下一步
              <ChevronRight className="ml-1 h-4 w-4" />
            </Button>
          </>
        ) : (
          <>
            <Button variant="outline" onClick={goBack} disabled={loading}>
              <ChevronLeft className="mr-1 h-4 w-4" />
              上一步
            </Button>
            <div className="flex gap-3">
              <Button variant="outline" onClick={handleClose} disabled={loading}>
                取消
              </Button>
              <Button onClick={handleSubmit} disabled={loading}>
                {loading ? "创建中..." : "创建文章"}
              </Button>
            </div>
          </>
        )}
      </div>
    </Modal>
  )
}
