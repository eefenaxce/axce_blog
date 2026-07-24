import { useState, useEffect, useCallback } from "react"
import type { Setting } from "../lib/api"
import { api } from "../lib/api"
import { Button } from "../components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { Switch } from "../components/ui/switch"
import { Save, Loader2, Upload, Trash2, Globe, ImageIcon, FileText, MessageSquare } from "lucide-react"
import { motion } from "framer-motion"
import { toast } from "sonner"

const settingLabels: Record<string, string> = {
  site_title: "网站标题",
  site_description: "网站描述",
  site_keywords: "网站关键词",
  site_author: "网站作者",
  site_copyright: "版权信息",
  posts_per_page: "每页文章数",
  enable_comments: "启用评论",
  require_review: "评论需审核",
  site_icon: "网站图标",
}

const fieldMeta: Record<string, { placeholder: string; hint?: string }> = {
  site_title: { placeholder: "输入网站标题", hint: "将显示在浏览器标签页和 SEO 结果中" },
  site_description: { placeholder: "简要描述你的网站", hint: "用于搜索引擎结果的摘要信息" },
  site_keywords: { placeholder: "博客, 技术, 编程", hint: "多个关键词用逗号分隔" },
  site_author: { placeholder: "输入作者名称" },
  site_copyright: { placeholder: "© 2025 Your Site" },
  posts_per_page: { placeholder: "10", hint: "建议 5-20 篇" },
  enable_comments: { placeholder: "" },
  require_review: { placeholder: "" },
  site_icon: { placeholder: "" },
}

