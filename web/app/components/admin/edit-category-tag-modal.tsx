import { useState, useEffect, useRef } from "react"
import { Modal } from "../ui/modal"
import { Button } from "../ui/button"
import { Input } from "../ui/input"
import { Textarea } from "../ui/textarea"
import { Label } from "../ui/label"
import { api } from "../../lib/api"
import { toast } from "sonner"
import type { Category, Tag } from "../../lib/api"
import { Upload } from "lucide-react"

type EditType = "category" | "tag"

interface EditCategoryTagModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  type: EditType
  item: Category | Tag
  onUpdated: () => void
}

export function EditCategoryTagModal({
  open,
  onOpenChange,
  type,
  item,
  onUpdated,
}: EditCategoryTagModalProps) {
  const [name, setName] = useState("")
  const [slug, setSlug] = useState("")
  const [description, setDescription] = useState("")
  const [icon, setIcon] = useState("")
  const [uploadingIcon, setUploadingIcon] = useState(false)
  const [loading, setLoading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const label = type === "category" ? "分类" : "标签"

  // 当弹窗打开时，初始化表单数据
  useEffect(() => {
    if (open && item) {
      setName(item.name)
      setSlug(item.slug || "")
      setDescription((item as Category).description || "")
      setIcon(item.icon || "")
    }
  }, [open, item])

  const handleSubmit = async () => {
    if (!name.trim()) {
      toast.error(`请输入${label}名称`)
      return
    }

    setLoading(true)
    try {
      if (type === "category") {
        await api.updateCategory(item.id, {
          name,
          slug: slug || undefined,
          description: description || undefined,
          icon: icon || undefined,
        })
      } else {
        await api.updateTag(item.id, {
          name,
          slug: slug || undefined,
          icon: icon || undefined,
        })
      }
      toast.success(`${label}更新成功`)
      onOpenChange(false)
      onUpdated()
    } catch (e: any) {
      toast.error(e.message || "更新失败")
    } finally {
      setLoading(false)
    }
  }

  const handleClose = () => {
    onOpenChange(false)
  }

  const handleIconUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
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
    setUploadingIcon(true)
    try {
      const res = await api.uploadImage(file)
      if (res.data?.url) {
        setIcon(res.data.url)
        toast.success("图标上传成功")
      }
    } catch (e: any) {
      toast.error(e.message || "图标上传失败")
    } finally {
      setUploadingIcon(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ""
      }
    }
  }

  return (
    <Modal open={open} onOpenChange={handleClose} title={`编辑${label}`} size="md">
      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="name">{label}名称 <span className="text-red-500">*</span></Label>
          <Input
            id="name"
            placeholder={`请输入${label}名称`}
            value={name}
            onChange={(e) => setName(e.target.value)}
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

        {type === "category" && (
          <div className="space-y-2">
            <Label htmlFor="description">描述</Label>
            <Textarea
              id="description"
              placeholder="请输入分类描述（可选）"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="min-h-[100px]"
              disabled={loading}
            />
          </div>
        )}

        <div className="space-y-2">
          <Label htmlFor="icon">图标 URL</Label>
          <div className="flex items-center gap-3">
            {icon ? (
              <img src={icon} alt="icon" className="h-9 w-9 rounded object-cover border" />
            ) : (
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-dashed border-border bg-muted/30" />
            )}
            <Input
              id="icon"
              placeholder={`请输入${label}图标 URL（可选）`}
              value={icon}
              onChange={(e) => setIcon(e.target.value)}
              disabled={loading || uploadingIcon}
              className="flex-1"
            />
            <label className="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-border bg-background px-3 py-2 text-xs font-medium hover:bg-accent transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
              <Upload className="h-3 w-3" />
              {uploadingIcon ? "上传中..." : "上传"}
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                onChange={handleIconUpload}
                disabled={loading || uploadingIcon}
                className="hidden"
              />
            </label>
          </div>
        </div>

        <div className="flex justify-end gap-3 pt-4">
          <Button variant="outline" onClick={handleClose} disabled={loading}>
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? "更新中..." : `更新${label}`}
          </Button>
        </div>
      </div>
    </Modal>
  )
}