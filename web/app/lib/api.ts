const API_BASE = "/api/v1"

export interface ApiResponse<T> {
  code: number
  data: T
  msg: string
}

export interface User {
  id: number
  username: string
  nickname?: string
  email: string
  avatar?: string
  group?: string
  status: number
  createdAt: string
  updatedAt: string
}

export interface Article {
  id: number
  title: string
  slug: string
  summary?: string
  content: string
  coverUrl?: string
  status: "draft" | "published"
  commentEnabled?: boolean
  authorId: number
  categories?: Category[]
  tags?: Tag[]
  createdAt: string
  updatedAt: string
}

export interface Category {
  id: number
  name: string
  slug: string
  description?: string
  icon?: string
  createdAt: string
}

export interface Tag {
  id: number
  name: string
  slug: string
  icon?: string
  createdAt?: string
}

export interface Comment {
  id: number
  articleId: number
  parentId?: number
  userId?: number
  authorName: string
  authorEmail?: string
  authorUrl?: string
  content: string
  status: "approved" | "pending" | "rejected"
  ipAddress?: string
  createdAt: string
}

export interface Setting {
  key: string
  value: string
  description?: string
}

export interface Theme {
  id: string
  name: string
  version: string
  author: string
  description: string
  screenshot: string
  active: boolean
  hasSettings: boolean
  settingName?: string
  configMapName?: string
}

export interface RemoteTheme {
  id: string
  name: string
  version: string
  author: string
  description: string
  screenshot: string
  active: boolean
  installed: boolean
  repoUrl: string
  homepage: string
}

export interface DownloadProgress {
  repo: string
  pct: number
  status: string  // "downloading" | "extracting" | "completed" | "error"
  error?: string
  theme?: Theme
}

// ─── Theme Settings (Halo 格式) ───
export interface ThemeSettingField {
  $formkit: string
  name: string
  id?: string
  key?: string
  label: string
  value: any
  help?: string
  if?: string
  options?: { value: any; label: string }[]
  children?: ThemeSettingField[]
  min?: number
  max?: number
  step?: number
  rows?: number
  placeholder?: string
  validation?: string
  accept?: string
}

export interface ThemeSettingForm {
  group: string
  label: string
  formSchema: ThemeSettingField[]
}

export interface ThemeSettingsResponse {
  settingName: string
  forms: ThemeSettingForm[]
  values: Record<string, any>
}

class ApiClient {
  private token: string | null = null

  setToken(token: string) {
    this.token = token
    localStorage.setItem("auth_token", token)
  }

  getToken(): string | null {
    if (!this.token) {
      this.token = localStorage.getItem("auth_token")
    }
    return this.token
  }

  clearToken() {
    this.token = null
    localStorage.removeItem("auth_token")
  }

