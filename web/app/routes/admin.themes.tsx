import { useState, useEffect, useRef, useMemo } from "react"
import { api, type Theme, type RemoteTheme, type DownloadProgress, type ThemeSettingsResponse, type ThemeSettingField } from "~/lib/api"
import { Card, CardContent } from "~/components/ui/card"
import { Button } from "~/components/ui/button"
import { Input } from "~/components/ui/input"
import { Label } from "~/components/ui/label"
import { Textarea } from "~/components/ui/textarea"
import { Switch } from "~/components/ui/switch"
import { Select } from "~/components/ui/select"
import { ScrollArea } from "~/components/ui/scroll-area"
import { Modal } from "~/components/ui/modal"
import { Loader2, Upload, Trash2, CheckCircle2, ExternalLink, Download, Search, PackageOpen, Monitor, AlertCircle, ChevronLeft, ChevronRight, Settings2, Plus } from "lucide-react"
import { motion, AnimatePresence } from "framer-motion"

const API_BASE = ""

export default function AdminThemes() {
  const [tab, setTab] = useState<"local" | "remote">("local")
  const [themes, setThemes] = useState<Theme[]>([])
  const [remoteThemes, setRemoteThemes] = useState<RemoteTheme[]>([])
  const [loading, setLoading] = useState(true)
  const [remoteLoading, setRemoteLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [downloadProgresses, setDownloadProgresses] = useState<Record<string, DownloadProgress>>({})
  const [activating, setActivating] = useState<string | null>(null)
  const [error, setError] = useState("")
  const [remoteError, setRemoteError] = useState("")
  const [deleteTarget, setDeleteTarget] = useState<Theme | null>(null)
  const [search, setSearch] = useState("")
  const [remotePage, setRemotePage] = useState(0)
  const [remoteTotal, setRemoteTotal] = useState(0)
  const [detailTheme, setDetailTheme] = useState<RemoteTheme | null>(null)
  const [imgLoaded, setImgLoaded] = useState<Set<string>>(new Set())
  const remotePageSize = 20
  const remoteTotalPages = Math.max(1, Math.ceil(remoteTotal / remotePageSize))
  const uploadRef = useRef<HTMLInputElement>(null)
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Theme settings state
  const [settingsTheme, setSettingsTheme] = useState<Theme | null>(null)
  const [settingsData, setSettingsData] = useState<ThemeSettingsResponse | null>(null)
  const [settingsValues, setSettingsValues] = useState<Record<string, any>>({})
  const [settingsLoading, setSettingsLoading] = useState(false)
  const [settingsSaving, setSettingsSaving] = useState(false)
  const [settingsError, setSettingsError] = useState("")
  const [settingsTab, setSettingsTab] = useState("")

  useEffect(() => {
    if (tab === "remote") {
      loadRemote()
    } else {
      loadThemes()
    }
  }, [tab])

  const loadThemes = async () => {
    setLoading(true)
    try {
      const res = await api.getThemes()
      setThemes(res.data?.themes ?? [])
    } catch {
      setThemes([])
    } finally {
      setLoading(false)
    }
  }

  const loadRemote = async (page: number = 0, keyword?: string) => {
    setRemoteLoading(true)
    setRemoteError("")
    try {
      const kw = keyword ?? search
      const res = await api.getRemoteThemes(kw, page, remotePageSize)
      const list = res.data?.themes ?? []
      setRemoteThemes(list)
      setRemoteTotal(res.data?.total ?? 0)
      setRemotePage(page)
      setImgLoaded(new Set())

      // 恢复页面刷新前仍在进行中的下载进度
      if (list.length > 0) {
        try {
          const repos = list.map((t) => t.id).join(",")
          const progressRes = await api.getDownloadProgress(repos)
          const progresses = progressRes.data?.progresses ?? []
          if (progresses.length > 0) {
            setDownloadProgresses((prev) => {
              const next = { ...prev }
              for (const p of progresses) {
                // 只恢复未完成/出错的状态，已完成时以 installed 字段为准
                if (p.status !== "completed") {
                  next[p.repo] = p
                }
              }
              return next
            })
          }
        } catch {
          // 恢复进度失败不影响列表展示
        }
      }
    } catch (err: any) {
      setRemoteError(err.message || "无法连接主题商店")
      setRemoteThemes([])
      setRemoteTotal(0)
    } finally {
      setRemoteLoading(false)
    }
  }

  const dpRef = useRef<Record<string, DownloadProgress>>({})

  // 同步 state → ref（保证闭包里拿到最新进度）
  useEffect(() => {
    dpRef.current = downloadProgresses
  })

  // 计算当前进行中的 repo 标识（只变状态时触发轮询启停，pct 变化不触发）
  const activeReposKey = useMemo(() => {
    return Object.keys(downloadProgresses)
      .filter((r) => downloadProgresses[r]?.status !== "completed" && downloadProgresses[r]?.status !== "error")
      .sort()
      .join(",")
  }, [downloadProgresses])

  // 启动 / 停止轮询定时器
  useEffect(() => {
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }

    if (!activeReposKey) return

    const poll = async () => {
      const repos = Object.keys(dpRef.current).filter(
        (r) => dpRef.current[r]?.status !== "completed" && dpRef.current[r]?.status !== "error"
      )
      if (repos.length === 0) return

      try {
        const res = await api.getDownloadProgress(repos.join(","))
        const progresses = res.data?.progresses ?? []
        if (progresses.length === 0) return

        setDownloadProgresses((prev) => {
          const next = { ...prev }
          for (const p of progresses) {
            next[p.repo] = p
          }
          return next
        })

        const hasCompleted = progresses.some((p) => p.status === "completed")
        if (hasCompleted) {
          await loadThemes()
          await loadRemote(remotePage)
        }
      } catch {
        // 忽略轮询错误
      }
    }

    poll() // 立即查一次
    pollTimerRef.current = setInterval(poll, 2000)

    return () => {
      if (pollTimerRef.current) {
        clearInterval(pollTimerRef.current)
      }
    }
  }, [activeReposKey])

  const handleDownload = async (repoUrl: string) => {
    setRemoteError("")
    try {
      const res = await api.downloadTheme(repoUrl)
      const repo = res.data?.repo
      if (!repo) return

      // 立即设置初始进度，触发轮询
      setDownloadProgresses((prev) => ({
        ...prev,
        [repo]: { repo, pct: 0, status: "downloading" },
      }))
    } catch (err: any) {
      setRemoteError(err.message || "下载主题失败")
    }
  }

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setUploading(true)
    setError("")
    try {
      await api.uploadTheme(file)
      await loadThemes()
    } catch (err: any) {
      setError(err.message || "上传失败")
    } finally {
      setUploading(false)
      if (uploadRef.current) uploadRef.current.value = ""
    }
  }

  const handleActivate = async (id: string) => {
    setActivating(id)
    setError("")
    try {
      await api.activateTheme(id)
      await loadThemes()
    } catch (err: any) {
      setError(err.message || "激活失败")
    } finally {
      setActivating(null)
    }
  }

  const handleDeleteClick = (theme: Theme) => {
    setDeleteTarget(theme)
  }

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return
    setError("")
    try {
      await api.deleteTheme(deleteTarget.id)
      setDeleteTarget(null)
      await loadThemes()
    } catch (err: any) {
      setError(err.message || "删除失败")
    }
  }

  // ─── Theme Settings ───
  const openSettings = async (theme: Theme) => {
    setSettingsTheme(theme)
    setSettingsLoading(true)
    setSettingsError("")
    try {
      const res = await api.getThemeSettings(theme.id)
      const data = res.data!
      setSettingsData(data)
      setSettingsValues(data.values || {})
      // 默认选中第一个分组
      if (data.forms.length > 0) {
        setSettingsTab(data.forms[0].group)
      }
    } catch (err: any) {
      setSettingsError(err.message || "无法加载主题设置")
    } finally {
      setSettingsLoading(false)
    }
  }

  const closeSettings = () => {
    setSettingsTheme(null)
    setSettingsData(null)
    setSettingsValues({})
    setSettingsError("")
    setSettingsTab("")
  }

  const handleSettingChange = (name: string, value: any) => {
    setSettingsValues((prev) => ({ ...prev, [name]: value }))
  }

  const handleSaveSettings = async () => {
    if (!settingsTheme) return
    setSettingsSaving(true)
    setSettingsError("")
    try {
      await api.updateThemeSettings(settingsTheme.id, settingsValues)
      closeSettings()
    } catch (err: any) {
      setSettingsError(err.message || "保存设置失败")
    } finally {
      setSettingsSaving(false)
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex items-center justify-between"
      >
        <div>
          <h1 className="text-2xl font-bold tracking-tight">主题管理</h1>
          <p className="text-sm text-muted-foreground mt-1">管理和切换站点主题</p>
        </div>
        {tab === "local" && (
          <>
            <input
              ref={uploadRef}
              type="file"
              accept=".zip"
              onChange={handleUpload}
              className="hidden"
            />
            <Button onClick={() => uploadRef.current?.click()} disabled={uploading}>
              {uploading ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Upload className="h-4 w-4 mr-2" />
              )}
              {uploading ? "上传中..." : "上传主题"}
            </Button>
          </>
        )}
      </motion.div>

      {/* Error */}
      <AnimatePresence>
        {error && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            className="rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive"
          >
            {error}
            <button onClick={() => setError("")} className="ml-2 underline">关闭</button>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Tabs */}
      <div className="flex gap-1 rounded-lg bg-muted p-1 w-fit">
        <button
          onClick={() => { closeSettings(); setTab("local") }}
          className={`flex items-center gap-1.5 rounded-md px-4 py-2 text-sm font-medium transition-colors ${
            tab === "local" && !settingsTheme ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <Monitor className="h-4 w-4" />
          本地主题
        </button>
        <button
          onClick={() => { closeSettings(); setTab("remote"); loadRemote(0) }}
          className={`flex items-center gap-1.5 rounded-md px-4 py-2 text-sm font-medium transition-colors ${
            tab === "remote" && !settingsTheme ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <PackageOpen className="h-4 w-4" />
          主题商店
        </button>
        {settingsTheme && (
          <button
            className="flex items-center gap-1.5 rounded-md px-4 py-2 text-sm font-medium bg-background text-foreground shadow-sm"
          >
            <Settings2 className="h-4 w-4" />
            {settingsTheme.name}
          </button>
        )}
      </div>

      {/* Theme Settings View */}
      {settingsTheme ? (
        <motion.div
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          className="space-y-6"
        >
          {settingsLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : settingsError && !settingsData ? (
            <div className="rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-12 text-center text-sm text-destructive">
              {settingsError}
              <button onClick={closeSettings} className="block mx-auto mt-3 underline text-xs">返回主题列表</button>
            </div>
          ) : settingsData && settingsData.forms.length > 0 ? (
            <>
              {/* 设置分组 Tab */}
              <ScrollArea className="max-w-full">
                <div className="flex gap-1 rounded-lg bg-muted p-1 w-fit">
                  {settingsData.forms.map((form) => (
                    <button
                      key={form.group}
                      onClick={() => setSettingsTab(form.group)}
                      className={`shrink-0 rounded-md px-4 py-1.5 text-sm font-medium transition-colors whitespace-nowrap ${
                        settingsTab === form.group
                          ? "bg-background text-foreground shadow-sm"
                          : "text-muted-foreground hover:text-foreground"
                      }`}
                    >
                      {form.label}
                    </button>
                  ))}
                </div>
              </ScrollArea>

              {/* 当前选中分组的表单 */}
              {settingsData.forms
                .filter((form) => form.group === settingsTab)
                .map((form) => (
                  <ScrollArea key={form.group} className="h-[55vh]">
                    <div className="space-y-4 pr-4">
                      {form.formSchema.map((field, idx) => (
                        <SettingFieldRenderer
                          key={field.name || idx}
                          field={field}
                          value={settingsValues[field.name]}
                          onChange={handleSettingChange}
                          allValues={settingsValues}
                        />
                      ))}
                    </div>
                  </ScrollArea>
                ))}

              {settingsError && (
                <div className="rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-xs text-destructive">
                  {settingsError}
                </div>
              )}

              <div className="flex justify-end gap-3 pt-4 border-t">
                <Button variant="outline" onClick={closeSettings}>
                  返回
                </Button>
                <Button
                  onClick={handleSaveSettings}
                  disabled={settingsSaving}
                >
                  {settingsSaving ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : null}
                  保存设置
                </Button>
              </div>
            </>
          ) : (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <Settings2 className="h-12 w-12 text-muted-foreground/30" />
              <p className="mt-4 text-sm text-muted-foreground">该主题没有可配置的设置项</p>
              <Button variant="outline" className="mt-4" onClick={closeSettings}>返回主题列表</Button>
            </div>
          )}
        </motion.div>
      ) : (
      <>
      {/* Local Themes */}
      {tab === "local" && (
        <div>
          {loading ? (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="rounded-xl border border-border/60 overflow-hidden">
                  <div className="aspect-video bg-muted animate-pulse" />
                  <div className="p-4 space-y-3">
                    <div className="h-4 w-2/3 rounded bg-muted animate-pulse" />
                    <div className="h-3 w-1/2 rounded bg-muted animate-pulse" />
                  </div>
                </div>
              ))}
            </div>
          ) : themes.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border/60 py-16 text-center">
              <PackageOpen className="h-12 w-12 text-muted-foreground/30" />
              <p className="mt-4 text-sm text-muted-foreground">还没有安装任何主题</p>
              <p className="mt-1 text-xs text-muted-foreground/60">上传一个主题包，或前往主题商店浏览</p>
            </div>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {themes.map((theme, i) => (
                <motion.div
                  key={theme.id}
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: i * 0.05 }}
                >
                  <Card className={`overflow-hidden border-border/60 transition-shadow hover:shadow-md ${
                    theme.active ? "ring-2 ring-primary ring-offset-2 ring-offset-background" : ""
                  }`}>
                    <div className="aspect-video bg-muted/50 relative overflow-hidden">
                      {theme.screenshot ? (
                        <img
                          src={API_BASE + theme.screenshot}
                          alt={theme.name}
                          className="h-full w-full object-cover"
                          onError={(e) => {
                            (e.target as HTMLImageElement).style.display = "none"
                          }}
                        />
                      ) : null}
                    </div>
                    <CardContent className="p-4">
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0 flex-1">
                          <h3 className="font-semibold text-sm truncate">{theme.name}</h3>
                          <p className="text-xs text-muted-foreground mt-0.5">
                            {theme.author && `${theme.author} · `}v{theme.version}
                          </p>
                        </div>
                        {theme.active && (
                          <CheckCircle2 className="h-5 w-5 shrink-0 text-primary" />
                        )}
                      </div>
                      {theme.description && (
                        <p className="text-xs text-muted-foreground/70 mt-2 line-clamp-2">{theme.description}</p>
                      )}
                      <div className="flex gap-2 mt-4">
                        {theme.active ? (
                          <>
                            <Button variant="outline" size="lg" disabled className="flex-1">
                              已激活
                            </Button>
                            {theme.hasSettings && (
                              <Button
                                variant="outline"
                                size="lg"
                                onClick={() => openSettings(theme)}
                                title="主题设置"
                              >
                                <Settings2 className="h-3.5 w-3.5" />
                              </Button>
                            )}
                          </>
                        ) : (
                          <Button
                            size="lg"
                            className="flex-1"
                            onClick={() => handleActivate(theme.id)}
                            disabled={activating === theme.id}
                          >
                            {activating === theme.id ? (
                              <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
                            ) : null}
                            激活
                          </Button>
                        )}
                        <Button
                          variant="outline"
                          size="lg"
                          onClick={() => handleDeleteClick(theme)}
                          disabled={theme.active}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                </motion.div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Remote Themes */}
      {tab === "remote" && (
        <div className="space-y-4">
          {/* Search */}
          <div className="flex gap-2">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索主题..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && loadRemote(0)}
                className="pl-9"
              />
            </div>
            <Button variant="outline" onClick={() => loadRemote(0)} disabled={remoteLoading}>
              {remoteLoading && <Loader2 className="h-4 w-4 mr-1 animate-spin" />}
              搜索
            </Button>
          </div>

          {remoteLoading ? (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {[1, 2, 3, 4, 5, 6].map((i) => (
                <div key={i} className="rounded-xl border border-border/60 overflow-hidden">
                  <div className="aspect-video bg-muted animate-pulse" />
                  <div className="p-4 space-y-3">
                    <div className="h-4 w-3/4 rounded bg-muted animate-pulse" />
                    <div className="h-3 w-1/2 rounded bg-muted animate-pulse" />
                    <div className="h-8 w-full rounded bg-muted animate-pulse mt-2" />
                  </div>
                </div>
              ))}
            </div>
          ) : remoteError ? (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border/60 py-16 text-center">
              <AlertCircle className="h-12 w-12 text-amber-500/60" />
              <p className="mt-4 text-sm text-destructive">{remoteError}</p>
              <p className="mt-1 text-xs text-muted-foreground/60">无法连接主题数据源，请先在本地安装主题</p>
            </div>
          ) : remoteThemes.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border/60 py-16 text-center">
              <Search className="h-12 w-12 text-muted-foreground/30" />
              <p className="mt-4 text-sm text-muted-foreground">未找到主题</p>
              <p className="mt-1 text-xs text-muted-foreground/60">尝试其他关键词搜索</p>
            </div>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {remoteThemes.map((theme, i) => (
                <motion.div
                  key={theme.id}
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: i * 0.03 }}
                >
                  <Card
                    className="overflow-hidden border-border/60 transition-shadow hover:shadow-md cursor-pointer group"
                    onClick={() => setDetailTheme(theme)}
                  >
                    <div className="aspect-video bg-muted/50 relative overflow-hidden">
                      {theme.screenshot ? (
                        <img
                          src={theme.screenshot}
                          alt={theme.name}
                          className="h-full w-full object-cover"
                          crossOrigin="anonymous"
                          onLoad={() => setImgLoaded(prev => new Set(prev).add(theme.id))}
                          onError={(e) => {
                            const img = e.target as HTMLImageElement
                            // 截图加载失败时回退到 GitHub OpenGraph 预览图
                            if (!img.dataset.fallbackTried) {
                              img.dataset.fallbackTried = "1"
                              const repoPath = theme.repoUrl?.replace(/^https?:\/\/github\.com\//, "")
                              if (repoPath) {
                                img.src = `https://opengraph.githubassets.com/1/${repoPath}`
                                return
                              }
                            }
                            img.style.display = "none"
                          }}
                        />
                      ) : null}
                      {/* Hover overlay */}
                      <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors flex items-center justify-center opacity-0 group-hover:opacity-100 pointer-events-none">
                        <span className="text-white text-sm font-medium bg-black/60 px-3 py-1.5 rounded-full">
                          查看详情
                        </span>
                      </div>
                    </div>
                    <CardContent className="p-4">
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0 flex-1">
                          <h3 className="font-semibold text-sm truncate">{theme.name}</h3>
                          <p className="text-xs text-muted-foreground mt-0.5">
                            {theme.author && `${theme.author} · `}v{theme.version}
                          </p>
                        </div>
                        {theme.installed && (
                          <CheckCircle2 className="h-5 w-5 shrink-0 text-emerald-500" />
                        )}
                      </div>
                      {theme.description && (
                        <p className="text-xs text-muted-foreground/70 mt-2 line-clamp-2">{theme.description}</p>
                      )}
                      <div className="flex gap-2 mt-4" onClick={(e) => e.stopPropagation()}>
                        {theme.installed || downloadProgresses[theme.id]?.status === "completed" ? (
                          <Button variant="outline" size="lg" disabled className="flex-1">
                            已拥有
                          </Button>
                        ) : downloadProgresses[theme.id]?.status === "error" ? (
                          <Button
                            variant="outline"
                            size="lg"
                            className="flex-1 text-destructive hover:text-destructive border-destructive/30 hover:border-destructive"
                            onClick={() => {
                              setDownloadProgresses((prev) => {
                                const next = { ...prev }
                                delete next[theme.id]
                                return next
                              })
                              handleDownload(theme.repoUrl)
                            }}
                          >
                            <Download className="h-3.5 w-3.5 mr-1" />
                            重试
                          </Button>
                        ) : downloadProgresses[theme.id] ? (
                          <div className="flex-1">
                            {(() => {
                              const dp = downloadProgresses[theme.id]
                              const pct = dp.pct || 0
                              const label = dp.status === "extracting" ? "解压中" : "下载中"
                              return (
                                <div className="space-y-1">
                                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                                    <span>{label}</span>
                                    <span>{pct}%</span>
                                  </div>
                                  <div className="h-2 rounded-full bg-muted overflow-hidden">
                                    <div
                                      className="h-full rounded-full transition-all duration-300 bg-primary"
                                      style={{ width: `${pct}%` }}
                                    />
                                  </div>
                                </div>
                              )
                            })()}
                          </div>
                        ) : (
                          <Button
                            variant="outline"
                            size="lg"
                            className="flex-1"
                            onClick={() => handleDownload(theme.repoUrl)}
                          >
                            <Download className="h-3.5 w-3.5 mr-1" />
                            获取
                          </Button>
                        )}
                        {theme.homepage && (
                          <Button variant="ghost" size="lg" asChild>
                            <a href={theme.homepage} target="_blank" rel="noopener noreferrer">
                              <ExternalLink className="h-3.5 w-3.5" />
                            </a>
                          </Button>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                </motion.div>
              ))}
            </div>
          )}

          {/* Pagination */}
          {!remoteLoading && !remoteError && remoteThemes.length > 0 && (
            <div className="flex items-center justify-center gap-4 pt-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => loadRemote(remotePage - 1)}
                disabled={remotePage <= 0}
              >
                <ChevronLeft className="h-4 w-4 mr-1" />
                上一页
              </Button>
              <span className="text-sm text-muted-foreground">
                第 {remotePage + 1} 页 / 共 {remoteTotalPages} 页
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => loadRemote(remotePage + 1)}
                disabled={remotePage + 1 >= remoteTotalPages}
              >
                下一页
                <ChevronRight className="h-4 w-4 ml-1" />
              </Button>
            </div>
          )}
        </div>
      )}
      </>
      )}

      {/* Delete Confirm Modal */}
      <Modal
        open={!!deleteTarget}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="确认删除"
        size="sm"
      >
        {deleteTarget && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              确定要删除主题 <span className="font-semibold text-foreground">{deleteTarget.name}</span> 吗？
              {deleteTarget.active && (
                <span className="block mt-1 text-amber-500">该主题为当前激活主题，删除后将无激活主题。</span>
              )}
            </p>
            <div className="flex justify-end gap-3">
              <Button variant="outline" onClick={() => setDeleteTarget(null)}>
                取消
              </Button>
              <Button variant="destructive" onClick={handleDeleteConfirm}>
                删除
              </Button>
            </div>
          </div>
        )}
      </Modal>

      {/* Detail Modal */}
      <Modal
        open={!!detailTheme}
        onOpenChange={(open) => { if (!open) setDetailTheme(null) }}
        title={detailTheme?.name || "主题详情"}
        size="lg"
      >
        {detailTheme && (
          <div className="space-y-5">
            {/* Screenshot */}
            <div className="aspect-video bg-muted/50 overflow-hidden rounded-lg border relative">
              {detailTheme.screenshot && (
                <img
                  src={detailTheme.repoUrl
                    ? `https://opengraph.githubassets.com/1/${detailTheme.repoUrl.replace(/^https?:\/\/github\.com\//, "")}`
                    : detailTheme.screenshot}
                  alt={detailTheme.name}
                  className="h-full w-full object-cover relative z-10"
                  onError={(e) => {
                    (e.target as HTMLImageElement).style.display = "none"
                  }}
                />
              )}
            </div>

            {/* Info */}
            <div className="space-y-3">
              <div>
                <p className="text-xs text-muted-foreground">作者</p>
                <p className="text-sm font-medium">{detailTheme.author || "未知"}</p>
              </div>
              {detailTheme.description && (
                <div>
                  <p className="text-xs text-muted-foreground">描述</p>
                  <p className="text-sm">{detailTheme.description}</p>
                </div>
              )}
              {detailTheme.version && (
                <div>
                  <p className="text-xs text-muted-foreground">版本</p>
                  <p className="text-sm font-mono">{detailTheme.version}</p>
                </div>
              )}
            </div>

            {/* Actions */}
            <div className="flex gap-3 pt-2">
              {detailTheme.repoUrl && (
                <Button asChild className="flex-1">
                  <a href={detailTheme.repoUrl} target="_blank" rel="noopener noreferrer">
                    <svg className="h-4 w-4 mr-2" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
                    </svg>
                    打开 GitHub
                  </a>
                </Button>
              )}
              {detailTheme.homepage && detailTheme.homepage !== detailTheme.repoUrl && (
                <Button variant="outline" asChild className="flex-1">
                  <a href={detailTheme.homepage} target="_blank" rel="noopener noreferrer">
                    <ExternalLink className="h-4 w-4 mr-2" />
                    预览演示
                  </a>
                </Button>
              )}
            </div>
          </div>
        )}
      </Modal>


    </div>
  )
}

