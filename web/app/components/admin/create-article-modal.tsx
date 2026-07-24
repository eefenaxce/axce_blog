import { useState } from "react"
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
import { Upload, ImageIcon, X } from "lucide-react"


interface CreateArticleModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  categories: Category[]
  tags: Tag[]
  onCreated: () => void
}

export function CreateArticleModal({
  open,
  onOpenChange,
  categories,
  tags,
  onCreated,
}: CreateArticleModalProps) {
  const [title, setTitle] = useState("")
  const [slug, setSlug] = useState("")
  const [summary, setSummary] = useState("")
  const [content, setContent] = useState("")
  const [coverUrl, setCoverUrl] = useState("")
  const [status, setStatus] = useState<Article["status"]>("draft")
  const [commentEnabled, setCommentEnabled] = useState(true)
  const [categoryIds, setCategoryIds] = useState<number[]>([])
  const [tagIds, setTagIds] = useState<number[]>([])
  const [loading, setLoading] = useState(false)
  const [uploadingCover, setUploadingCover] = useState(false)

const handleSubmit = async (): Promise<void> => {
    if (!title.trim()) {
      toast.error("请输入文章标题")
      return
    }

    setLoading(true)
    try {
      await api.createArticle({
        title,
        slug: slug || undefined,
        summary: summary || undefined,
        content: content || undefined,
        coverUrl: coverUrl || undefined,
        status,
        commentEnabled,
        categoryIds: categoryIds.length > 0 ? categoryIds : undefined,
        tagIds: tagIds.length > 0 ? tagIds : undefined,
      })
      toast.success("文章创建成功")
      onOpenChange(false)
      onCreated()
      // Reset form
      setTitle("")
      setSlug("")
      setSummary("")
      setContent("")
      setCoverUrl("")
      setStatus("draft")
      setCommentEnabled(true)
      setCategoryIds([])
      setTagIds([])
    } catch (e: any) {
      toast.error(e.message || "创建失败")
    } finally {
      setLoading(false)
    }
  }

  const handleClose = () => {
    onOpenChange(false)
    // Reset form
    setTitle("")
    setSlug("")
    setSummary("")
    setContent("")
    setCoverUrl("")
    setStatus("draft")
    setCommentEnabled(true)
    setCategoryIds([])
    setTagIds([])
  }

  const toggleCategory = (id: number) => {
    setCategoryIds((prev) =>
      prev.includes(id) ? prev.filter((c) => c !== id) : [...prev, id]
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
        setCoverUrl(res.data.url)
        toast.success("封面上传成功")
      }
    } catch (e: any) {
      toast.error(e.message || "封面上传失败")
    } finally {
      setUploadingCover(false)
    }
  }

  const toggleTag = (id: number) => {
    setTagIds((prev) =>
      prev.includes(id) ? prev.filter((t) => t !== id) : [...prev, id]
    )
  }

  return (
    <Modal open={open} onOpenChange={handleClose} title="新建文章" size="xl">
      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="title">标题 <span className="text-red-500">*</span></Label>
          <Input
            id="title"
            placeholder="请输入文章标题"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            disabled={loading}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="slug">别名</Label>
          <Input
            id="slug"
            placeholder="URL 别名（留空自动生成）"
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            disabled={loading}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="summary">简述</Label>
          <Textarea
            id="summary"
            placeholder="文章简述（留空将自动从正文生成）"
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
            className="min-h-[80px] resize-none"
            disabled={loading}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="cover">封面</Label>
          <div className="flex items-center gap-3">
            <div className="relative flex h-16 w-16 shrink-0 items-center justify-center rounded-lg border border-dashed border-border bg-muted/30 overflow-hidden">
              {coverUrl ? (
                <img src={coverUrl} alt="cover" className="h-full w-full object-cover" />
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
                {coverUrl && (
                  <button
                    type="button"
                    onClick={() => setCoverUrl("")}
                    disabled={loading}
                    className="inline-flex items-center gap-1 rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 transition-colors"
                  >
                    <X className="h-3 w-3" />
                    移除
                  </button>
                )}
              </div>
              <Input
                id="cover"
                placeholder="或输入封面图片 URL"
                value={coverUrl}
                onChange={(e) => setCoverUrl(e.target.value)}
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
                    variant={categoryIds.includes(cat.id) ? "default" : "outline"}
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
              value={status}
              onValueChange={(v) => setStatus(v as Article["status"])}
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
                checked={commentEnabled}
                onCheckedChange={setCommentEnabled}
                disabled={loading}
              />
              <span className="text-sm text-muted-foreground">
                {commentEnabled ? "开启" : "关闭"}
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
                  variant={tagIds.includes(tag.id) ? "default" : "outline"}
                  className="cursor-pointer select-none"
                  onClick={() => toggleTag(tag.id)}
                >
                  {tag.name}
                </Badge>
              ))
            )}
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="content">内容</Label>
          <Textarea
            id="content"
            placeholder="请输入文章内容"
            value={content}
            onChange={(e) => setContent(e.target.value)}
            className="min-h-[200px]"
            disabled={loading}
          />
        </div>

        <div className="flex justify-end gap-3 pt-4">
          <Button variant="outline" onClick={handleClose} disabled={loading}>
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? "创建中..." : "创建文章"}
          </Button>
        </div>
      </div>
    </Modal>
  )
}