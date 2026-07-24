import { useState } from "react"
import { Link, useSearchParams } from "react-router"
import { Button } from "../components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { api } from "../lib/api"
import { AnimatedPage, FadeIn } from "../components/animate"
import { toast } from "sonner"

export default function ForgotPassword() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get("token")

  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [loading, setLoading] = useState(false)
  const [sent, setSent] = useState(false)
  const [resetDone, setResetDone] = useState(false)
  const [error, setError] = useState("")

  const handleSendEmail = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")

    try {
      await api.forgotPassword(email)
      setSent(true)
      toast.success("如果该邮箱已注册，将收到重置密码邮件")
    } catch (err) {
      const msg = err instanceof Error ? err.message : "发送失败，请稍后重试"
      setError(msg)
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")

    if (password !== confirmPassword) {
      setError("两次输入的密码不一致")
      return
    }
    if (password.length < 6) {
      setError("密码长度不能少于 6 位")
      return
    }

    setLoading(true)
    try {
      await api.resetPassword(token!, password)
      setResetDone(true)
      toast.success("密码重置成功")
    } catch (err) {
      const msg = err instanceof Error ? err.message : "重置失败，请稍后重试"
      setError(msg)
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  if (resetDone) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-muted/50">
        <AnimatedPage>
          <Card className="w-full max-w-md">
            <CardHeader>
              <CardTitle className="text-2xl text-center">重置成功</CardTitle>
            </CardHeader>
            <CardContent className="text-center space-y-4">
              <p className="text-muted-foreground">您的密码已重置成功，请使用新密码登录。</p>
              <Link to="/login" className="inline-block text-sm text-primary hover:underline">
                前往登录
              </Link>
            </CardContent>
          </Card>
        </AnimatedPage>
      </div>
    )
  }

  if (sent) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-muted/50">
        <AnimatedPage>
          <Card className="w-full max-w-md">
            <CardHeader>
              <CardTitle className="text-2xl text-center">邮件已发送</CardTitle>
            </CardHeader>
            <CardContent className="text-center space-y-4">
              <p className="text-muted-foreground">
                如果该邮箱已注册，您将收到一封包含密码重置链接的邮件，请检查您的收件箱。
              </p>
              <Link to="/login" className="inline-block text-sm text-primary hover:underline">
                返回登录
              </Link>
            </CardContent>
          </Card>
        </AnimatedPage>
      </div>
    )
  }

  if (token) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-muted/50">
        <AnimatedPage className="w-full max-w-md">
          <Card className="w-full max-w-md">
            <CardHeader>
              <CardTitle className="text-2xl text-center">重置密码</CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleResetPassword} className="space-y-4">
                {error && (
                  <FadeIn>
                    <div className="bg-destructive/10 text-destructive text-sm p-3 rounded-md">{error}</div>
                  </FadeIn>
                )}
                <div className="space-y-2">
                  <Label htmlFor="password">新密码</Label>
                  <Input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    minLength={6}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="confirmPassword">确认新密码</Label>
                  <Input
                    id="confirmPassword"
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    required
                    minLength={6}
                  />
                </div>
                <Button type="submit" className="w-full" disabled={loading}>
                  {loading ? "提交中..." : "确认重置"}
                </Button>
              </form>
              <div className="mt-4 text-center text-sm text-muted-foreground">
                想起密码了？ <Link to="/login" className="text-primary hover:underline">返回登录</Link>
              </div>
            </CardContent>
          </Card>
        </AnimatedPage>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-muted/50">
      <AnimatedPage className="w-full max-w-md">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-2xl text-center">忘记密码</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSendEmail} className="space-y-4">
              {error && (
                <FadeIn>
                  <div className="bg-destructive/10 text-destructive text-sm p-3 rounded-md">{error}</div>
                </FadeIn>
              )}
              <div className="space-y-2">
                <Label htmlFor="email">注册邮箱</Label>
                <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
              </div>
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "发送中..." : "发送重置邮件"}
              </Button>
            </form>
            <div className="mt-4 text-center text-sm text-muted-foreground">
              想起密码了？ <Link to="/login" className="text-primary hover:underline">返回登录</Link>
            </div>
          </CardContent>
        </Card>
      </AnimatedPage>
    </div>
  )
}