export default function AdminSettings() {
  const [loaded, setLoaded] = useState(false)
  const [saving, setSaving] = useState(false)
  const [formData, setFormData] = useState<Record<string, string>>({})
  const [imagePreview, setImagePreview] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)

  const loadSettings = useCallback(async () => {
    try {
      const res = await api.getSettings()
      if (res.data?.settings) {
        const data: Record<string, string> = {}
        res.data.settings.forEach((s: Setting) => {
          data[s.key] = s.value
        })
        setFormData(data)
        if (data.site_icon) setImagePreview(data.site_icon)
      }
    } catch {
      toast.error("加载设置失败")
    } finally {
      setLoaded(true)
    }
  }, [])

  useEffect(() => { loadSettings() }, [loadSettings])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      await Promise.all(
        Object.entries(formData).map(([key, value]) => api.updateSetting(key, value))
      )
      toast.success("设置已保存")
    } finally {
      setSaving(false)
    }
  }

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const allowed = ["image/png", "image/jpeg", "image/jpg", "image/svg+xml", "image/webp"]
    if (!allowed.includes(file.type)) {
      toast.error("只支持 PNG、JPG、SVG 或 WebP 格式的图片")
      return
    }
    if (file.size > 2 * 1024 * 1024) {
      toast.error("图片大小不能超过 2MB")
      return
    }
    setUploading(true)
    try {
      const result = await api.uploadIcon(file)
      if (result.data?.icon_url) {
        setFormData((prev) => ({ ...prev, site_icon: result.data.icon_url }))
        setImagePreview(result.data.icon_url)
        toast.success("图标上传成功")
      }
    } catch {
      toast.error("图标上传失败")
    } finally {
      setUploading(false)
    }
  }

  const handleRemoveIcon = async () => {
    try {
      await api.updateSetting("site_icon", "")
      setFormData((prev) => ({ ...prev, site_icon: "" }))
      setImagePreview(null)
      toast.success("图标已移除")
    } catch {
      toast.error("移除图标失败")
    }
  }

  const setValue = (key: string, value: string) =>
    setFormData((prev) => ({ ...prev, [key]: value }))

  if (!loaded) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const generalKeys = ["site_title", "site_description", "site_keywords", "site_author", "site_copyright"]

  return (
    <form onSubmit={handleSave} className="flex flex-col gap-6">
      {/* ─── Header ─── */}
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-3"
      >
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">系统设置</h1>
          <p className="text-sm text-muted-foreground">配置网站的基础信息和全局功能</p>
        </div>
        <Button type="submit" disabled={saving} size="lg" className="shrink-0">
          {saving ? (
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
          ) : (
            <Save className="h-4 w-4 mr-2" />
          )}
          {saving ? "保存中..." : "保存设置"}
        </Button>
      </motion.div>

      {/* ─── Content Grid ─── */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* ─── Left Column: 基本信息 ─── */}
        <motion.div
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.08 }}
        >
          <Card className="h-full border-border/60 shadow-none">
            <CardHeader className="pb-4">
              <CardTitle className="flex items-center gap-2.5 text-base font-semibold">
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Globe className="h-4 w-4" />
                </span>
                基本信息
              </CardTitle>
              <CardDescription>网站的品牌标识与 SEO 元数据</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              {/* ── Icon Upload ── */}
              <div className="flex items-start gap-4 pb-5 border-b border-border/50">
                <div className="relative shrink-0 flex h-16 w-16 items-center justify-center rounded-xl border-2 border-dashed border-border/60 bg-muted/30 overflow-hidden">
                  {imagePreview ? (
                    <img src={imagePreview} alt="icon" className="h-full w-full object-contain" />
                  ) : (
                    <ImageIcon className="h-6 w-6 text-muted-foreground/60" />
                  )}
                </div>
                <div className="flex-1 space-y-1.5 min-w-0">
                  <Label className="text-sm font-medium">网站图标 (Favicon)</Label>
                  <p className="text-xs text-muted-foreground">
                    PNG / JPG / SVG / WebP，建议 32×32，最大 2MB
                  </p>
                  <div className="flex flex-wrap gap-2 pt-1">
                    <label className="inline-flex items-center gap-1.5 cursor-pointer rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium hover:bg-accent transition-colors">
                      <Upload className="h-3 w-3" />
                      {uploading ? "上传中..." : "上传图标"}
                      <input
                        type="file"
                        accept="image/png,image/jpeg,image/svg+xml,image/webp"
                        onChange={handleImageUpload}
                        disabled={uploading || saving}
                        className="hidden"
                      />
                    </label>
                    {imagePreview && (
                      <button
                        type="button"
                        onClick={handleRemoveIcon}
                        disabled={saving}
                        className="inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 transition-colors disabled:opacity-50"
                      >
                        <Trash2 className="h-3 w-3" />
                        移除
                      </button>
                    )}
                  </div>
                </div>
              </div>

              {/* ── Text Fields ── */}
              <div className="space-y-4">
                {generalKeys.map((key) => {
                  const meta = fieldMeta[key] || { placeholder: "" }
                  return (
                    <div key={key} className="space-y-1.5">
                      <Label htmlFor={key} className="text-sm font-medium">
                        {settingLabels[key] || key}
                      </Label>
                      <Input
                        id={key}
                        value={formData[key] || ""}
                        onChange={(e) => setValue(key, e.target.value)}
                        placeholder={meta.placeholder}
                        disabled={saving}
                        className="h-10 text-sm"
                      />
                      {meta.hint && (
                        <p className="text-xs text-muted-foreground">{meta.hint}</p>
                      )}
                    </div>
                  )
                })}
              </div>
            </CardContent>
          </Card>
        </motion.div>

        {/* ─── Right Column: 内容与功能 ─── */}
        <motion.div
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.16 }}
          className="flex flex-col gap-6"
        >
          {/* 显示设置 */}
          <Card className="border-border/60 shadow-none">
            <CardHeader className="pb-4">
              <CardTitle className="flex items-center gap-2.5 text-base font-semibold">
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500/10 text-blue-500">
                  <FileText className="h-4 w-4" />
                </span>
                内容显示
              </CardTitle>
              <CardDescription>控制网站文章列表的展示</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-1.5">
                <Label htmlFor="posts_per_page" className="text-sm font-medium">
                  每页文章数
                </Label>
                <Input
                  id="posts_per_page"
                  type="number"
                  min={1}
                  max={50}
                  value={formData["posts_per_page"] || "10"}
                  onChange={(e) => setValue("posts_per_page", e.target.value)}
                  placeholder="10"
                  disabled={saving}
                  className="h-10 w-28 text-sm"
                />
                <p className="text-xs text-muted-foreground">建议 5-20 篇</p>
              </div>
            </CardContent>
          </Card>

          {/* 评论设置 */}
          <Card className="border-border/60 shadow-none">
            <CardHeader className="pb-4">
              <CardTitle className="flex items-center gap-2.5 text-base font-semibold">
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-500">
                  <MessageSquare className="h-4 w-4" />
                </span>
                评论管理
              </CardTitle>
              <CardDescription>配置评论功能的开启与审核策略</CardDescription>
            </CardHeader>
            <CardContent className="space-y-1">
              {["enable_comments", "require_review"].map((key) => {
                const isOn = formData[key] === "true"
                return (
                  <div
                    key={key}
                    className="flex items-center justify-between py-3 px-2 -mx-2 rounded-lg hover:bg-accent/50 transition-colors"
                  >
                    <div className="space-y-0.5 min-w-0 mr-4">
                      <Label className="text-sm font-medium cursor-pointer" htmlFor={key}>
                        {settingLabels[key] || key}
                      </Label>
                      <p className="text-xs text-muted-foreground">
                        {key === "enable_comments"
                          ? isOn
                            ? "读者可以在文章下方发表评论"
                            : "所有文章的评论区将被关闭"
                          : isOn
                            ? "评论需要管理员审核后才能公开显示"
                            : "评论发布后立即公开显示"}
                      </p>
                    </div>
                    <Switch
                      id={key}
                      checked={isOn}
                      onCheckedChange={(checked) => setValue(key, checked ? "true" : "false")}
                      disabled={saving}
                    />
                  </div>
                )
              })}
            </CardContent>
          </Card>
        </motion.div>
      </div>
    </form>
  )
}