// ─── Setting Field Renderer ───
// 根据 Halo settings.yaml 中的 $formkit 类型渲染对应的表单控件

interface SettingFieldRendererProps {
  field: ThemeSettingField
  value: any
  onChange: (name: string, value: any) => void
  allValues: Record<string, any>
  itemValues?: Record<string, any>
}

function SettingFieldRenderer({ field, value, onChange, allValues, itemValues }: SettingFieldRendererProps) {
  // 处理条件显示 if 表达式: "$get(fieldName).value === 'xxx'" 或 "$value.value === 'xxx'"
  if (field.if) {
    const visible = evalHaloIf(field.if, allValues, itemValues)
    if (!visible) return null
  }

  const currentValue = value !== undefined ? value : field.value
  const fieldId = `setting-${field.name}`

  switch (field.$formkit) {
    case "select":
      return (
        <div className="space-y-1.5">
          <Label className="text-sm">{field.label}</Label>
          <Select
            value={String(currentValue ?? "")}
            onValueChange={(v) => onChange(field.name, v)}
            placeholder={field.placeholder || `选择 ${field.label}`}
            options={field.options?.map(opt => ({ value: String(opt.value), label: opt.label })) ?? []}
          />
          {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
        </div>
      )

    case "radio":
      return (
        <div className="space-y-2">
          <Label className="text-sm">{field.label}</Label>
          <div className="flex flex-wrap gap-3">
            {field.options?.map((opt, i) => (
              <label key={i} className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name={field.name}
                  value={String(opt.value)}
                  checked={String(currentValue) === String(opt.value)}
                  onChange={() => onChange(field.name, opt.value)}
                  className="h-4 w-4 text-primary"
                />
                <span className="text-sm">{opt.label}</span>
              </label>
            ))}
          </div>
          {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
        </div>
      )

    case "switch":
      return (
        <div className="flex items-center justify-between gap-4">
          <div className="space-y-0.5">
            <Label className="text-sm">{field.label}</Label>
            {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
          </div>
          <Switch
            checked={currentValue === true || currentValue === "true"}
            onCheckedChange={(checked) => onChange(field.name, checked)}
          />
        </div>
      )

    case "color":
      return (
        <div className="space-y-1.5">
          <Label htmlFor={fieldId} className="text-sm">{field.label}</Label>
          <div className="flex gap-2 items-center">
            <input
              id={fieldId}
              type="color"
              value={String(currentValue ?? "#000000")}
              onChange={(e) => onChange(field.name, e.target.value)}
              className="h-9 w-12 rounded border cursor-pointer"
            />
            <Input
              value={String(currentValue ?? "")}
              onChange={(e) => onChange(field.name, e.target.value)}
              className="flex-1 font-mono text-sm"
            />
          </div>
          {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
        </div>
      )

    case "number":
      return (
        <div className="space-y-1.5">
          <Label htmlFor={fieldId} className="text-sm">{field.label}</Label>
          <Input
            id={fieldId}
            type="number"
            value={currentValue ?? ""}
            onChange={(e) => onChange(field.name, e.target.value === "" ? "" : Number(e.target.value))}
            min={field.min}
            max={field.max}
            step={field.step}
            placeholder={field.placeholder}
          />
          {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
        </div>
      )

    case "textarea":
      return (
        <div className="space-y-1.5">
          <Label htmlFor={fieldId} className="text-sm">{field.label}</Label>
          <Textarea
            id={fieldId}
            value={String(currentValue ?? "")}
            onChange={(e) => onChange(field.name, e.target.value)}
            rows={field.rows || 4}
            placeholder={field.placeholder}
          />
          {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
        </div>
      )

    case "code":
      return (
        <div className="space-y-1.5">
          <Label htmlFor={fieldId} className="text-sm">{field.label}</Label>
          <Textarea
            id={fieldId}
            value={String(currentValue ?? "")}
            onChange={(e) => onChange(field.name, e.target.value)}
            rows={field.rows || 6}
            placeholder={field.placeholder}
            className="font-mono text-xs"
          />
          {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
        </div>
      )

    case "attachment":
      return (
        <div className="space-y-1.5">
          <Label htmlFor={fieldId} className="text-sm">{field.label}</Label>
          <Input
            id={fieldId}
            type="text"
            value={String(currentValue ?? "")}
            onChange={(e) => onChange(field.name, e.target.value)}
            placeholder={field.placeholder || "输入附件 URL"}
          />
          {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
        </div>
      )

    case "group":
      return (
        <div className="space-y-3 pl-2 border-l-2 border-border/60">
          {field.label && <Label className="text-sm font-medium">{field.label}</Label>}
          {field.children?.map((child, idx) => (
            <SettingFieldRenderer
              key={child.name || idx}
              field={child}
              value={allValues[child.name]}
              onChange={onChange}
              allValues={allValues}
              itemValues={itemValues}
            />
          ))}
        </div>
      )

    case "array": {
      const items = Array.isArray(currentValue) ? [...currentValue] : []
      const defaultItem = () => {
        const item: Record<string, any> = {}
        field.children?.forEach((child) => {
          if (child.value !== undefined && child.value !== null) {
            item[child.name] = child.value
          }
        })
        return item
      }
      return (
        <div className="space-y-3">
          <Label className="text-sm">{field.label}</Label>
          {items.map((item, idx) => (
            <div key={idx} className="space-y-2 p-3 rounded-lg border border-border/60 bg-muted/30">
              <div className="flex items-center justify-between">
                <span className="text-xs text-muted-foreground">第 {idx + 1} 项</span>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onChange(field.name, items.filter((_, i) => i !== idx))}
                >
                  <Trash2 className="h-4 w-4 text-destructive" />
                </Button>
              </div>
              {field.children?.map((child, cidx) => (
                <SettingFieldRenderer
                  key={child.name || cidx}
                  field={child}
                  value={item[child.name]}
                  onChange={(name, val) => {
                    const next = [...items]
                    next[idx] = { ...next[idx], [name]: val }
                    onChange(field.name, next)
                  }}
                  allValues={allValues}
                  itemValues={item}
                />
              ))}
            </div>
          ))}
          <Button
            variant="outline"
            size="sm"
            onClick={() => onChange(field.name, [...items, defaultItem()])}
          >
            <Plus className="h-4 w-4 mr-1" />
            添加
          </Button>
          {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
        </div>
      )
    }

    // text 和未知类型默认使用文本输入
    default:
      return (
        <div className="space-y-1.5">
          <Label htmlFor={fieldId} className="text-sm">{field.label}</Label>
          <Input
            id={fieldId}
            type="text"
            value={String(currentValue ?? "")}
            onChange={(e) => onChange(field.name, e.target.value)}
            placeholder={field.placeholder}
          />
          {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
        </div>
      )
  }
}

// evaluateHaloIf 简化版 Halo if 条件表达式求值
// 支持: "$get(fieldName).value === 'constant'" 或 "$value.value === 'constant'"
function evalHaloIf(expr: string, values: Record<string, any>, itemValues?: Record<string, any>): boolean {
  try {
    // 匹配 $value.value === 'constant'
    const itemMatch = expr.match(/\$value\.value\s*[=!]==?\s*['"]?([^'"]+)['"]?/)
    if (itemMatch) {
      const expected = itemMatch[1]
      const actual = itemValues?.value
      return String(actual) === expected
    }
    // 匹配 $get(fieldName).value === 'constant' 或 $get(fieldName).value == 'constant'
    const match = expr.match(/\$get\((\w+)\)\.value\s*[=!]==?\s*['"]?([^'"]+)['"]?/)
    if (!match) return true
    const [, fieldName, expected] = match
    const actual = values[fieldName]
    return String(actual) === expected
  } catch {
    return true
  }
}
