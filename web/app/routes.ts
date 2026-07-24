import { type RouteConfig, index, route, layout } from "@react-router/dev/routes"

export default [
  // 认证页
  route("login", "routes/login.tsx"),
  route("register", "routes/register.tsx"),
  route("forgot-password", "routes/forgot-password.tsx"),

  // 管理后台
  layout("components/auth-guard.tsx", [
    route("admin", "routes/admin.tsx", [
      index("routes/admin._index.tsx"),
      route("users", "routes/admin.users.tsx"),
      route("articles", "routes/admin.articles.tsx"),
      route("comments", "routes/admin.comments.tsx"),
      route("categories", "routes/admin.categories.tsx"),
      route("settings", "routes/admin.settings.tsx"),
      route("themes", "routes/admin.themes.tsx"),
    ]),
  ]),
] satisfies RouteConfig