  private async request<T>(
    method: "GET" | "POST" | "PUT" | "DELETE",
    endpoint: string,
    body?: any
  ): Promise<ApiResponse<T>> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    }

    const token = this.getToken()
    if (token) {
      headers["Authorization"] = `Bearer ${token}`
    }

    const res = await fetch(`${API_BASE}${endpoint}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    })

    const data = await res.json()

    // 401: token 过期或无效，清除并跳转登录页
    if (res.status === 401) {
      this.clearToken()
      window.location.href = "/login"
      throw new Error("登录已过期，请重新登录")
    }

    // New unified response format: { code, data, msg }
    if (!res.ok && data.msg) {
      throw new Error(data.msg)
    }

    return data as ApiResponse<T>
  }

  async login(username: string, password: string) {
    return this.request<{ user: User; token: string }>("POST", "/login", {
      username,
      password,
    })
  }

  async register(username: string, nickname: string, email: string, password: string, passwordConfirm: string, code: string) {
    return this.request<{ user: User; token: string }>("POST", "/register", {
      username,
      nickname,
      email,
      password,
      password_confirm: passwordConfirm,
      code,
    })
  }

  async sendRegisterCode(email: string) {
    return this.request<{ message: string }>("POST", "/send-register-code", { email })
  }

  async forgotPassword(email: string) {
    return this.request<{ message: string }>("POST", "/forgot-password", { email })
  }

  async resetPassword(token: string, password: string) {
    return this.request<{ message: string }>("POST", "/reset-password", { token, password })
  }

  async getUsers() {
    return this.request<{ users: User[]; total: number }>("GET", "/admin/users")
  }

  async updateUserStatus(id: number, status: number) {
    return this.request<void>("PUT", `/admin/users/${id}/status?status=${status}`)
  }

  async deleteUser(id: number) {
    return this.request<void>("DELETE", `/admin/users/${id}`)
  }

  async getArticles() {
    return this.request<{ articles: Article[]; total: number }>("GET", "/admin/articles")
  }

  async getArticle(id: number) {
    return this.request<Article>("GET", `/articles/${id}`)
  }

  async createArticle(data: {
    title: string
    slug?: string
    summary?: string
    content?: string
    coverUrl?: string
    status?: Article["status"]
    commentEnabled?: boolean
    categoryIds?: number[]
    tagIds?: number[]
  }) {
    return this.request<Article>("POST", "/articles", data)
  }

  async updateArticle(
    id: number,
    data: {
      title?: string
      slug?: string
      summary?: string
      content?: string
      coverUrl?: string
      status?: Article["status"]
      commentEnabled?: boolean
      categoryIds?: number[]
      tagIds?: number[]
    }
  ){
    return this.request<Article>("PUT", `/articles/${id}`, data)
  }

  async deleteArticle(id: number) {
    return this.request<void>("DELETE", `/articles/${id}`)
  }

  async getCategories() {
    return this.request<{ categories: Category[] }>("GET", "/categories")
  }

  async createCategory(data: Partial<Category>) {
    return this.request<Category>("POST", "/admin/categories", data)
  }

  async updateCategory(id: number, data: Partial<Category>) {
    return this.request<Category>("PUT", `/admin/categories/${id}`, data)
  }

  async deleteCategory(id: number) {
    return this.request<void>("DELETE", `/admin/categories/${id}`)
  }

  async getTags() {
    return this.request<{ tags: Tag[] }>("GET", "/tags")
  }

  async createTag(data: Partial<Tag>) {
    return this.request<Tag>("POST", "/admin/tags", data)
  }

  async updateTag(id: number, data: Partial<Tag>) {
    return this.request<Tag>("PUT", `/admin/tags/${id}`, data)
  }

  async deleteTag(id: number) {
    return this.request<void>("DELETE", `/admin/tags/${id}`)
  }

  // ─── Comments ───
  async getComments(articleId: number, offset: number = 0, limit: number = 20) {
    return this.request<{ items: Comment[]; total: number; offset: number; limit: number }>(
      "GET",
      `/articles/${articleId}/comments?offset=${offset}&limit=${limit}`
    )
  }

  async createComment(data: {
    article_id: number
    content: string
    parent_id?: number
    author_name?: string
    author_email?: string
    author_url?: string
  }) {
    return this.request<Comment>("POST", "/comments", data)
  }

  async getAdminComments(status: string = "", offset: number = 0, limit: number = 20) {
    const statusParam = status ? `&status=${encodeURIComponent(status)}` : ""
    return this.request<{ items: Comment[]; total: number; offset: number; limit: number }>(
      "GET",
      `/admin/comments?offset=${offset}&limit=${limit}${statusParam}`
    )
  }

  async updateCommentStatus(id: number, status: "approved" | "rejected") {
    return this.request<void>("PUT", `/admin/comments/${id}/status`, { status })
  }

  async deleteComment(id: number) {
    return this.request<void>("DELETE", `/comments/${id}`)
  }

  async getSettings() {
    return this.request<{ settings: Setting[] }>("GET", "/settings")
  }

  async updateSetting(key: string, value: string) {
    return this.request<void>("PUT", `/admin/settings/${key}`, { key, value })
  }

  async uploadIcon(file: File): Promise<ApiResponse<{ icon_url: string }>> {
    const formData = new FormData()
    formData.append('icon', file)

    const token = this.getToken()
    const headers: Record<string, string> = {}
    if (token) {
      headers["Authorization"] = `Bearer ${token}`
    }

    const res = await fetch(`${API_BASE}/admin/settings/icon`, {
      method: 'POST',
      headers,
      body: formData,
    })

    const data = await res.json()

    if (res.status === 401) {
      this.clearToken()
      window.location.href = "/login"
      throw new Error("登录已过期，请重新登录")
    }

    if (!res.ok && data.msg) {
      throw new Error(data.msg)
    }

    return data as ApiResponse<{ icon_url: string }>
  }

  async uploadImage(file: File): Promise<ApiResponse<{ url: string }>> {
    const formData = new FormData()
    formData.append('image', file)

    const token = this.getToken()
    const headers: Record<string, string> = {}
    if (token) {
      headers["Authorization"] = `Bearer ${token}`
    }

    const res = await fetch(`${API_BASE}/admin/upload/image`, {
      method: 'POST',
      headers,
      body: formData,
    })

    const data = await res.json()

    if (res.status === 401) {
      this.clearToken()
      window.location.href = "/login"
      throw new Error("登录已过期，请重新登录")
    }

    if (!res.ok && data.msg) {
      throw new Error(data.msg)
    }

    return data as ApiResponse<{ url: string }>
  }

  // ─── Themes ───
  async getThemes() {
    return this.request<{ themes: Theme[] }>("GET", "/admin/themes")
  }

  async uploadTheme(file: File): Promise<ApiResponse<Theme>> {
    const formData = new FormData()
    formData.append('theme', file)

    const token = this.getToken()
    const headers: Record<string, string> = {}
    if (token) {
      headers["Authorization"] = `Bearer ${token}`
    }

    const res = await fetch(`${API_BASE}/admin/themes/upload`, {
      method: 'POST',
      headers,
      body: formData,
    })

    const data = await res.json()

    if (res.status === 401) {
      this.clearToken()
      window.location.href = "/login"
      throw new Error("登录已过期，请重新登录")
    }

    if (!res.ok && data.msg) {
      throw new Error(data.msg)
    }

    return data as ApiResponse<Theme>
  }

  async activateTheme(id: string) {
    return this.request<void>("POST", `/admin/themes/${id}/activate`)
  }

  async deleteTheme(id: string) {
    return this.request<void>("DELETE", `/admin/themes/${id}`)
  }

  async downloadTheme(repoUrl: string): Promise<ApiResponse<{ repo: string }>> {
    return this.request<{ repo: string }>("POST", "/admin/themes/download", { repoUrl })
  }

  async getDownloadProgress(repos: string): Promise<ApiResponse<{ progresses: DownloadProgress[] }>> {
    return this.request<{ progresses: DownloadProgress[] }>(
      "GET",
      `/admin/themes/download/progress?repos=${encodeURIComponent(repos)}`
    )
  }

  async getRemoteThemes(keyword: string = "", page: number = 0, size: number = 20) {
    return this.request<{ themes: RemoteTheme[]; total: number }>(
      "GET",
      `/admin/themes/remote?keyword=${encodeURIComponent(keyword)}&page=${page}&size=${size}`
    )
  }

  // ─── Theme Settings ───
  async getThemeSettings(id: string) {
    return this.request<ThemeSettingsResponse>("GET", `/admin/themes/${id}/settings`)
  }

  async updateThemeSettings(id: string, values: Record<string, any>) {
    return this.request<void>("PUT", `/admin/themes/${id}/settings`, { values })
  }

  // 获取当前激活主题（公开接口）
  async getActiveTheme() {
    return this.request<{ theme: Theme }>("GET", `/theme`)
  }

  // ─── Public API ───
  async getPublicArticles(page: number = 1, size: number = 10) {
    return this.request<{ items: any[]; total: number }>("GET", `/articles?page=${page}&size=${size}`)
  }

  async getPublicArticle(slug: string) {
    return this.request<any>("GET", `/articles/${slug}`)
  }

  async getPublicCategories() {
    return this.request<any[]>("GET", `/categories`)
  }

  async getPublicTags() {
    return this.request<any[]>("GET", `/tags`)
  }

  async getPublicArticlesByCategory(slug: string, page: number = 1, size: number = 10) {
    return this.request<{ items: any[]; total: number }>("GET", `/articles?category=${slug}&page=${page}&size=${size}`)
  }

  async getPublicArticlesByTag(slug: string, page: number = 1, size: number = 10) {
    return this.request<{ items: any[]; total: number }>("GET", `/articles?tag=${slug}&page=${page}&size=${size}`)
  }

  async getPublicPage(slug: string) {
    return this.request<any>("GET", `/page/${slug}`)
  }

  async searchArticles(keyword: string, page: number = 1, size: number = 10) {
    return this.request<{ items: any[]; total: number; page: number; size: number }>(
      "GET",
      `/search?keyword=${encodeURIComponent(keyword)}&page=${page}&size=${size}`
    )
  }
}

export const api = new ApiClient()
