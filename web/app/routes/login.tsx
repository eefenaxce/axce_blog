import { useState } from "react"
import { useNavigate, Link } from "react-router"
import { Button } from "../components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { api } from "../lib/api"
import { AnimatedPage, FadeIn } from "../components/animate"
import { ThemeToggle } from "~/components/theme-toggle"
import { toast } from "sonner"

export default function Login() {
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")

    try {
      const res = await api.login(username, password)
      api.setToken(res.data.token)
      localStorage.setItem("user", JSON.stringify(res.data.user))
      if (res.data.user.group === "admin") {
        navigate("/admin")
      } else {
        navigate("/")
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : "登录失败，请稍后重试"
      setError(msg)
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-muted/50">
      <div className="fixed right-4 top-4">
        <ThemeToggle />
      </div>
      <AnimatedPage className="w-full max-w-md">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-2xl text-center">管理后台登录</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <FadeIn>
                  <div className="bg-destructive/10 text-destructive text-sm p-3 rounded-md">
                    {error}
                  </div>
                </FadeIn>
              )}
              <div className="space-y-2">
                <Label htmlFor="username">用户名</Label>
                <Input
                  id="username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">密码</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </div>
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "登录中..." : "登录"}
              </Button>
            </form>
            <div className="mt-4 flex flex-col items-center gap-2 text-sm text-muted-foreground">
              <Link to="/forgot-password" className="text-primary hover:underline">
                忘记密码？
              </Link>
              <span>
                还没有账号？{" "}
                <Link to="/register" className="text-primary hover:underline">
                  立即注册
                </Link>
              </span>
            </div>
          </CardContent>
        </Card>
      </AnimatedPage>
    </div>
  )
}
